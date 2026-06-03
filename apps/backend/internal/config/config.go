package config

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	App      AppConfig
	Telegram TelegramConfig
	Mongo    MongoConfig
	Auth     AuthConfig
}

type AppConfig struct {
	HTTPAddr string
}

type MongoConfig struct {
	URI        string
	Database   string
	Collection string
	UsersColl  string
	SeenColl   string
}

type AuthConfig struct {
	JWTSecret string
}

type TelegramConfig struct {
	APIID           int
	APIHash         string
	Phone           string
	SessionFile     string
	ZenflUsername   string
	TargetUsernames []string
}

func LoadFromEnv() (Config, error) {
	_ = loadDotEnv(".env")

	cfg := Config{
		App: AppConfig{HTTPAddr: envOrDefault("APP_HTTP_ADDR", ":8080")},
		Mongo: MongoConfig{
			URI:        envOrDefault("MONGO_URI", "mongodb://localhost:27017"),
			Database:   envOrDefault("MONGO_DB", "zenfl"),
			Collection: envOrDefault("MONGO_MESSAGES_COLLECTION", "job_messages"),
			UsersColl:  envOrDefault("MONGO_USERS_COLLECTION", "users"),
			SeenColl:   envOrDefault("MONGO_SEEN_COLLECTION", "user_seen_jobs"),
		},
		Auth: AuthConfig{JWTSecret: envOrDefault("AUTH_JWT_SECRET", "change-me")},
	}

	apiID, err := strconv.Atoi(strings.TrimSpace(os.Getenv("TG_API_ID")))
	if err != nil || apiID == 0 {
		return Config{}, errors.New("TG_API_ID is required and must be an integer")
	}

	apiHash := strings.TrimSpace(os.Getenv("TG_API_HASH"))
	if apiHash == "" {
		return Config{}, errors.New("TG_API_HASH is required")
	}

	phone := strings.TrimSpace(os.Getenv("TG_PHONE"))
	if phone == "" {
		return Config{}, errors.New("TG_PHONE is required")
	}

	zenfl := normalizeUsername(os.Getenv("ZENFL_USERNAME"))
	if zenfl == "" {
		return Config{}, errors.New("ZENFL_USERNAME is required")
	}

	targetsRaw := strings.TrimSpace(os.Getenv("TARGET_USERNAMES"))
	if targetsRaw == "" {
		return Config{}, errors.New("TARGET_USERNAMES is required (comma-separated @usernames)")
	}
	parts := strings.Split(targetsRaw, ",")
	targets := make([]string, 0, len(parts))
	for _, p := range parts {
		u := normalizeUsername(p)
		if u != "" {
			targets = append(targets, u)
		}
	}
	if len(targets) == 0 {
		return Config{}, errors.New("TARGET_USERNAMES has no valid usernames")
	}

	sessionFile := strings.TrimSpace(os.Getenv("TG_SESSION_FILE"))
	if sessionFile == "" {
		sessionFile = "session.json"
	}

	cfg.Telegram = TelegramConfig{
		APIID:           apiID,
		APIHash:         apiHash,
		Phone:           phone,
		SessionFile:     sessionFile,
		ZenflUsername:   zenfl,
		TargetUsernames: targets,
	}

	return cfg, nil
}

func envOrDefault(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func normalizeUsername(v string) string {
	s := strings.TrimSpace(v)
	s = strings.TrimPrefix(s, "https://t.me/")
	s = strings.TrimPrefix(s, "t.me/")
	s = strings.TrimPrefix(s, "@")
	return s
}

func loadDotEnv(path string) error {
	abs, err := findUpward(path)
	if err != nil {
		return err
	}
	f, err := os.Open(abs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, "\"")
		val = strings.Trim(val, "'")
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, val)
	}
	return scanner.Err()
}

func findUpward(name string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return filepath.Abs(name)
		}
		dir = parent
	}
}
