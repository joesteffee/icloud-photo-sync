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
// Example: https://www.icloud.com/sharedalbum/#EXAMPLE_TOKEN -> EXAMPLE_TOKEN
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
		// Priority: original > medium (skip thumbnail - not high quality enough)
		// Only use high-quality versions for both email and Google Photos sync
		var bestURL *string
		var qualityUsed string
		
		// Try original first (highest quality)
		if derivative, ok := photo.Derivatives["original"]; ok && derivative.URL != nil {
			bestURL = derivative.URL
			qualityUsed = "original"
		} else if derivative, ok := photo.Derivatives["medium"]; ok && derivative.URL != nil {
			// Fall back to medium if original not available
			bestURL = derivative.URL
			qualityUsed = "medium"
		}
		
		// Skip thumbnail - not high quality enough for email/Google Photos
		// If neither original nor medium is available, skip this photo
		if bestURL == nil {
			// Log that we're skipping due to insufficient quality
			continue
		}
		
		urls = append(urls, *bestURL)
		// Note: Quality logging can be added here if needed for debugging
		_ = qualityUsed // Quality used for this image (original or medium)
	}

	return urls, nil
}


