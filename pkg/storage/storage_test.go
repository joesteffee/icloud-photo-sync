package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestManager_DownloadAndHash(t *testing.T) {
	// Create test image data
	testImageData := []byte("fake image data for testing")
	hashBytes := sha256.Sum256(testImageData)
	expectedHash := hex.EncodeToString(hashBytes[:])

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		w.Write(testImageData)
	}))
	defer server.Close()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	imagePath, hash, err := manager.DownloadAndHash(server.URL)
	if err != nil {
		t.Fatalf("DownloadAndHash() error = %v", err)
	}

	if hash != expectedHash {
		t.Errorf("DownloadAndHash() hash = %v, want %v", hash, expectedHash)
	}

	// Verify file exists
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		t.Errorf("DownloadAndHash() file does not exist: %v", imagePath)
	}

	// Verify file content
	fileData, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(fileData) != string(testImageData) {
		t.Errorf("DownloadAndHash() file content mismatch")
	}
}

func TestManager_DownloadAndHash_Duplicate(t *testing.T) {
	testImageData := []byte("duplicate test image")
	hashBytes := sha256.Sum256(testImageData)
	expectedHash := hex.EncodeToString(hashBytes[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		w.Write(testImageData)
	}))
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Download first time
	path1, hash1, err := manager.DownloadAndHash(server.URL)
	if err != nil {
		t.Fatalf("DownloadAndHash() first download error = %v", err)
	}

	// Download second time (should return existing file)
	path2, hash2, err := manager.DownloadAndHash(server.URL)
	if err != nil {
		t.Fatalf("DownloadAndHash() second download error = %v", err)
	}

	if hash1 != expectedHash || hash2 != expectedHash {
		t.Errorf("Hashes don't match expected: got %v and %v, want %v", hash1, hash2, expectedHash)
	}

	if path1 != path2 {
		t.Errorf("DownloadAndHash() returned different paths for same content: %v vs %v", path1, path2)
	}
}

func TestManager_GetFileExtension(t *testing.T) {
	manager := &Manager{}

	tests := []struct {
		name        string
		url         string
		contentType string
		want        string
	}{
		{
			name:        "extension from URL",
			url:         "https://example.com/image.jpg",
			contentType: "",
			want:        ".jpg",
		},
		{
			name:        "extension from Content-Type",
			url:         "https://example.com/image",
			contentType: "image/png",
			want:        ".png",
		},
		{
			name:        "default to jpg",
			url:         "https://example.com/image",
			contentType: "image/unknown",
			want:        ".jpg",
		},
		{
			name:        "remove query params",
			url:         "https://example.com/image.jpg?width=100",
			contentType: "",
			want:        ".jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manager.getFileExtension(tt.url, tt.contentType)
			if got != tt.want {
				t.Errorf("getFileExtension() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_GetImagePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	hash := "testhash123"
	
	// Create a test file
	testFile := filepath.Join(tmpDir, hash+".jpg")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	path, err := manager.GetImagePath(hash)
	if err != nil {
		t.Fatalf("GetImagePath() error = %v", err)
	}

	if path != testFile {
		t.Errorf("GetImagePath() = %v, want %v", path, testFile)
	}

	// Test non-existent hash
	_, err = manager.GetImagePath("nonexistent")
	if err == nil {
		t.Error("GetImagePath() expected error for non-existent hash")
	}
}

func TestManager_NewManager_CreatesDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	newDir := filepath.Join(tmpDir, "new-subdir")
	manager, err := NewManager(newDir)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if manager.imageDir != newDir {
		t.Errorf("NewManager() imageDir = %v, want %v", manager.imageDir, newDir)
	}

	// Verify directory was created
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Error("NewManager() did not create directory")
	}
}

