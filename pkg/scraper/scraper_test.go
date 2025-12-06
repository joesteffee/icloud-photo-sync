package scraper

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestScraper_ExtractImageURLs(t *testing.T) {
	scraper := &Scraper{}

	tests := []struct {
		name     string
		html     string
		wantURLs []string
	}{
		{
			name: "iCloud CDN URLs",
			html: `<html><body><img src="https://cvws.icloud-content.com/B/image.jpg"></body></html>`,
			wantURLs: []string{"https://cvws.icloud-content.com/B/image.jpg"},
		},
		{
			name: "JSON embedded URLs",
			html: `<script>{"url": "https://example.com/image.jpg"}</script>`,
			wantURLs: []string{"https://example.com/image.jpg"},
		},
		{
			name: "data-src attributes",
			html: `<img data-src="https://example.com/image.jpg">`,
			wantURLs: []string{"https://example.com/image.jpg"},
		},
		{
			name: "multiple URLs",
			html: `<html>
				<img src="https://example.com/image1.jpg">
				<img src="https://example.com/image2.jpg">
				<script>{"url": "https://example.com/image3.jpg"}</script>
			</html>`,
			wantURLs: []string{
				"https://example.com/image1.jpg",
				"https://example.com/image2.jpg",
				"https://example.com/image3.jpg",
			},
		},
		{
			name: "relative URLs filtered",
			html: `<html><img src="/relative/path.jpg"></html>`,
			wantURLs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls := scraper.extractImageURLs(tt.html)
			
			// Check that we got at least the expected URLs
			urlMap := make(map[string]bool)
			for _, url := range urls {
				urlMap[url] = true
			}
			
			for _, wantURL := range tt.wantURLs {
				if !urlMap[wantURL] {
					t.Errorf("extractImageURLs() missing URL: %v", wantURL)
				}
			}
		})
	}
}

func TestScraper_GetImageURLs(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `<html>
			<img src="https://cvws.icloud-content.com/B/test1.jpg">
			<img src="https://cvws.icloud-content.com/B/test2.jpg">
		</html>`
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	scraper := NewScraper(server.URL)
	urls, err := scraper.GetImageURLs()
	if err != nil {
		t.Fatalf("GetImageURLs() error = %v", err)
	}

	if len(urls) == 0 {
		t.Error("GetImageURLs() returned no URLs")
	}

	// Check for expected URLs
	found1 := false
	found2 := false
	for _, url := range urls {
		if url == "https://cvws.icloud-content.com/B/test1.jpg" {
			found1 = true
		}
		if url == "https://cvws.icloud-content.com/B/test2.jpg" {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Errorf("GetImageURLs() did not find expected URLs. Found: %v", urls)
	}
}

func TestScraper_GetImageURLs_Error(t *testing.T) {
	// Create a test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	scraper := NewScraper(server.URL)
	_, err := scraper.GetImageURLs()
	if err == nil {
		t.Error("GetImageURLs() expected error for 500 status")
	}
}

func TestScraper_GetImageURLs_Deduplication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `<html>
			<img src="https://example.com/duplicate.jpg">
			<img src="https://example.com/duplicate.jpg">
			<img src="https://example.com/duplicate.jpg">
		</html>`
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	scraper := NewScraper(server.URL)
	urls, err := scraper.GetImageURLs()
	if err != nil {
		t.Fatalf("GetImageURLs() error = %v", err)
	}

	// Should only have one unique URL
	urlMap := make(map[string]bool)
	for _, url := range urls {
		if urlMap[url] {
			t.Errorf("GetImageURLs() returned duplicate URL: %v", url)
		}
		urlMap[url] = true
	}
}

