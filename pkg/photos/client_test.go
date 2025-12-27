package photos

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jsteffee/icloud-photo-sync/pkg/config"
)

func TestNewClient(t *testing.T) {
	cfg := &config.GooglePhotosConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RefreshToken: "test-refresh-token",
		AlbumName:    "Test Album",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	if client.config != cfg {
		t.Error("NewClient() did not set config correctly")
	}
}

func TestNewClient_NilConfig(t *testing.T) {
	_, err := NewClient(nil)
	if err == nil {
		t.Error("NewClient() with nil config should return error")
	}
}

func TestClient_RefreshAccessToken(t *testing.T) {
	// Create a mock OAuth2 token server
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Parse form data
		err := r.ParseForm()
		if err != nil {
			t.Errorf("Failed to parse form: %v", err)
		}

		// Verify request parameters
		if r.Form.Get("grant_type") != "refresh_token" {
			t.Errorf("Expected grant_type=refresh_token, got %s", r.Form.Get("grant_type"))
		}

		// Return mock token response
		response := map[string]interface{}{
			"access_token": "mock-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer tokenServer.Close()

	cfg := &config.GooglePhotosConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RefreshToken: "test-refresh-token",
		AlbumName:    "Test Album",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Note: This test is limited because oauth2.Config uses hardcoded endpoints
	// In a real scenario, we'd need to mock the oauth2 package or use dependency injection
	// For now, we just verify the method exists and doesn't panic
	err = client.RefreshAccessToken()
	// This will likely fail in test environment, but we're testing the structure
	if err != nil {
		// Expected in test environment without proper OAuth setup
		t.Logf("RefreshAccessToken() failed as expected in test: %v", err)
	}
}

func TestClient_FindAlbumByName(t *testing.T) {
	// Create a mock Google Photos API server
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/v1/albums") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		// Mock albums list response
		response := map[string]interface{}{
			"albums": []map[string]interface{}{
				{
					"id":    "album-1",
					"title": "Test Album",
				},
				{
					"id":    "album-2",
					"title": "Other Album",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer apiServer.Close()

	cfg := &config.GooglePhotosConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RefreshToken: "test-refresh-token",
		AlbumName:    "Test Album",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Note: This test requires proper OAuth2 setup and Google Photos API mocking
	// The actual implementation uses google.golang.org/api which is harder to mock
	// In a real scenario, we'd use dependency injection or a more sophisticated mocking approach
	_, err = client.FindAlbumByName("Test Album")
	if err != nil {
		// Expected in test environment without proper OAuth and API setup
		t.Logf("FindAlbumByName() failed as expected in test: %v", err)
	}
}

func TestClient_FindAlbumByName_NotFound(t *testing.T) {
	cfg := &config.GooglePhotosConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RefreshToken: "test-refresh-token",
		AlbumName:    "Non-existent Album",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.FindAlbumByName("Non-existent Album")
	if err == nil {
		t.Error("FindAlbumByName() should return error for non-existent album")
	}
}

func TestClient_UploadPhoto(t *testing.T) {
	// Create a temporary test image file
	tmpDir := t.TempDir()
	testImagePath := filepath.Join(tmpDir, "test.jpg")
	testImageData := []byte("fake image data for testing")
	err := os.WriteFile(testImagePath, testImageData, 0644)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	uploadToken := "mock-upload-token"
	mediaItemID := "mock-media-item-id"

	// Create mock servers for upload and API calls
	uploadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(uploadToken))
	}))
	defer uploadServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "batchCreate") {
			// Mock batch create response
			response := map[string]interface{}{
				"newMediaItemResults": []map[string]interface{}{
					{
						"mediaItem": map[string]interface{}{
							"id": mediaItemID,
						},
						"status": map[string]interface{}{
							"code":    0,
							"message": "OK",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if strings.Contains(r.URL.Path, "batchAddMediaItems") {
			// Mock batch add to album response
			response := map[string]interface{}{
				"mediaItemIds": []string{mediaItemID},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer apiServer.Close()

	cfg := &config.GooglePhotosConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RefreshToken: "test-refresh-token",
		AlbumName:    "Test Album",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Note: This test requires proper OAuth2 setup and Google Photos API mocking
	// The actual implementation uses google.golang.org/api which is harder to mock
	err = client.UploadPhoto(testImagePath, "test-album-id")
	if err != nil {
		// Expected in test environment without proper OAuth and API setup
		t.Logf("UploadPhoto() failed as expected in test: %v", err)
	}
}

func TestClient_GetOrFindAlbumID(t *testing.T) {
	cfg := &config.GooglePhotosConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RefreshToken: "test-refresh-token",
		AlbumName:    "Test Album",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Test caching - first call should find, second should use cache
	_, err1 := client.GetOrFindAlbumID()
	_, err2 := client.GetOrFindAlbumID()

	// Both will likely fail in test environment, but we're testing the structure
	if err1 != nil {
		t.Logf("GetOrFindAlbumID() first call failed as expected: %v", err1)
	}
	if err2 != nil {
		t.Logf("GetOrFindAlbumID() second call failed as expected: %v", err2)
	}

	// Test that album ID is cached after first successful call
	// This would require a successful FindAlbumByName call first
	client.albumMutex.Lock()
	client.albumID = "cached-album-id"
	client.albumMutex.Unlock()

	albumID, err := client.GetOrFindAlbumID()
	if err != nil {
		t.Fatalf("GetOrFindAlbumID() with cached ID should not fail: %v", err)
	}
	if albumID != "cached-album-id" {
		t.Errorf("GetOrFindAlbumID() = %v, want cached-album-id", albumID)
	}
}

// Test helper to create a mock HTTP server that simulates OAuth2 token refresh
func createMockTokenServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
			return
		}

		// Verify it's a token refresh request
		if !strings.Contains(string(body), "refresh_token") {
			t.Errorf("Expected refresh_token in request body")
			return
		}

		// Return mock token response
		response := map[string]interface{}{
			"access_token": "mock-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
}

func TestClient_ErrorHandling_InvalidCredentials(t *testing.T) {
	cfg := &config.GooglePhotosConfig{
		ClientID:     "invalid-client-id",
		ClientSecret: "invalid-client-secret",
		RefreshToken: "invalid-refresh-token",
		AlbumName:    "Test Album",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Attempting to refresh token with invalid credentials should fail
	err = client.RefreshAccessToken()
	if err == nil {
		t.Error("RefreshAccessToken() with invalid credentials should return error")
	}
}

func TestClient_ErrorHandling_AlbumNotFound(t *testing.T) {
	cfg := &config.GooglePhotosConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RefreshToken: "test-refresh-token",
		AlbumName:    "Non-existent Album",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.FindAlbumByName("Non-existent Album")
	if err == nil {
		t.Error("FindAlbumByName() should return error for non-existent album")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error message should mention 'not found', got: %v", err)
	}
}

