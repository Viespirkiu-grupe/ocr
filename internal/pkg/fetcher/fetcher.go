package fetcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Viespirkiu-grupe/ocr/internal/model"
)

func File(ctx context.Context, url string, path string) error {
	slog.Info("fetching file", "url", url, "path", path)
	defer func() {
		slog.Info("fetched file", "url", url, "path", path)
	}()
	hc := &http.Client{
		Timeout: 120 * time.Second,
	}

	resp, err := hc.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch file: %s", resp.Status)
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	// bar := progressbar.NewOptions64(
	// 	resp.ContentLength,
	// 	progressbar.OptionSetDescription("downloading"),
	// 	progressbar.OptionShowBytes(true),
	// 	progressbar.OptionSetWidth(20),
	// 	progressbar.OptionThrottle(100*time.Millisecond),
	// 	progressbar.OptionShowCount(),
	// 	progressbar.OptionClearOnFinish(),
	// )

	// _, err = io.Copy(io.MultiWriter(out, bar), resp.Body)
	_, err = io.Copy(out, resp.Body)

	return err
}

func Task(ctx context.Context, url string, apiKey string) (model.Task, error) {
	slog.Info("fetching task")
	defer func() {
		slog.Info("fetched task", "url", url)
	}()
	hc := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return model.Task{}, err
	}
	req.Header.Set("Accept", "application/json")
	setBearerAuth(req, apiKey)

	resp, err := hc.Do(req)
	if err != nil {
		return model.Task{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return model.Task{}, fmt.Errorf("failed to fetch task: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.Task{}, err
	}
	var task model.Task
	err = json.Unmarshal(b, &task)
	if err != nil {
		return model.Task{}, err
	}

	return task, nil
}

func Results(ctx context.Context, url string, result model.Response, apiKey string) error {
	slog.Info("posting result", "id", result.ID, "texts", len(result.Text))
	defer func() {
		slog.Info("posted result", "id", result.ID, "texts", len(result.Text))
	}()
	hc := &http.Client{
		Timeout: 30 * time.Second,
	}

	b, err := json.Marshal(result)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	setBearerAuth(req, apiKey)

	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("failed to post result: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	return nil
}

func setBearerAuth(req *http.Request, apiKey string) {
	if apiKey == "" {
		return
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
}
