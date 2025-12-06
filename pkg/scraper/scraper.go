package scraper

import (
	"fmt"
	"strings"

	icloudalbum "github.com/Shogoki/icloud-shared-album-go"
)

// Scraper scrapes iCloud shared albums for image URLs
type Scraper struct {
	albumURL string
	token    string
	client   *icloudalbum.Client
}

// NewScraper creates a new scraper instance
func NewScraper(albumURL string) *Scraper {
	// Extract token from URL (part after #)
	token := extractTokenFromURL(albumURL)
	
	return &Scraper{
		albumURL: albumURL,
		token:    token,
		client:   icloudalbum.NewClient(),
	}
}

// extractTokenFromURL extracts the album token from an iCloud shared album URL
// Example: https://www.icloud.com/sharedalbum/#B2Z59UlCqSTGqW -> B2Z59UlCqSTGqW
func extractTokenFromURL(url string) string {
	// Find the token after #
	hashIdx := strings.Index(url, "#")
	if hashIdx == -1 {
		return ""
	}
	token := url[hashIdx+1:]
	// Remove any query parameters or fragments
	if semicolonIdx := strings.Index(token, ";"); semicolonIdx != -1 {
		token = token[:semicolonIdx]
	}
	return token
}

// GetImageURLs extracts image URLs from the iCloud shared album using the API
func (s *Scraper) GetImageURLs() ([]string, error) {
	if s.token == "" {
		return nil, fmt.Errorf("invalid album URL: could not extract token from %s", s.albumURL)
	}

	// Use the iCloud shared album library to get images
	response, err := s.client.GetImages(s.token)
	if err != nil {
		return nil, fmt.Errorf("failed to get images from iCloud API: %w", err)
	}

	var urls []string
	for _, photo := range response.Photos {
		// Get the highest quality derivative available
		// Priority: original > medium > thumbnail
		var bestURL *string
		for _, size := range []string{"original", "medium", "thumbnail"} {
			if derivative, ok := photo.Derivatives[size]; ok && derivative.URL != nil {
				bestURL = derivative.URL
				break // Use first available (original is preferred)
			}
		}
		
		// If no named size found, try to get any derivative with a URL
		if bestURL == nil {
			for _, derivative := range photo.Derivatives {
				if derivative.URL != nil {
					bestURL = derivative.URL
					break
				}
			}
		}
		
		if bestURL != nil {
			urls = append(urls, *bestURL)
		}
	}

	return urls, nil
}


