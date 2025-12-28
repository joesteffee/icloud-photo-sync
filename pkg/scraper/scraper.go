package scraper

import (
	"fmt"
	"log"
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
	skippedCount := 0
	for i, photo := range response.Photos {
		// Log available derivatives for debugging
		availableDerivatives := make([]string, 0, len(photo.Derivatives))
		for name := range photo.Derivatives {
			availableDerivatives = append(availableDerivatives, name)
		}
		if len(availableDerivatives) > 0 {
			log.Printf("Photo %d has derivatives: %v", i+1, availableDerivatives)
		} else {
			log.Printf("Photo %d has no derivatives", i+1)
		}
		
		// Get the highest quality derivative available
		// Priority: original > medium (skip thumbnail - not high quality enough)
		// Only use high-quality versions for both email and Google Photos sync
		var bestURL *string
		var qualityUsed string
		
		// Helper function to find derivative by name (case-insensitive)
		findDerivative := func(name string) (*icloudalbum.Derivative, bool) {
			// Try exact match first
			if deriv, ok := photo.Derivatives[name]; ok {
				return &deriv, true
			}
			// Try case-insensitive match
			for key, deriv := range photo.Derivatives {
				if strings.EqualFold(key, name) {
					return &deriv, true
				}
			}
			return nil, false
		}
		
		// Try original first (highest quality)
		if derivative, ok := findDerivative("original"); ok && derivative.URL != nil {
			bestURL = derivative.URL
			qualityUsed = "original"
			log.Printf("Photo %d: Using 'original' quality", i+1)
		} else if derivative, ok := findDerivative("medium"); ok && derivative.URL != nil {
			// Fall back to medium if original not available
			bestURL = derivative.URL
			qualityUsed = "medium"
			log.Printf("Photo %d: Using 'medium' quality (original not available)", i+1)
		}
		
		// Skip thumbnail - not high quality enough for email/Google Photos
		// If neither original nor medium is available, skip this photo
		if bestURL == nil {
			// Check if only thumbnail is available
			if _, hasThumbnail := photo.Derivatives["thumbnail"]; hasThumbnail {
				log.Printf("Photo %d: Skipping - only 'thumbnail' quality available (not high quality enough)", i+1)
			} else {
				log.Printf("Photo %d: Skipping - no 'original' or 'medium' derivative found. Available: %v", i+1, availableDerivatives)
			}
			skippedCount++
			continue
		}
		
		urls = append(urls, *bestURL)
		log.Printf("Photo %d: Added URL with quality '%s'", i+1, qualityUsed)
	}
	
	if skippedCount > 0 {
		log.Printf("Skipped %d photos due to insufficient quality (only thumbnail or no original/medium available)", skippedCount)
	}
	log.Printf("Total photos processed: %d, URLs extracted: %d", len(response.Photos), len(urls))

	return urls, nil
}


