package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	// creating a new router to handle incoming http requests
	mux := http.NewServeMux()

	// registering the screenshot endpoint to handle screenshot requests
	mux.HandleFunc("/screenshot", screenshotHandler)

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

	// creating a new chrome context for browser automation
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// setting a timeout to prevent requests from hanging indefinitely
	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// navigating to the url and capturing a full page screenshot
	var buf []byte
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.FullScreenshot(&buf, 90), // 90% quality
	); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// returning the screenshot as a png image to the client
	w.Header().Set("Content-Type", "image/png")
	w.Write(buf)
}