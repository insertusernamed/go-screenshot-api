package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// allowing all origins
		w.Header().Set("Access-Control-Allow-Origin", "*")
		next.ServeHTTP(w, r)
	})
}

func main() {
	// creating a new router to handle incoming http requests
	mux := http.NewServeMux()

	// registering the screenshot endpoint to handle screenshot requests
	screenshotHandlerFunc := http.HandlerFunc(screenshotHandler)
	mux.Handle("/screenshot", corsMiddleware(screenshotHandlerFunc))

	// starting the http server on port 8080 to listen for requests
	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func screenshotHandler(w http.ResponseWriter, r *http.Request) {
	// extracting the target url from query parameters for screenshot capture
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "url query parameter is required", http.StatusBadRequest)
		return
	}

	// parsing screen resolution parameters with defaults and limits
	width, height := parseResolution(r.URL.Query().Get("width"), r.URL.Query().Get("height"))

	// determining whether to capture full page or just viewport
	fullPage := r.URL.Query().Get("fullpage") == "true"

	// determining whether to wait for network idle
	waitNetworkIdle := r.URL.Query().Get("networkidle") == "true"

	// creating a new chrome context with custom viewport size
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.WindowSize(width, height),
	)
	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	// setting a timeout to prevent requests from hanging indefinitely
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second) // increased timeout for network waiting
	defer cancel()

	// navigating to the url and capturing screenshot based on settings
	var buf []byte
	var err error

	if waitNetworkIdle {
		if fullPage {
			err = chromedp.Run(ctx,
				network.Enable(),
				chromedp.Navigate(url),
				waitForNetworkIdle(500*time.Millisecond, 2*time.Second),
				chromedp.FullScreenshot(&buf, 90),
			)
		} else {
			err = chromedp.Run(ctx,
				network.Enable(),
				chromedp.Navigate(url),
				waitForNetworkIdle(500*time.Millisecond, 2*time.Second),
				chromedp.CaptureScreenshot(&buf),
			)
		}
	} else {
		if fullPage {
			err = chromedp.Run(ctx,
				chromedp.Navigate(url),
				chromedp.FullScreenshot(&buf, 90),
			)
		} else {
			err = chromedp.Run(ctx,
				chromedp.Navigate(url),
				chromedp.CaptureScreenshot(&buf),
			)
		}
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// returning the screenshot as a png image to the client
	w.Header().Set("Content-Type", "image/png")
	w.Write(buf)
}

// waitForNetworkIdle creates a custom action that waits for network activity to settle
func waitForNetworkIdle(idleDuration, maxWait time.Duration) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		var activeRequests int
		var mu sync.Mutex
		var lastActivity time.Time

		mu.Lock()
		lastActivity = time.Now()
		mu.Unlock()

		// listening for network events
		chromedp.ListenTarget(ctx, func(ev interface{}) {
			mu.Lock()
			defer mu.Unlock()

			switch ev.(type) {
			case *network.EventRequestWillBeSent:
				activeRequests++
				lastActivity = time.Now()
			case *network.EventLoadingFinished, *network.EventLoadingFailed:
				if activeRequests > 0 {
					activeRequests--
				}
				lastActivity = time.Now()
			}
		})

		// waiting for network to be idle
		start := time.Now()
		for {
			mu.Lock()
			currentActiveRequests := activeRequests
			timeSinceLastActivity := time.Since(lastActivity)
			mu.Unlock()

			// checking if network has been idle for the required duration
			if currentActiveRequests <= 2 && timeSinceLastActivity >= idleDuration {
				break
			}

			// checking if we've exceeded maximum wait time
			if time.Since(start) >= maxWait {
				break
			}

			time.Sleep(50 * time.Millisecond)
		}

		return nil
	})
}

// parseResolution parses width and height from query parameters with validation
func parseResolution(widthStr, heightStr string) (int, int) {
	// defaulting to 720p resolution (maybe it should be 1080p)
	width, height := 1280, 720

	// parsing width with 4k limit (3840px)
	if w, err := strconv.Atoi(widthStr); err == nil && w > 0 && w <= 3840 {
		width = w
	}

	// parsing height with 4k limit (2160px)
	if h, err := strconv.Atoi(heightStr); err == nil && h > 0 && h <= 2160 {
		height = h
	}

	return width, height
}