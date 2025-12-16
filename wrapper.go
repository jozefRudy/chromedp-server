package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ScrapeRequest struct {
	URLs []string `json:"urls"`
}

type ScrapeResult struct {
	Content string `json:"content"`
	Title   string `json:"title"`
	Error   string `json:"error,omitempty"`
}

type ScrapeResponse struct {
	Results []ScrapeResult `json:"results"`
}

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_1_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (iPad; CPU OS 17_1_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Mobile/15E148 Safari/604.1",
}

func scrapeURL(ctx context.Context, url string, timeout int, userAgent string) ScrapeResult {
	start := time.Now()

	urlCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()

	var htmlContent string
	var title string

	// Simulate human behavior: random mouse movements, scrolling, waiting
	randomScroll := fmt.Sprintf(`
		window.scrollTo(0, %d);
		setTimeout(() => window.scrollTo(0, document.body.scrollHeight * 0.5), %d);
		setTimeout(() => window.scrollTo(0, document.body.scrollHeight), %d);
	`, rand.Intn(300)+100, rand.Intn(500)+300, rand.Intn(700)+500)

	err := chromedp.Run(urlCtx,
		// Override navigator properties to evade headless detection
		chromedp.Evaluate(fmt.Sprintf(`
			Object.defineProperty(navigator, 'webdriver', {get: () => undefined});
			Object.defineProperty(navigator, 'plugins', {get: () => [
				{name: 'Chrome PDF Plugin', description: 'Portable Document Format', filename: 'internal-pdf-viewer'},
				{name: 'Chrome PDF Viewer', description: '', filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai'},
				{name: 'Native Client', description: '', filename: 'internal-nacl-plugin'}
			]});
			navigator.languages = ['en-US', 'en'];
			Object.defineProperty(navigator, 'userAgent', {get: () => '%s'});
		`, userAgent), nil),
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
		// Extended wait for anti-bot challenges
		chromedp.Sleep(time.Duration(rand.Intn(5000)+10000)*time.Millisecond), // 10-15s
		// Check for bot detection challenges
		chromedp.Evaluate(`
			(async () => {
				const isChallenge = document.title.includes('Just a moment') ||
								   document.body.innerText.includes('Verifying you are human') ||
								   document.body.innerText.includes('Checking your browser') ||
								   document.body.innerText.includes('Please wait while we are checking your browser') ||
								   document.body.innerText.includes('Access denied') ||
								   document.body.innerText.includes('403 Forbidden') ||
								   document.body.innerText.includes('Bot detected') ||
								   document.body.innerText.includes('Security check in progress') ||
								   window.location.href.includes('challenge');
				if (isChallenge) {
					// Wait additional time for challenge completion
					await new Promise(resolve => setTimeout(resolve, 5000 + Math.random() * 5000));
				}
			})();
		`, nil),
		// Simulate scrolling behavior
		chromedp.Evaluate(randomScroll, nil),
		chromedp.Title(&title),
		chromedp.OuterHTML("html", &htmlContent),
	)

	if err != nil {
		log.Error().
			Str("url", url).
			Dur("duration", time.Since(start)).
			Err(err).
			Msg("Scrape failed")
		return ScrapeResult{Error: err.Error()}
	}

	if title == "" {
		title = "Untitled"
	}

	log.Info().
		Str("url", url).
		Dur("duration", time.Since(start)).
		Int("size", len(htmlContent)).
		Msg("Scrape completed")

	return ScrapeResult{Content: htmlContent, Title: title}
}

func scrapeHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		log.Error().Str("remote_ip", r.RemoteAddr).Str("method", r.Method).Msg("Invalid method")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ScrapeResponse{
			Results: []ScrapeResult{{Error: "Method not allowed"}},
		})
		return
	}

	var req ScrapeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error().Str("remote_ip", r.RemoteAddr).Err(err).Msg("Invalid request body")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ScrapeResponse{
			Results: []ScrapeResult{{Error: "Invalid request"}},
		})
		return
	}

	minURLs := 1
	maxURLs := 10
	if len(req.URLs) > maxURLs || len(req.URLs) < minURLs {
		log.Error().
			Str("remote_ip", r.RemoteAddr).
			Int("url_count", len(req.URLs)).
			Int("min_urls", minURLs).
			Int("max_urls", maxURLs).
			Msg("Invalid URL count")
		w.WriteHeader(http.StatusBadRequest)
		var errorMsg string
		if len(req.URLs) < minURLs {
			errorMsg = "No URLs provided"
		} else {
			errorMsg = fmt.Sprintf("Too many URLs (max %d)", maxURLs)
		}

		json.NewEncoder(w).Encode(ScrapeResponse{
			Results: []ScrapeResult{{Error: errorMsg}},
		})
		return
	}

	// Process each URL with its own browser instance for maximum robustness
	results := make([]ScrapeResult, len(req.URLs))
	for i, url := range req.URLs {
		func() {
			// Random viewport size to avoid fingerprinting
			viewportWidth := int64(rand.Intn(400) + 1280) // 1280-1680
			viewportHeight := int64(rand.Intn(300) + 720) // 720-1020

			opts := append(chromedp.DefaultExecAllocatorOptions[:],
				chromedp.ExecPath("/headless-shell/headless-shell"),
				chromedp.Flag("no-sandbox", true),
				chromedp.Flag("disable-dev-shm-usage", true),
				chromedp.Flag("disable-blink-features", "AutomationControlled"),
				chromedp.WindowSize(int(viewportWidth), int(viewportHeight)),
			)

			selectedUA := userAgents[rand.Intn(len(userAgents))]
			opts = append(opts, chromedp.UserAgent(selectedUA))

			// Each URL gets its own browser instance (fresh fingerprint)
			allocCtx, allocCancel := chromedp.NewExecAllocator(r.Context(), opts...)
			defer allocCancel()

			urlCtx, urlCancel := chromedp.NewContext(allocCtx)
			defer urlCancel()

			results[i] = scrapeURL(urlCtx, url, 60_000, selectedUA)
		}()
	}

	log.Info().
		Int("url_count", len(req.URLs)).
		Dur("total_duration", time.Since(start)).
		Msg("Scrape request completed")

	json.NewEncoder(w).Encode(ScrapeResponse{Results: results})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func main() {
	zerolog.LevelFieldName = "message.LogLevel"
	zerolog.MessageFieldName = "message.Message"
	log.Logger = zerolog.New(os.Stdout)

	log.Info().Msg("Chromedp HTTP wrapper starting")

	http.HandleFunc("/scrape", scrapeHandler)
	http.HandleFunc("/health", healthHandler)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		log.Info().Msg("Server listening on port 4001")
		if err := http.ListenAndServe("0.0.0.0:4001", nil); err != nil {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	// Block until signal received
	<-quit
	log.Info().Msg("Shutdown signal received, exiting")
}
