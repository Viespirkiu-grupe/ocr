package config

import (
	"os"
	"strconv"
)

type Config struct {
	BaseFileURL       string
	NextURL           string
	ResultURL         string
	Concurrency       int
	TesseractLang     string
	ExtraResultFields map[string]any
}

func Load() Config {
	getEnv := func(k, def string) string {
		if v := os.Getenv(k); v != "" {
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
