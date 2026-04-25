package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	unsetEnv(t, "INBOX_DIR", "BASE_FILE_URL", "GET_TASK_URL", "POST_RESULT_URL", "API_KEY", "CONCURRENCY", "TESSERACT_LANG")

	cfg := Load()

	if cfg.BaseFileURL != "https://failai.viespirkiai.org/" {
		t.Fatalf("unexpected BaseFileURL: %q", cfg.BaseFileURL)
	}
	if cfg.NextURL != "https://viespirkiai.org/failas/ocr/checkout" {
		t.Fatalf("unexpected NextURL: %q", cfg.NextURL)
	}
	if cfg.ResultURL != "https://viespirkiai.org/failas/ocr/submit" {
		t.Fatalf("unexpected ResultURL: %q", cfg.ResultURL)
	}
	if cfg.APIKey != "" {
		t.Fatalf("unexpected APIKey: %q", cfg.APIKey)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("API_KEY", "secret")
	t.Setenv("GET_TASK_URL", "https://example.com/checkout")
	t.Setenv("POST_RESULT_URL", "https://example.com/submit")
	t.Setenv("BASE_FILE_URL", "https://example.com/file/")
	t.Setenv("INBOX_DIR", "/tmp/inbox")
	t.Setenv("CONCURRENCY", "7")
	t.Setenv("TESSERACT_LANG", "eng")

	cfg := Load()

	if cfg.APIKey != "secret" {
		t.Fatalf("unexpected APIKey: %q", cfg.APIKey)
	}
	if cfg.NextURL != "https://example.com/checkout" {
		t.Fatalf("unexpected NextURL: %q", cfg.NextURL)
	}
	if cfg.ResultURL != "https://example.com/submit" {
		t.Fatalf("unexpected ResultURL: %q", cfg.ResultURL)
	}
	if cfg.BaseFileURL != "https://example.com/file/" {
		t.Fatalf("unexpected BaseFileURL: %q", cfg.BaseFileURL)
	}
	if cfg.InboxDir != "/tmp/inbox" {
		t.Fatalf("unexpected InboxDir: %q", cfg.InboxDir)
	}
	if cfg.Concurrency != 7 {
		t.Fatalf("unexpected Concurrency: %d", cfg.Concurrency)
	}
	if cfg.TesseractLang != "eng" {
		t.Fatalf("unexpected TesseractLang: %q", cfg.TesseractLang)
	}
}

func unsetEnv(t *testing.T, keys ...string) {
	t.Helper()

	for _, key := range keys {
		value, ok := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
		t.Cleanup(func() {
			if !ok {
				_ = os.Unsetenv(key)
				return
			}
			_ = os.Setenv(key, value)
		})
	}
}
