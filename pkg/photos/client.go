package photos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"sync"

	"github.com/jsteffee/icloud-photo-sync/pkg/config"
	"golang.org/x/oauth2"
)

// Client handles Google Photos API interactions
type Client struct {
	config      *config.GooglePhotosConfig
	oauthConfig *oauth2.Config
	httpClient  *http.Client
	ctx         context.Context
	albumID     string
	albumMutex  sync.RWMutex
}

// NewClient creates a new Google Photos client
func NewClient(cfg *config.GooglePhotosConfig) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("GooglePhotosConfig is required")
	}

	oauthConfig := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint: oauth2.Endpoint{
			TokenURL: "https://oauth2.googleapis.com/token",
		},
		Scopes: []string{
			"https://www.googleapis.com/auth/photoslibrary.appendonly",
			"https://www.googleapis.com/auth/photoslibrary.readonly.appcreateddata",
			"https://www.googleapis.com/auth/photoslibrary.edit.appcreateddata",
		},
	}

	ctx := context.Background()
	
	// Create a token with the refresh token - the HTTP client will use this to get access tokens
	token := &oauth2.Token{
		RefreshToken: cfg.RefreshToken,
	}
	
	// Create a reusable token source that will automatically refresh when needed
	tokenSource := oauthConfig.TokenSource(ctx, token)
	httpClient := oauth2.NewClient(ctx, tokenSource)

	return &Client{
		config:      cfg,
		oauthConfig: oauthConfig,
		httpClient:  httpClient,
		ctx:         ctx,
	}, nil
}

// RefreshAccessToken refreshes the OAuth2 access token using the refresh token
// Note: This is typically not needed as the HTTP client automatically refreshes tokens
// This method is provided for manual token refresh if needed
func (c *Client) RefreshAccessToken() error {
	token := &oauth2.Token{
		RefreshToken: c.config.RefreshToken,
	}

	tokenSource := c.oauthConfig.TokenSource(c.ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return fmt.Errorf("failed to refresh access token: %w", err)
	}

	// Update the HTTP client with a new token source using the refreshed token
	c.httpClient = oauth2.NewClient(c.ctx, c.oauthConfig.TokenSource(c.ctx, newToken))
	return nil
}

// albumResponse is used for JSON unmarshaling
type albumResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// CreateAlbum creates a new Google Photos album
func (c *Client) CreateAlbum(albumName string) (string, error) {
	requestBody := map[string]interface{}{
		"album": map[string]string{
			"title": albumName,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(c.ctx, "POST", "https://photoslibrary.googleapis.com/v1/albums", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create album: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create album: status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var albumResponse struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&albumResponse); err != nil {
		return "", fmt.Errorf("failed to decode album response: %w", err)
	}

	// Cache the album ID
	c.albumMutex.Lock()
	c.albumID = albumResponse.ID
	c.albumMutex.Unlock()

	return albumResponse.ID, nil
}

// FindAlbumByName finds a Google Photos album by name (only app-created albums)
// With the new API scopes, we can only access albums created by this app
func (c *Client) FindAlbumByName(albumName string) (string, error) {
	// Check cached album ID first
	c.albumMutex.RLock()
	if c.albumID != "" {
		cachedID := c.albumID
		c.albumMutex.RUnlock()
		return cachedID, nil
	}
	c.albumMutex.RUnlock()

	// The HTTP client will automatically refresh the token if needed
	// With new scopes, we can only list app-created albums
	var nextPageToken string
	for {
		url := "https://photoslibrary.googleapis.com/v1/albums"
		// Filter to only show app-created albums
		if nextPageToken != "" {
			url += "?pageToken=" + nextPageToken + "&excludeNonAppCreatedData=true"
		} else {
			url += "?excludeNonAppCreatedData=true"
		}

		req, err := http.NewRequestWithContext(c.ctx, "GET", url, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to list albums: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return "", fmt.Errorf("failed to list albums: status %d: %s", resp.StatusCode, string(bodyBytes))
		}

		var albumsList struct {
			Albums        []albumResponse `json:"albums"`
			NextPageToken string          `json:"nextPageToken"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&albumsList); err != nil {
			return "", fmt.Errorf("failed to decode albums list: %w", err)
		}

		for _, album := range albumsList.Albums {
			if album.Title == albumName {
				// Cache the album ID
				c.albumMutex.Lock()
				c.albumID = album.ID
				c.albumMutex.Unlock()
				return album.ID, nil
			}
		}

		// Check if there are more pages
		if albumsList.NextPageToken == "" {
			break
		}
		nextPageToken = albumsList.NextPageToken
	}

	return "", fmt.Errorf("album not found: %s (note: with new API scopes, only app-created albums are accessible)", albumName)
}

// GetOrCreateAlbumID gets the album ID, creating it if it doesn't exist
// Returns empty string if AlbumName is not configured (for library-only uploads/partner sharing)
func (c *Client) GetOrCreateAlbumID() (string, error) {
	// If no album name is configured, return empty string (upload to library only)
	if c.config.AlbumName == "" {
		return "", nil
	}

	c.albumMutex.RLock()
	if c.albumID != "" {
		cachedID := c.albumID
		c.albumMutex.RUnlock()
		return cachedID, nil
	}
	c.albumMutex.RUnlock()

	// Try to find the album first
	albumID, err := c.FindAlbumByName(c.config.AlbumName)
	if err == nil {
		return albumID, nil
	}

	// If not found, create it
	log.Printf("Album '%s' not found, creating new album...", c.config.AlbumName)
	return c.CreateAlbum(c.config.AlbumName)
}

// BatchCreateMediaItemsRequest represents the request to create media items
type BatchCreateMediaItemsRequest struct {
	NewMediaItems []NewMediaItem `json:"newMediaItems"`
}

// NewMediaItem represents a new media item to create
type NewMediaItem struct {
	SimpleMediaItem SimpleMediaItem `json:"simpleMediaItem"`
}

// SimpleMediaItem represents a simple media item
type SimpleMediaItem struct {
	UploadToken string `json:"uploadToken"`
}

// BatchCreateMediaItemsResponse represents the response from creating media items
type BatchCreateMediaItemsResponse struct {
	NewMediaItemResults []NewMediaItemResult `json:"newMediaItemResults"`
}

// NewMediaItemResult represents the result of creating a media item
type NewMediaItemResult struct {
	MediaItem *mediaItemResponse `json:"mediaItem"`
	Status    *Status            `json:"status"`
}

// MediaItem represents a Google Photos media item
type MediaItem struct {
	ID string `json:"id"`
}

// mediaItemResponse is used for JSON unmarshaling
type mediaItemResponse struct {
	ID string `json:"id"`
}

// Status represents an API status
type Status struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// BatchAddMediaItemsRequest represents the request to add media items to an album
type BatchAddMediaItemsRequest struct {
	MediaItemIds []string `json:"mediaItemIds"`
}

// UploadPhoto uploads a photo to Google Photos and optionally adds it to an album
// If albumID is empty, the photo is uploaded to the library only (useful for partner sharing)
func (c *Client) UploadPhoto(imagePath string, albumID string) error {
	// The HTTP client will automatically refresh the token if needed
	// Step 1: Upload the media file
	uploadToken, err := c.uploadMedia(imagePath)
	if err != nil {
		return fmt.Errorf("failed to upload media: %w", err)
	}

	// Step 2: Create media item
	mediaItem, err := c.createMediaItem(uploadToken)
	if err != nil {
		return fmt.Errorf("failed to create media item: %w", err)
	}

	// Step 3: Add media item to album (if album ID is provided)
	if albumID != "" {
		if err := c.addMediaItemToAlbum(albumID, mediaItem.ID); err != nil {
			return fmt.Errorf("failed to add media item to album: %w", err)
		}
	}

	return nil
}

// uploadMedia uploads the media file and returns an upload token
func (c *Client) uploadMedia(imagePath string) (string, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info for filename
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}
	fileName := fileInfo.Name()

	// Create multipart form with metadata and file parts
	// Google Photos API requires 2 parts: metadata (JSON) and file data
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Part 1: Metadata (required, must be JSON with Content-Type header)
	metadataHeader := make(textproto.MIMEHeader)
	metadataHeader.Set("Content-Type", "application/json")
	metadataPart, err := writer.CreatePart(metadataHeader)
	if err != nil {
		return "", fmt.Errorf("failed to create metadata part: %w", err)
	}
	_, err = metadataPart.Write([]byte("{}"))
	if err != nil {
		return "", fmt.Errorf("failed to write metadata: %w", err)
	}

	// Part 2: File data (binary with Content-Type header)
	// Reset file position to beginning
	_, err = file.Seek(0, 0)
	if err != nil {
		return "", fmt.Errorf("failed to seek file: %w", err)
	}

	fileHeader := make(textproto.MIMEHeader)
	fileHeader.Set("Content-Type", "application/octet-stream")
	filePart, err := writer.CreatePart(fileHeader)
	if err != nil {
		return "", fmt.Errorf("failed to create file part: %w", err)
	}

	_, err = io.Copy(filePart, file)
	if err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	// Upload to Google Photos
	req, err := http.NewRequestWithContext(c.ctx, "POST", "https://photoslibrary.googleapis.com/v1/uploads", &body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Goog-Upload-Protocol", "multipart")
	req.Header.Set("X-Goog-Upload-File-Name", fileName)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	uploadTokenBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read upload token: %w", err)
	}

	return string(uploadTokenBytes), nil
}

// createMediaItem creates a media item from an upload token
func (c *Client) createMediaItem(uploadToken string) (*MediaItem, error) {
	requestBody := BatchCreateMediaItemsRequest{
		NewMediaItems: []NewMediaItem{
			{
				SimpleMediaItem: SimpleMediaItem{
					UploadToken: uploadToken,
				},
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(c.ctx, "POST", "https://photoslibrary.googleapis.com/v1/mediaItems:batchCreate", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create media item: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create media item: status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var response BatchCreateMediaItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.NewMediaItemResults) == 0 {
		return nil, fmt.Errorf("no media items created")
	}

	result := response.NewMediaItemResults[0]
	if result.Status != nil && result.Status.Code != 0 {
		return nil, fmt.Errorf("media item creation failed: %s", result.Status.Message)
	}

	if result.MediaItem == nil {
		return nil, fmt.Errorf("media item is nil in response")
	}

	return &MediaItem{ID: result.MediaItem.ID}, nil
}

// addMediaItemToAlbum adds a media item to an album
func (c *Client) addMediaItemToAlbum(albumID string, mediaItemID string) error {
	requestBody := BatchAddMediaItemsRequest{
		MediaItemIds: []string{mediaItemID},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("https://photoslibrary.googleapis.com/v1/albums/%s:batchAddMediaItems", albumID)
	req, err := http.NewRequestWithContext(c.ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to add media item to album: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to add media item to album: status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// GetOrFindAlbumID gets the cached album ID or finds it by name
// Deprecated: Use GetOrCreateAlbumID instead for better compatibility with new API scopes
func (c *Client) GetOrFindAlbumID() (string, error) {
	return c.GetOrCreateAlbumID()
}
