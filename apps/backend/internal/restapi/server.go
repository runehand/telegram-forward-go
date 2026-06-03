package restapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"zenfl-forwarder/apps/backend/internal/config"
	"zenfl-forwarder/apps/backend/internal/domain"
	"zenfl-forwarder/apps/backend/internal/store/mongo"
)

type Server struct {
	cfg   config.Config
	log   *zap.Logger
	store *mongo.Store
	http  *http.Server
}

type authClaims struct {
	UserID string          `json:"uid"`
	Email  string          `json:"email"`
	Role   domain.UserRole `json:"role"`
	jwt.RegisteredClaims
}

type ctxUser struct{}

func New(cfg config.Config, log *zap.Logger, store *mongo.Store) *Server {
	s := &Server{cfg: cfg, log: log, store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/auth/login", s.handleLogin)
	mux.Handle("/api/auth/me", s.auth(http.HandlerFunc(s.handleMe)))
	mux.Handle("/api/jobs", s.auth(http.HandlerFunc(s.handleJobs)))
	mux.Handle("/api/jobs/", s.auth(http.HandlerFunc(s.handleJobActions)))
	mux.Handle("/api/admin/users", s.auth(s.requireAdmin(http.HandlerFunc(s.handleAdminUsers))))
	mux.Handle("/api/admin/users/", s.auth(s.requireAdmin(http.HandlerFunc(s.handleAdminUserByID))))

	s.http = &http.Server{
		Addr:              cfg.App.HTTPAddr,
		Handler:           s.cors(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s
}

func (s *Server) Run(ctx context.Context) error {
	_ = s.seedUsers(ctx)

	errCh := make(chan error, 1)
	go func() {
		s.log.Info("rest api listening", zap.String("addr", s.cfg.App.HTTPAddr))
		errCh <- s.http.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.http.Shutdown(shutdownCtx)
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

func (s *Server) seedUsers(ctx context.Context) error {
	adminHash, _ := bcrypt.GenerateFromPassword([]byte("admin1234"), bcrypt.DefaultCost)
	userHash, _ := bcrypt.GenerateFromPassword([]byte("demo1234"), bcrypt.DefaultCost)

	if err := s.store.EnsureUser(ctx, mongo.UserUpsertInput{
		Name:         "Platform Admin",
		Email:        "admin@zenfl.local",
		Role:         domain.RoleAdmin,
		PasswordHash: string(adminHash),
		Preferences: domain.UserPreferences{
			OnlyUnseen: true,
			OnlyUS:     false,
			Hours:      24,
		},
	}); err != nil {
		return err
	}
	return s.store.EnsureUser(ctx, mongo.UserUpsertInput{
		Name:         "Demo User",
		Email:        "demo@zenfl.local",
		Role:         domain.RoleNormal,
		PasswordHash: string(userHash),
		Preferences: domain.UserPreferences{
			OnlyUnseen: true,
			OnlyUS:     true,
			Hours:      24,
		},
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	u, err := s.store.FindUserByEmail(r.Context(), body.Email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(body.Password)) != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	tokenString, err := s.signToken(u)
	if err != nil {
		http.Error(w, "token error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token": tokenString,
		"user":  sanitizeUser(u),
	})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"user": sanitizeUser(u)})
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	u := userFromContext(r.Context())
	q := parseJobQuery(r, u.Preferences)
	items, err := s.store.ListJobsForUser(r.Context(), u, q)
	if err != nil {
		http.Error(w, "failed to fetch jobs", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"query": q,
	})
}

func (s *Server) handleJobActions(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r.Context())
	path := strings.TrimPrefix(r.URL.Path, "/api/jobs/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	jobID := parts[0]

	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		job, err := s.store.GetJob(r.Context(), jobID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = s.store.MarkJobSeen(r.Context(), u.ID, jobID)
		writeJSON(w, http.StatusOK, map[string]any{"item": job})
	case len(parts) == 2 && parts[1] == "seen" && r.Method == http.MethodPost:
		if err := s.store.MarkJobSeen(r.Context(), u.ID, jobID); err != nil {
			http.Error(w, "failed to mark seen", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		users, err := s.store.ListUsers(r.Context())
		if err != nil {
			http.Error(w, "failed to list users", http.StatusInternalServerError)
			return
		}
		out := make([]domain.User, 0, len(users))
		for _, user := range users {
			out = append(out, sanitizeUser(user))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": out})
	case http.MethodPost:
		var body struct {
			Name     string                 `json:"name"`
			Email    string                 `json:"email"`
			Password string                 `json:"password"`
			Role     domain.UserRole        `json:"role"`
			Prefs    domain.UserPreferences `json:"preferences"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "failed to hash password", http.StatusInternalServerError)
			return
		}
		if err := s.store.EnsureUser(r.Context(), mongo.UserUpsertInput{
			Name:         body.Name,
			Email:        body.Email,
			Role:         body.Role,
			PasswordHash: string(hash),
			Preferences:  body.Prefs,
		}); err != nil {
			http.Error(w, "failed to save user", http.StatusInternalServerError)
			return
		}
		user, err := s.store.FindUserByEmail(r.Context(), body.Email)
		if err != nil {
			http.Error(w, "failed to fetch user", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"item": sanitizeUser(user)})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAdminUserByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	var body struct {
		Role        domain.UserRole         `json:"role"`
		Password    string                  `json:"password"`
		Preferences *domain.UserPreferences `json:"preferences"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	passwordHash := ""
	if strings.TrimSpace(body.Password) != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "failed to hash password", http.StatusInternalServerError)
			return
		}
		passwordHash = string(hash)
	}
	if err := s.store.UpdateUser(r.Context(), id, body.Role, body.Preferences, passwordHash); err != nil {
		http.Error(w, "failed to update user", http.StatusInternalServerError)
		return
	}
	user, err := s.store.FindUserByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": sanitizeUser(user)})
}

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &authClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(s.cfg.Auth.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		user, err := s.store.FindUserByID(r.Context(), claims.UserID)
		if err != nil {
			http.Error(w, "invalid user", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxUser{}, user)))
	})
}

func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFromContext(r.Context())
		if u.Role != domain.RoleAdmin {
			http.Error(w, "admin access required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) signToken(u domain.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, authClaims{
		UserID: u.ID,
		Email:  u.Email,
		Role:   u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	})
	return token.SignedString([]byte(s.cfg.Auth.JWTSecret))
}

func parseJobQuery(r *http.Request, prefs domain.UserPreferences) domain.JobQuery {
	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	hours := prefs.Hours
	if q := strings.TrimSpace(r.URL.Query().Get("hours")); q != "" {
		if parsed, err := strconv.Atoi(q); err == nil {
			hours = parsed
		}
	}
	query := domain.JobQuery{
		Limit:      limit,
		Query:      strings.TrimSpace(r.URL.Query().Get("q")),
		OnlyUnseen: prefs.OnlyUnseen,
		OnlyUS:     prefs.OnlyUS,
		OnlyMobile: prefs.OnlyMobile,
		Country:    strings.TrimSpace(r.URL.Query().Get("country")),
		Tag:        strings.TrimSpace(r.URL.Query().Get("tag")),
		Hours:      hours,
	}
	if query.Country == "" {
		query.Country = prefs.Country
	}
	if v := strings.TrimSpace(r.URL.Query().Get("unseen")); v != "" {
		query.OnlyUnseen = v == "true"
	}
	if v := strings.TrimSpace(r.URL.Query().Get("onlyUS")); v != "" {
		query.OnlyUS = v == "true"
	}
	if v := strings.TrimSpace(r.URL.Query().Get("onlyMobile")); v != "" {
		query.OnlyMobile = v == "true"
	}
	if v := strings.TrimSpace(r.URL.Query().Get("verified")); v != "" {
		parsed := v == "true"
		query.Verified = &parsed
	}
	return query
}

func sanitizeUser(u domain.User) domain.User {
	u.PasswordHash = ""
	return u
}

func userFromContext(ctx context.Context) domain.User {
	if u, ok := ctx.Value(ctxUser{}).(domain.User); ok {
		return u
	}
	return domain.User{}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
