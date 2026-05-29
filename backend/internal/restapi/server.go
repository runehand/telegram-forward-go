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

	"zenfl-forwarder/backend/internal/config"
	"zenfl-forwarder/backend/internal/store/mongo"
)

type Server struct {
	cfg    config.Config
	log    *zap.Logger
	store  *mongo.Store
	http   *http.Server
}

func New(cfg config.Config, log *zap.Logger, store *mongo.Store) *Server {
	s := &Server{cfg: cfg, log: log, store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/auth/login", s.handleLogin)
	mux.Handle("/api/auth/me", s.auth(http.HandlerFunc(s.handleMe)))
	mux.Handle("/api/jobs", s.auth(http.HandlerFunc(s.handleJobs)))

	s.http = &http.Server{Addr: cfg.App.HTTPAddr, Handler: s.cors(mux), ReadHeaderTimeout: 5 * time.Second}
	return s
}

func (s *Server) Run(ctx context.Context) error {
	hash, _ := bcrypt.GenerateFromPassword([]byte("demo1234"), bcrypt.DefaultCost)
	_ = s.store.EnsureDemoUser(ctx, "demo@zenfl.local", string(hash))

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

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
	var body struct{ Email, Password string }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { http.Error(w, "invalid body", http.StatusBadRequest); return }

	u, err := s.store.FindUserByEmail(r.Context(), strings.ToLower(strings.TrimSpace(body.Email)))
	if err != nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(body.Password)) != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized); return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": u.Email, "exp": time.Now().Add(24 * time.Hour).Unix()})
	t, err := token.SignedString([]byte(s.cfg.Auth.JWTSecret))
	if err != nil { http.Error(w, "token error", http.StatusInternalServerError); return }
	writeJSON(w, http.StatusOK, map[string]string{"token": t})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	email, _ := r.Context().Value(ctxUserEmail{}).(string)
	writeJSON(w, http.StatusOK, map[string]string{"email": email})
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	items, err := s.store.ListMessages(r.Context(), limit)
	if err != nil { http.Error(w, "failed to fetch jobs", http.StatusInternalServerError); return }
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type ctxUserEmail struct{}

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") { http.Error(w, "missing token", http.StatusUnauthorized); return }
		tok := strings.TrimPrefix(h, "Bearer ")
		parsed, err := jwt.Parse(tok, func(token *jwt.Token) (interface{}, error) { return []byte(s.cfg.Auth.JWTSecret), nil })
		if err != nil || !parsed.Valid { http.Error(w, "invalid token", http.StatusUnauthorized); return }
		claims, _ := parsed.Claims.(jwt.MapClaims)
		email, _ := claims["sub"].(string)
		ctx := context.WithValue(r.Context(), ctxUserEmail{}, email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
		next.ServeHTTP(w, r)
	})
}
