package config

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	InboxDir          string
	BaseFileURL       string
	NextURL           string
	ResultURL         string
	Concurrency       int
	TesseractLang     string
	ExtraResultFields map[string]any
}

func Load() Config {
	slog.Debug("loading config", "env", os.Environ())
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found")
	}
	getEnv := func(k, def string) string {
		if v, ok := os.LookupEnv(k); ok {
			return v
		}
		return def
	}
	atoi := func(s string, def int) int {
		if v := os.Getenv(s); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				return n
			}
		}
		return def
	}
	// parseBool := func(s string, def bool) bool {
	// 	if v := os.Getenv(s); v != "" {
	// 		switch strings.ToLower(v) {
	// 		case "1", "true", "yes", "y", "on":
	// 			return true
	// 		case "0", "false", "no", "n", "off":
	// 			return false
	// 		}
	// 	}
	// 	return def
	// }

	return Config{
		InboxDir:      getEnv("INBOX_DIR", "./inbox"),
		BaseFileURL:   getEnv("BASE_FILE_URL", "http://localhost:8080/file/"),
		NextURL:       getEnv("GET_TASK_URL", "http://localhost:8080/next"),
		ResultURL:     getEnv("POST_RESULT_URL", "http://localhost:8080/result"),
		Concurrency:   atoi("CONCURRENCY", 4),
		TesseractLang: getEnv("TESSERACT_LANG", "lit+eng"),
		ExtraResultFields: map[string]any{
			"source": "golang-worker",
		},
	}
}
