package main

import (
	"context"
	"flag"
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
			if strings.EqualFold(a.Key, "authorization") {
				return slog.String(a.Key, "Bearer ******")
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
	fileFlag := flag.String("file", "", "directly OCR a single file URL and print text to stdout")
	flag.Parse()

	config := config.Load()
	since = time.Now()

	if *fileFlag != "" {
		return runSingleFile(ctx, *fileFlag, config)
	}

	slog.Info("starting ocr worker",
		"inbox_dir", config.InboxDir,
		"base_file_url", config.BaseFileURL,
		"get_task_url", config.NextURL,
		"post_result_url", config.ResultURL,
		"concurrency", config.Concurrency,
		"tesseract_lang", config.TesseractLang,
		"api_key_configured", config.APIKey != "",
	)

	if _, err := os.Stat(config.InboxDir); os.IsNotExist(err) {
		return fmt.Errorf("inbox dir does not exist. Check for env INBOX_DIR")
	}
	go func() {
		for {
			slog.Info("polling for task")
			task, err := fetcher.Task(ctx, config.NextURL, config.APIKey)
			if err != nil {
				slog.Error("fetch task", "error", err)
				time.Sleep(10 * time.Second)
				continue
			}

			slog.Info("fetched task", "id", task.ID, "filename", task.Uri)

			if err := process(ctx, task, config); err != nil {
				slog.Error("process task", "id", task.ID, "error", err)
				time.Sleep(10 * time.Second)
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

// processPages runs gs+tesseract for every page using a fixed worker pool.
// If any page fails the first error is returned and all in-flight pages are cancelled.
func processPages(ctx context.Context, tmpDir, tmpFile string, pageCount int, cfg config.Config, logAttrs ...any) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 1)
	pageCh := make(chan int, pageCount)
	for i := 1; i <= pageCount; i++ {
		pageCh <- i
	}
	close(pageCh)

	var wg sync.WaitGroup
	for range cfg.Concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for page := range pageCh {
				if ctx.Err() != nil {
					return
				}
				if err := runGs(ctx, tmpDir, tmpFile, page, cfg.GsTimeout); err != nil {
					slog.Error("run gs", append(logAttrs, "page", page, "error", err)...)
					select {
					case errCh <- fmt.Errorf("page %d gs: %w", page, err):
						cancel()
					default:
					}
					return
				}
				if err := runTesseract(ctx, tmpDir+"/"+fmt.Sprintf("page-%04d.png", page), tmpDir+"/page-"+fmt.Sprintf("%04d", page), cfg.TesseractLang, cfg.TesseractTimeout); err != nil {
					slog.Error("run tesseract", append(logAttrs, "page", page, "error", err)...)
					select {
					case errCh <- fmt.Errorf("page %d tesseract: %w", page, err):
						cancel()
					default:
					}
					return
				}
				os.Remove(tmpDir + "/" + fmt.Sprintf("page-%04d.png", page))
				slog.Info("processed page", append(logAttrs, "page", page)...)
			}
		}()
	}
	wg.Wait()

	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func runSingleFile(ctx context.Context, fileURL string, cfg config.Config) error {
	uri := fileURL
	base := strings.TrimRight(cfg.BaseFileURL, "/")
	if strings.HasPrefix(fileURL, base) {
		uri = strings.TrimPrefix(fileURL, base)
	}

	tmpFile := cfg.InboxDir + "/test-single.pdf"
	defer os.RemoveAll(tmpFile)
	if err := fetcher.File(ctx, base+uri, tmpFile); err != nil {
		return fmt.Errorf("fetch file: %w", err)
	}

	tmpDir := cfg.InboxDir + "/tmp/test-single"
	defer os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)

	pageCount, err := getPageCountMuTool(ctx, tmpFile)
	if err != nil {
		return fmt.Errorf("page count: %w", err)
	}
	slog.Info("page count", "pages", pageCount)

	if err := processPages(ctx, tmpDir, tmpFile, pageCount, cfg); err != nil {
		return err
	}

	texts, err := collectTextFiles(tmpDir, pageCount)
	if err != nil {
		return fmt.Errorf("collect text files: %w", err)
	}

	for i, t := range texts {
		fmt.Printf("=== page %d ===\n%s\n", i+1, t)
	}
	return nil
}

func process(ctx context.Context, task model.Task, cfg config.Config) error {
	fileURL := strings.TrimRight(cfg.BaseFileURL, "/") + task.Uri

	tmpFile := cfg.InboxDir + "/" + task.IDString() + ".pdf"
	defer os.RemoveAll(tmpFile)
	if err := fetcher.File(ctx, fileURL, tmpFile); err != nil {
		return err
	}

	slog.Info("fetched file", "id", task.ID, "file", tmpFile)
	tmpDir := cfg.InboxDir + "/tmp/" + task.IDString()
	defer os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)

	pageCount, err := getPageCountMuTool(ctx, tmpFile)
	if err != nil {
		return err
	}
	slog.Info("page count", "id", task.ID, "pages", pageCount)

	start := time.Now().UnixMilli()
	if err := processPages(ctx, tmpDir, tmpFile, pageCount, cfg, "id", task.ID); err != nil {
		return err
	}
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
	postResults := func() error {
		var err error
		for range 5 {
			err = fetcher.Results(ctx, cfg.ResultURL, result, cfg.APIKey)
			if err == nil {
				break
			}
			slog.Error("post results", "id", task.ID, "error", err)
			time.Sleep(10 * time.Second)
		}
		return nil
	}
	return postResults()
}

func getPageCountMuTool(ctx context.Context, inputFile string) (int, error) {
	cmd := exec.CommandContext(ctx, "mutool", "show", inputFile, "trailer/Root/Pages/Count")
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	var n int
	_, err = fmt.Sscanf(string(out), "%d", &n)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func getPageCount(ctx context.Context, inputFile string) (int, error) {
	cmd := exec.CommandContext(ctx, "pdfinfo", inputFile)
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

func runGs(ctx context.Context, dir, inputFile string, page int, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gs", "-dNOPAUSE", "-dBATCH", "-sDEVICE=pnggray", "-r300", "-dQUIET", "-dSAFER",
		"-dFirstPage="+fmt.Sprintf("%d", page), "-dLastPage="+fmt.Sprintf("%d", page), "-sstdout=%stderr",
		"-sOutputFile="+dir+"/page-"+fmt.Sprintf("%04d", page)+".png", "--", inputFile)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func runTesseract(ctx context.Context, inputFile string, outputFile string, lang string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
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
