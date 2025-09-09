package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

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

	// creating a new chrome context with custom viewport size
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.WindowSize(width, height),
	)
	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	// setting a timeout to prevent requests from hanging indefinitely
	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// navigating to the url and capturing screenshot based on fullPage setting
	var buf []byte
	var err error

	if fullPage {
		err = chromedp.Run(ctx,
			chromedp.Navigate(url),
			chromedp.FullScreenshot(&buf, 90), // capturing entire page height
		)
	} else {
		err = chromedp.Run(ctx,
			chromedp.Navigate(url),
			chromedp.CaptureScreenshot(&buf), // capturing viewport only
		)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// returning the screenshot as a png image to the client
	w.Header().Set("Content-Type", "image/png")
	w.Write(buf)
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