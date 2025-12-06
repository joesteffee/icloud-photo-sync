package scraper

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Scraper scrapes iCloud shared albums for image URLs
type Scraper struct {
	albumURL string
	client   *http.Client
}

// NewScraper creates a new scraper instance
func NewScraper(albumURL string) *Scraper {
	return &Scraper{
		albumURL: albumURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetImageURLs extracts image URLs from the iCloud shared album page
func (s *Scraper) GetImageURLs() ([]string, error) {
	req, err := http.NewRequest("GET", s.albumURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers to mimic a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch album page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	html := string(body)
	urls := s.extractImageURLs(html)

	// Remove duplicates
	uniqueURLs := make(map[string]bool)
	var result []string
	for _, url := range urls {
		if !uniqueURLs[url] {
			uniqueURLs[url] = true
			result = append(result, url)
		}
	}

	return result, nil
}

// extractImageURLs extracts image URLs from HTML content
// iCloud shared albums embed image URLs in various ways:
// 1. In JSON data embedded in script tags
// 2. In data attributes
// 3. In img src attributes
func (s *Scraper) extractImageURLs(html string) []string {
	var urls []string

	// Pattern 1: Look for iCloud CDN URLs (icloud-content.com)
	// These are typically high-resolution image URLs
	icloudPattern := regexp.MustCompile(`https://[^"'\s]+icloud-content\.com[^"'\s]+`)
	matches := icloudPattern.FindAllString(html, -1)
	urls = append(urls, matches...)

	// Pattern 2: Look for JSON data with image URLs
	// iCloud often embeds data in JSON format
	jsonPattern := regexp.MustCompile(`"url":\s*"([^"]+)"`)
	jsonMatches := jsonPattern.FindAllStringSubmatch(html, -1)
	for _, match := range jsonMatches {
		if len(match) > 1 && strings.Contains(match[1], "http") {
			urls = append(urls, match[1])
		}
	}

	// Pattern 3: Look for data-src or data-url attributes
	dataSrcPattern := regexp.MustCompile(`data-(?:src|url)="([^"]+)"`)
	dataMatches := dataSrcPattern.FindAllStringSubmatch(html, -1)
	for _, match := range dataMatches {
		if len(match) > 1 && strings.Contains(match[1], "http") {
			urls = append(urls, match[1])
		}
	}

	// Pattern 4: Look for img src attributes with iCloud URLs
	imgSrcPattern := regexp.MustCompile(`<img[^>]+src="([^"]+)"`)
	imgMatches := imgSrcPattern.FindAllStringSubmatch(html, -1)
	for _, match := range imgMatches {
		if len(match) > 1 && (strings.Contains(match[1], "icloud") || strings.Contains(match[1], "http")) {
			urls = append(urls, match[1])
		}
	}

	// Filter to only include valid HTTP/HTTPS URLs and remove relative URLs
	var validURLs []string
	for _, url := range urls {
		url = strings.TrimSpace(url)
		if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
			// Remove query parameters that might be for thumbnails, look for full-size images
			// iCloud URLs often have parameters like ?width=, ?size=, etc.
			// We want the full-size version, so we'll keep the URL as-is for now
			// The storage manager can handle downloading the best quality
			validURLs = append(validURLs, url)
		}
	}

	return validURLs
}

