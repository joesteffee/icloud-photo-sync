package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// SMTPConfig holds SMTP configuration
type SMTPConfig struct {
	Server   string
	Port     int
	Username string
	Password string
	From     string // Optional "From" email address (defaults to Username if not set)
}

// GooglePhotosConfig holds Google Photos API configuration
type GooglePhotosConfig struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	AlbumName    string
}

// AlbumConfig represents the configuration file structure
type AlbumConfig struct {
	AlbumURLs []string `json:"album_urls"`
}

// Config holds all application configuration
type Config struct {
	AlbumURLs         []string
	RedisURL          string
	SMTPConfig        *SMTPConfig
	SMTPDestination   string
	GooglePhotosConfig *GooglePhotosConfig // Optional - nil if not configured
	RunInterval       int
	MaxItems          int
	ImageDir          string
}

// Load loads configuration from environment variables and config file
func Load() (*Config, error) {
	cfg := &Config{}

	// Get image directory (default: /images)
	imageDir := os.Getenv("IMAGE_DIR")
	if imageDir == "" {
		imageDir = "/images" // Default: /images
	}
	cfg.ImageDir = imageDir

	// Load album URLs from config file
	configPath := filepath.Join(imageDir, "config.json")
	albumConfig, err := loadAlbumConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load album config from %s: %w", configPath, err)
	}
	if len(albumConfig.AlbumURLs) == 0 {
		return nil, fmt.Errorf("no album URLs found in config file at %s", configPath)
	}
	cfg.AlbumURLs = albumConfig.AlbumURLs

	cfg.RedisURL = os.Getenv("REDIS_URL")
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("REDIS_URL is required")
	}

	smtpServer := os.Getenv("SMTP_SERVER")
	if smtpServer == "" {
		return nil, fmt.Errorf("SMTP_SERVER is required")
	}

	smtpPortStr := os.Getenv("SMTP_PORT")
	if smtpPortStr == "" {
		return nil, fmt.Errorf("SMTP_PORT is required")
	}
	smtpPort, err := strconv.Atoi(smtpPortStr)
	if err != nil {
		return nil, fmt.Errorf("SMTP_PORT must be a valid integer: %v", err)
	}

	smtpUsername := os.Getenv("SMTP_USERNAME")
	if smtpUsername == "" {
		return nil, fmt.Errorf("SMTP_USERNAME is required")
	}

	smtpPassword := os.Getenv("SMTP_PASSWORD")
	if smtpPassword == "" {
		return nil, fmt.Errorf("SMTP_PASSWORD is required")
	}

	// Optional SMTP_FROM environment variable
	smtpFrom := os.Getenv("SMTP_FROM")
	if smtpFrom == "" {
		smtpFrom = smtpUsername // Default to username if not specified
	}

	cfg.SMTPConfig = &SMTPConfig{
		Server:   smtpServer,
		Port:     smtpPort,
		Username: smtpUsername,
		Password: smtpPassword,
		From:     smtpFrom,
	}

	cfg.SMTPDestination = os.Getenv("SMTP_DESTINATION")
	if cfg.SMTPDestination == "" {
		return nil, fmt.Errorf("SMTP_DESTINATION is required")
	}

	// Optional variables with defaults
	runIntervalStr := os.Getenv("RUN_INTERVAL")
	if runIntervalStr == "" {
		cfg.RunInterval = 3600 // Default: 1 hour
	} else {
		runInterval, err := strconv.Atoi(runIntervalStr)
		if err != nil {
			return nil, fmt.Errorf("RUN_INTERVAL must be a valid integer: %v", err)
		}
		cfg.RunInterval = runInterval
	}

	maxItemsStr := os.Getenv("MAX_ITEMS")
	if maxItemsStr == "" {
		cfg.MaxItems = 5 // Default: 5 items
	} else {
		maxItems, err := strconv.Atoi(maxItemsStr)
		if err != nil {
			return nil, fmt.Errorf("MAX_ITEMS must be a valid integer: %v", err)
		}
		cfg.MaxItems = maxItems
	}

	// Google Photos configuration (optional - only enabled if all vars are provided)
	googlePhotosClientID := os.Getenv("GOOGLE_PHOTOS_CLIENT_ID")
	googlePhotosClientSecret := os.Getenv("GOOGLE_PHOTOS_CLIENT_SECRET")
	googlePhotosRefreshToken := os.Getenv("GOOGLE_PHOTOS_REFRESH_TOKEN")
	googlePhotosAlbumName := os.Getenv("GOOGLE_PHOTOS_ALBUM_NAME")

	// If any Google Photos env var is set, all must be set
	if googlePhotosClientID != "" || googlePhotosClientSecret != "" || googlePhotosRefreshToken != "" || googlePhotosAlbumName != "" {
		if googlePhotosClientID == "" {
			return nil, fmt.Errorf("GOOGLE_PHOTOS_CLIENT_ID is required when Google Photos is enabled")
		}
		if googlePhotosClientSecret == "" {
			return nil, fmt.Errorf("GOOGLE_PHOTOS_CLIENT_SECRET is required when Google Photos is enabled")
		}
		if googlePhotosRefreshToken == "" {
			return nil, fmt.Errorf("GOOGLE_PHOTOS_REFRESH_TOKEN is required when Google Photos is enabled")
		}
		if googlePhotosAlbumName == "" {
			return nil, fmt.Errorf("GOOGLE_PHOTOS_ALBUM_NAME is required when Google Photos is enabled")
		}

		cfg.GooglePhotosConfig = &GooglePhotosConfig{
			ClientID:     googlePhotosClientID,
			ClientSecret: googlePhotosClientSecret,
			RefreshToken: googlePhotosRefreshToken,
			AlbumName:    googlePhotosAlbumName,
		}
	}

	return cfg, nil
}

// loadAlbumConfig loads the album configuration from a JSON file
func loadAlbumConfig(configPath string) (*AlbumConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var albumConfig AlbumConfig
	if err := json.Unmarshal(data, &albumConfig); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &albumConfig, nil
}

