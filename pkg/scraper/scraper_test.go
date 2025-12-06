package scraper

import (
	"testing"
)

func TestExtractTokenFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantToken string
	}{
		{
			name:     "standard URL",
			url:      "https://www.icloud.com/sharedalbum/#B2Z59UlCqSTGqW",
			wantToken: "B2Z59UlCqSTGqW",
		},
		{
			name:     "URL with semicolon",
			url:      "https://www.icloud.com/sharedalbum/#B2Z59UlCqSTGqW;param",
			wantToken: "B2Z59UlCqSTGqW",
		},
		{
			name:     "URL without hash",
			url:      "https://www.icloud.com/sharedalbum/",
			wantToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scraper := NewScraper(tt.url)
			if scraper.token != tt.wantToken {
				t.Errorf("extractTokenFromURL() = %v, want %v", scraper.token, tt.wantToken)
			}
		})
	}
}

func TestScraper_GetImageURLs_InvalidToken(t *testing.T) {
	// Test with invalid URL (no token)
	scraper := NewScraper("https://www.icloud.com/sharedalbum/")
	_, err := scraper.GetImageURLs()
	if err == nil {
		t.Error("GetImageURLs() expected error for invalid token")
	}
}

// Note: Testing GetImageURLs with a real token would require network access
// and a valid iCloud shared album. These integration tests are skipped
// in unit test runs but can be enabled for manual testing.
func TestScraper_GetImageURLs_Integration(t *testing.T) {
	t.Skip("Integration test - requires valid iCloud shared album token")
	
	// Uncomment and provide a valid token for integration testing:
	// scraper := NewScraper("https://www.icloud.com/sharedalbum/#YOUR_TOKEN_HERE")
	// urls, err := scraper.GetImageURLs()
	// if err != nil {
	// 	t.Fatalf("GetImageURLs() error = %v", err)
	// }
	// if len(urls) == 0 {
	// 	t.Error("GetImageURLs() returned no URLs")
	// }
}

