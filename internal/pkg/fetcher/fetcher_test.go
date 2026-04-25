package fetcher

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viespirkiu-grupe/ocr/internal/model"
)

func TestTaskSendsBearerToken(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("unexpected accept header: %q", got)
		}
		if err := json.NewEncoder(w).Encode(model.Task{ID: 42, Uri: "/doc.pdf"}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer ts.Close()

	task, err := Task(context.Background(), ts.URL, "secret")
	if err != nil {
		t.Fatalf("Task returned error: %v", err)
	}
	if task.ID != 42 {
		t.Fatalf("unexpected task id: %d", task.ID)
	}
}

func TestResultsSendsBearerToken(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("unexpected content type: %q", got)
		}
		var result model.Response
		if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if result.ID != 42 {
			t.Fatalf("unexpected result id: %d", result.ID)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	err := Results(context.Background(), ts.URL, model.Response{
		ID:       42,
		Text:     []string{"page one"},
		Duration: 100,
	}, "secret")
	if err != nil {
		t.Fatalf("Results returned error: %v", err)
	}
}
