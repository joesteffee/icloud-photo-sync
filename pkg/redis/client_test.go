package redis

import (
	"testing"
)

func setupTestRedis(t *testing.T) *Client {
	// Use a test Redis instance or mock
	// For testing, we'll use a real Redis connection to localhost
	// In CI, this would use testcontainers or a mock
	client, err := NewClient("redis://localhost:6379")
	if err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
	}
	return client
}

func TestClient_HashExists(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	hash := "test-hash-123"
	imageURL := "https://example.com/image.jpg"

	// Test non-existent hash
	exists, err := client.HashExists(hash)
	if err != nil {
		t.Fatalf("HashExists() error = %v", err)
	}
	if exists {
		t.Error("HashExists() = true, want false for non-existent hash")
	}

	// Set hash
	err = client.SetHash(hash, imageURL)
	if err != nil {
		t.Fatalf("SetHash() error = %v", err)
	}

	// Test existing hash
	exists, err = client.HashExists(hash)
	if err != nil {
		t.Fatalf("HashExists() error = %v", err)
	}
	if !exists {
		t.Error("HashExists() = false, want true for existing hash")
	}
}

func TestClient_SetHash(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	hash := "test-hash-456"
	imageURL := "https://example.com/image2.jpg"

	err := client.SetHash(hash, imageURL)
	if err != nil {
		t.Fatalf("SetHash() error = %v", err)
	}

	// Verify it was set
	exists, err := client.HashExists(hash)
	if err != nil {
		t.Fatalf("HashExists() error = %v", err)
	}
	if !exists {
		t.Error("SetHash() did not set the hash")
	}
}

func TestClient_GetHash(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	hash := "test-hash-789"
	imageURL := "https://example.com/image3.jpg"

	// Set hash
	err := client.SetHash(hash, imageURL)
	if err != nil {
		t.Fatalf("SetHash() error = %v", err)
	}

	// Get hash
	url, err := client.GetHash(hash)
	if err != nil {
		t.Fatalf("GetHash() error = %v", err)
	}
	if url != imageURL {
		t.Errorf("GetHash() = %v, want %v", url, imageURL)
	}

	// Test non-existent hash
	url, err = client.GetHash("non-existent-hash")
	if err != nil {
		t.Fatalf("GetHash() error = %v", err)
	}
	if url != "" {
		t.Errorf("GetHash() = %v, want empty string", url)
	}
}

func TestClient_Close(t *testing.T) {
	client := setupTestRedis(t)
	
	err := client.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestHashKey(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	hash := "abc123"
	// Use reflection or test the behavior indirectly
	// Since hashKey is private, we test it through public methods
	err := client.SetHash(hash, "https://example.com/test.jpg")
	if err != nil {
		t.Fatalf("SetHash() error = %v", err)
	}

	// Verify the key format by checking if it exists
	exists, err := client.HashExists(hash)
	if err != nil {
		t.Fatalf("HashExists() error = %v", err)
	}
	if !exists {
		t.Error("Hash key was not set correctly")
	}
}

// Test with a mock Redis for unit tests without requiring Redis
func TestClient_WithMock(t *testing.T) {
	// This would use a mock Redis client for true unit testing
	// For now, we rely on integration tests with real Redis
	t.Skip("Mock Redis tests not implemented")
}

func TestClient_SeparateEmailAndGooglePhotosTracking(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	hash := "test-hash-separate"
	imageURL := "https://example.com/image.jpg"

	// Set hash for email only
	err := client.SetHashForEmail(hash, imageURL)
	if err != nil {
		t.Fatalf("SetHashForEmail() error = %v", err)
	}

	// Verify email tracking
	emailExists, err := client.HashExistsForEmail(hash)
	if err != nil {
		t.Fatalf("HashExistsForEmail() error = %v", err)
	}
	if !emailExists {
		t.Error("HashExistsForEmail() = false, want true")
	}

	// Verify Google Photos tracking is independent (should not exist)
	gphotosExists, err := client.HashExistsForGooglePhotos(hash)
	if err != nil {
		t.Fatalf("HashExistsForGooglePhotos() error = %v", err)
	}
	if gphotosExists {
		t.Error("HashExistsForGooglePhotos() = true, want false (email and Google Photos should be independent)")
	}

	// Now set hash for Google Photos
	err = client.SetHashForGooglePhotos(hash, imageURL)
	if err != nil {
		t.Fatalf("SetHashForGooglePhotos() error = %v", err)
	}

	// Verify both now exist independently
	emailExists, err = client.HashExistsForEmail(hash)
	if err != nil {
		t.Fatalf("HashExistsForEmail() error = %v", err)
	}
	if !emailExists {
		t.Error("HashExistsForEmail() = false, want true")
	}

	gphotosExists, err = client.HashExistsForGooglePhotos(hash)
	if err != nil {
		t.Fatalf("HashExistsForGooglePhotos() error = %v", err)
	}
	if !gphotosExists {
		t.Error("HashExistsForGooglePhotos() = false, want true")
	}
}

func TestClient_BackwardCompatibility(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	hash := "test-hash-backward"
	imageURL := "https://example.com/image.jpg"

	// Test that old HashExists and SetHash still work (should use email tracking)
	err := client.SetHash(hash, imageURL)
	if err != nil {
		t.Fatalf("SetHash() error = %v", err)
	}

	exists, err := client.HashExists(hash)
	if err != nil {
		t.Fatalf("HashExists() error = %v", err)
	}
	if !exists {
		t.Error("HashExists() = false, want true (backward compatibility)")
	}

	// Verify it's actually using email tracking
	emailExists, err := client.HashExistsForEmail(hash)
	if err != nil {
		t.Fatalf("HashExistsForEmail() error = %v", err)
	}
	if !emailExists {
		t.Error("SetHash() should set email tracking for backward compatibility")
	}
}

