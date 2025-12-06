package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Manager handles image downloads and hash calculation
type Manager struct {
	imageDir string
	client   *http.Client
}

// NewManager creates a new storage manager
func NewManager(imageDir string) (*Manager, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create image directory: %w", err)
	}

	return &Manager{
		imageDir: imageDir,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// DownloadAndHash downloads an image and calculates its SHA-256 hash
// Returns the local file path and the hash
func (m *Manager) DownloadAndHash(imageURL string) (string, string, error) {
	// Download the image
	resp, err := m.client.Get(imageURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Create a tee reader to both hash and write the file
	hasher := sha256.New()
	tee := io.TeeReader(resp.Body, hasher)

	// Determine file extension from URL or Content-Type
	ext := m.getFileExtension(imageURL, resp.Header.Get("Content-Type"))
	
	// Create a temporary file first
	tmpFile, err := os.CreateTemp(m.imageDir, "download-*"+ext)
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Write to temp file
	_, err = io.Copy(tmpFile, tee)
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		return "", "", fmt.Errorf("failed to write image: %w", err)
	}

	// Calculate hash
	hash := hex.EncodeToString(hasher.Sum(nil))

	// Check if file with this hash already exists
	hashPath := filepath.Join(m.imageDir, hash+ext)
	if _, err := os.Stat(hashPath); err == nil {
		// File already exists, remove temp file and return existing
		os.Remove(tmpPath)
		return hashPath, hash, nil
	}

	// Rename temp file to hash-based filename
	if err := os.Rename(tmpPath, hashPath); err != nil {
		os.Remove(tmpPath)
		return "", "", fmt.Errorf("failed to rename file: %w", err)
	}

	return hashPath, hash, nil
}

// getFileExtension determines the file extension from URL or Content-Type
func (m *Manager) getFileExtension(url, contentType string) string {
	// Try to get extension from URL
	if ext := filepath.Ext(url); ext != "" {
		// Remove query parameters
		ext = strings.Split(ext, "?")[0]
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp" {
			return ext
		}
	}

	// Try to get extension from Content-Type
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		// Default to .jpg
		return ".jpg"
	}
}

// GetImagePath returns the path to an image by hash
func (m *Manager) GetImagePath(hash string) (string, error) {
	// Try common extensions
	extensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	for _, ext := range extensions {
		path := filepath.Join(m.imageDir, hash+ext)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("image not found for hash: %s", hash)
}

