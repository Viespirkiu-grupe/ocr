package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Viespirkiu-grupe/ocr/internal/config"
	"github.com/Viespirkiu-grupe/ocr/internal/model"
	"github.com/Viespirkiu-grupe/ocr/internal/pkg/fetcher"
)

var (
	pages int64
	files int64
	since time.Time
)

func main() {
	ctx := context.Background()
	slog.SetLogLoggerLevel(slog.LevelDebug)
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Value.Kind() == slog.KindString && a.Key == "url" {
				val := a.Value.String()
				if u, err := url.Parse(val); err == nil {
					q := u.Query()
					if q.Has("apiKey") {
						q.Set("apiKey", "******")
						u.RawQuery = q.Encode()
						val = u.String()
					}
				}
				return slog.String(a.Key, val)
			}
			return a
		},
	})
	slog.SetDefault(slog.New(handler))
	if err := run(ctx); err != nil {
		slog.Error("fatal error", "error", err)
	}
}

func run(ctx context.Context) error {
	config := config.Load()
	since = time.Now()

	slog.Info("starting ocr worker", "config", config)

	if _, err := os.Stat(config.InboxDir); os.IsNotExist(err) {
		return fmt.Errorf("inbox dir does not exist. Check for env INBOX_DIR")
	}
	go func() {
		for {
			slog.Info("polling for task")
			task, err := fetcher.Task(ctx, config.NextURL)
			if err != nil {
				slog.Error("fetch task", "error", err)
				time.Sleep(10 * time.Second)
				continue
			}

			slog.Info("fetched task", "id", task.ID, "filename", task.Uri)

			if err := process(ctx, task, config); err != nil {
				slog.Error("process task", "id", task.ID, "error", err)
				continue
			}
			atomic.AddInt64(&files, 1)
		}
	}()

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				slog.Info("stats", "files", atomic.LoadInt64(&files), "duration", time.Since(since), "files/sec", float64(atomic.LoadInt64(&files))/time.Since(since).Seconds(), "files/min", float64(atomic.LoadInt64(&files))/time.Since(since).Minutes(), "pages", atomic.LoadInt64(&pages), "pages/sec", float64(atomic.LoadInt64(&pages))/time.Since(since).Seconds(), "pages/min", float64(atomic.LoadInt64(&pages))/time.Since(since).Minutes())
			case <-ctx.Done():
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
	}
	return nil
}

func process(ctx context.Context, task model.Task, config config.Config) error {
	fileURL := strings.TrimRight(config.BaseFileURL, "/") + task.Uri

	tmpFile := os.Getenv("INBOX_DIR") + "/" + task.IDString() + ".pdf"
	defer os.RemoveAll(tmpFile)
	if err := fetcher.File(ctx, fileURL, tmpFile); err != nil {
		return err
	}

	slog.Info("fetched file", "id", task.ID, "file", tmpFile)
	tmpDir := os.Getenv("INBOX_DIR") + "/tmp/" + task.IDString()
	defer os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)

	pageCount, err := getPageCount(ctx, tmpFile)
	if err != nil {
		return err
	}

	slog.Info("page count", "id", task.ID, "pages", pageCount)

	var wg sync.WaitGroup
	start := time.Now().UnixMilli()
	sem := make(chan struct{}, config.Concurrency)
	for i := 1; i <= pageCount; i++ {
		wg.Add(1)
		go func(page int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := runGs(ctx, tmpDir, tmpFile, page); err != nil {
				slog.Error("run gs", "id", task.ID, "page", page, "error", err)
				return
			}
			if err := runTesseract(ctx, tmpDir+"/"+fmt.Sprintf("page-%04d.png", page), tmpDir+"/page-"+fmt.Sprintf("%04d", page), config.TesseractLang); err != nil {
				slog.Error("run tesseract", "id", task.ID, "page", page, "error", err)
				return
			}
			os.Remove(fmt.Sprintf("page-%04d.png", page))
			slog.Info("processed page", "id", task.ID, "page", page)
		}(i)
	}
	wg.Wait()

	diff := time.Now().UnixMilli() - start
	slog.Info("all pages processed", "id", task.ID, "pages", pageCount, "ms", diff)

	texts, err := collectTextFiles(tmpDir, pageCount)
	if err != nil {
		return fmt.Errorf("collect text files: %w", err)
	}

	atomic.AddInt64(&pages, int64(pageCount))

	result := model.Response{
		ID:       task.ID,
		Text:     texts,
		Duration: diff,
	}

	slog.Info("collected texts", "id", task.ID, "pages", len(texts))

	return fetcher.Results(ctx, config.ResultURL, result)
}

func getPageCount(ctx context.Context, inputFile string) (int, error) {
	cmd := exec.Command("pdfinfo", inputFile)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Pages:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				var n int
				_, err := fmt.Sscanf(parts[1], "%d", &n)
				if err != nil {
					return 0, err
				}
				return n, nil
			}
		}
	}
	return 0, fmt.Errorf("could not find page count in pdfinfo output")
}

func runGs(ctx context.Context, dir, inputFile string, page int) error {
	cmd := exec.CommandContext(ctx, "gs", "-dNOPAUSE", "-dBATCH", "-sDEVICE=pnggray", "-r300", "-dQUIET", "-dSAFER",
		"-dFirstPage="+fmt.Sprintf("%d", page), "-dLastPage="+fmt.Sprintf("%d", page), "-sstdout=%stderr",
		"-sOutputFile="+dir+"/page-"+fmt.Sprintf("%04d", page)+".png", "--", inputFile)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func runTesseract(ctx context.Context, inputFile string, outputFile string, lang string) error {
	cmd := exec.CommandContext(ctx, "tesseract", "-l", lang, inputFile, outputFile, "txt")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func collectTextFiles(dir string, pageCount int) ([]string, error) {
	var texts []string
	for i := 1; i <= pageCount; i++ {
		filename := fmt.Sprintf("%s/page-%04d.txt", dir, i)
		data, err := os.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		texts = append(texts, string(data))
	}
	return texts, nil
}
