package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	InboxDir          string
	BaseFileURL       string
	NextURL           string
	ResultURL         string
	APIKey            string
	Concurrency       int
	TesseractLang     string
	ExtraResultFields map[string]any
}

func Load() Config {
	if err := godotenv.Load(); err != nil {
		// Environment variables can also be provided externally, e.g. via Docker.
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
		BaseFileURL:   getEnv("BASE_FILE_URL", "https://failai.viespirkiai.org/"),
		NextURL:       getEnv("GET_TASK_URL", "https://viespirkiai.org/failas/ocr/checkout"),
		ResultURL:     getEnv("POST_RESULT_URL", "https://viespirkiai.org/failas/ocr/submit"),
		APIKey:        getEnv("API_KEY", ""),
		Concurrency:   atoi("CONCURRENCY", 4),
		TesseractLang: getEnv("TESSERACT_LANG", "lit+eng"),
		ExtraResultFields: map[string]any{
			"source": "golang-worker",
		},
	}
}
