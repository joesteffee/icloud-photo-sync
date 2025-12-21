package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Save original env
	originalEnv := make(map[string]string)
	envVars := []string{
		"REDIS_URL", "SMTP_SERVER", "SMTP_PORT",
		"SMTP_USERNAME", "SMTP_PASSWORD", "SMTP_DESTINATION",
		"RUN_INTERVAL", "MAX_ITEMS", "IMAGE_DIR",
	}
	for _, key := range envVars {
		originalEnv[key] = os.Getenv(key)
		os.Unsetenv(key)
	}
	defer func() {
		for key, value := range originalEnv {
			if value != "" {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	// Create temporary directory for test config files
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		env        map[string]string
		configJSON string
		wantErr    bool
		validate   func(*testing.T, *Config)
	}{
		{
			name: "all required fields",
			env: map[string]string{
				"REDIS_URL":        "redis://localhost:6379",
				"SMTP_SERVER":      "smtp.example.com",
				"SMTP_PORT":        "587",
				"SMTP_USERNAME":    "user@example.com",
				"SMTP_PASSWORD":    "password",
				"SMTP_DESTINATION": "dest@example.com",
				"IMAGE_DIR":        tmpDir,
			},
			configJSON: `{"album_urls": ["https://example.com/album1", "https://example.com/album2"]}`,
			wantErr:    false,
			validate: func(t *testing.T, cfg *Config) {
				if len(cfg.AlbumURLs) != 2 {
					t.Errorf("AlbumURLs length = %v, want 2", len(cfg.AlbumURLs))
				}
				if cfg.AlbumURLs[0] != "https://example.com/album1" {
					t.Errorf("AlbumURLs[0] = %v, want https://example.com/album1", cfg.AlbumURLs[0])
				}
			},
		},
		{
			name: "missing config file",
			env: map[string]string{
				"REDIS_URL":        "redis://localhost:6379",
				"SMTP_SERVER":      "smtp.example.com",
				"SMTP_PORT":        "587",
				"SMTP_USERNAME":    "user@example.com",
				"SMTP_PASSWORD":    "password",
				"SMTP_DESTINATION": "dest@example.com",
				"IMAGE_DIR":        tmpDir,
			},
			configJSON: "",
			wantErr:    true,
		},
		{
			name: "empty album URLs",
			env: map[string]string{
				"REDIS_URL":        "redis://localhost:6379",
				"SMTP_SERVER":      "smtp.example.com",
				"SMTP_PORT":        "587",
				"SMTP_USERNAME":    "user@example.com",
				"SMTP_PASSWORD":    "password",
				"SMTP_DESTINATION": "dest@example.com",
				"IMAGE_DIR":        tmpDir,
			},
			configJSON: `{"album_urls": []}`,
			wantErr:    true,
		},
		{
			name: "with optional fields",
			env: map[string]string{
				"REDIS_URL":        "redis://localhost:6379",
				"SMTP_SERVER":      "smtp.example.com",
				"SMTP_PORT":        "587",
				"SMTP_USERNAME":    "user@example.com",
				"SMTP_PASSWORD":    "password",
				"SMTP_DESTINATION": "dest@example.com",
				"RUN_INTERVAL":     "1800",
				"MAX_ITEMS":        "10",
				"IMAGE_DIR":        tmpDir,
			},
			configJSON: `{"album_urls": ["https://example.com/album"]}`,
			wantErr:    false,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.RunInterval != 1800 {
					t.Errorf("RunInterval = %v, want 1800", cfg.RunInterval)
				}
				if cfg.MaxItems != 10 {
					t.Errorf("MaxItems = %v, want 10", cfg.MaxItems)
				}
			},
		},
		{
			name: "invalid SMTP_PORT",
			env: map[string]string{
				"REDIS_URL":        "redis://localhost:6379",
				"SMTP_SERVER":      "smtp.example.com",
				"SMTP_PORT":        "invalid",
				"SMTP_USERNAME":    "user@example.com",
				"SMTP_PASSWORD":    "password",
				"SMTP_DESTINATION": "dest@example.com",
				"IMAGE_DIR":        tmpDir,
			},
			configJSON: `{"album_urls": ["https://example.com/album"]}`,
			wantErr:    true,
		},
		{
			name: "custom IMAGE_DIR",
			env: map[string]string{
				"REDIS_URL":        "redis://localhost:6379",
				"SMTP_SERVER":      "smtp.example.com",
				"SMTP_PORT":        "587",
				"SMTP_USERNAME":    "user@example.com",
				"SMTP_PASSWORD":    "password",
				"SMTP_DESTINATION": "dest@example.com",
				"IMAGE_DIR":        tmpDir,
			},
			configJSON: `{"album_urls": ["https://example.com/album"]}`,
			wantErr:    false,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.ImageDir != tmpDir {
					t.Errorf("ImageDir = %v, want %v", cfg.ImageDir, tmpDir)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.env {
				os.Setenv(key, value)
			}

			// Set up test directory and config file
			testImageDir := tmpDir
			if dir, ok := tt.env["IMAGE_DIR"]; ok && dir != "" {
				testImageDir = dir
			}
			err := os.MkdirAll(testImageDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}

			configPath := filepath.Join(testImageDir, "config.json")
			
			// Remove config file if it exists (for tests that expect it to be missing)
			if tt.configJSON == "" {
				os.Remove(configPath)
			} else {
				// Create config file if needed
				err = os.WriteFile(configPath, []byte(tt.configJSON), 0644)
				if err != nil {
					t.Fatalf("Failed to write test config file: %v", err)
				}
			}

			cfg, err := Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if cfg == nil {
					t.Fatal("Load() returned nil config")
				}
				if tt.validate != nil {
					tt.validate(t, cfg)
				}
			}

			// Clean up
			for key := range tt.env {
				os.Unsetenv(key)
			}
		})
	}
}
