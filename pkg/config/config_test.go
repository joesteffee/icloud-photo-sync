package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Save original env
	originalEnv := make(map[string]string)
	envVars := []string{
		"ICLOUD_ALBUM_URL", "REDIS_URL", "SMTP_SERVER", "SMTP_PORT",
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

	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
	}{
		{
			name: "all required fields",
			env: map[string]string{
				"ICLOUD_ALBUM_URL":  "https://example.com/album",
				"REDIS_URL":         "redis://localhost:6379",
				"SMTP_SERVER":       "smtp.example.com",
				"SMTP_PORT":         "587",
				"SMTP_USERNAME":     "user@example.com",
				"SMTP_PASSWORD":     "password",
				"SMTP_DESTINATION":  "dest@example.com",
			},
			wantErr: false,
		},
		{
			name: "missing ICLOUD_ALBUM_URL",
			env: map[string]string{
				"REDIS_URL":         "redis://localhost:6379",
				"SMTP_SERVER":       "smtp.example.com",
				"SMTP_PORT":         "587",
				"SMTP_USERNAME":     "user@example.com",
				"SMTP_PASSWORD":     "password",
				"SMTP_DESTINATION":  "dest@example.com",
			},
			wantErr: true,
		},
		{
			name: "with optional fields",
			env: map[string]string{
				"ICLOUD_ALBUM_URL":  "https://example.com/album",
				"REDIS_URL":         "redis://localhost:6379",
				"SMTP_SERVER":       "smtp.example.com",
				"SMTP_PORT":         "587",
				"SMTP_USERNAME":     "user@example.com",
				"SMTP_PASSWORD":     "password",
				"SMTP_DESTINATION":  "dest@example.com",
				"RUN_INTERVAL":      "1800",
				"MAX_ITEMS":         "10",
				"IMAGE_DIR":         "/custom/images",
			},
			wantErr: false,
		},
		{
			name: "invalid SMTP_PORT",
			env: map[string]string{
				"ICLOUD_ALBUM_URL":  "https://example.com/album",
				"REDIS_URL":         "redis://localhost:6379",
				"SMTP_SERVER":       "smtp.example.com",
				"SMTP_PORT":         "invalid",
				"SMTP_USERNAME":     "user@example.com",
				"SMTP_PASSWORD":     "password",
				"SMTP_DESTINATION":  "dest@example.com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.env {
				os.Setenv(key, value)
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
				if cfg.ICloudAlbumURL != tt.env["ICLOUD_ALBUM_URL"] {
					t.Errorf("ICloudAlbumURL = %v, want %v", cfg.ICloudAlbumURL, tt.env["ICLOUD_ALBUM_URL"])
				}
				if tt.env["RUN_INTERVAL"] != "" {
					if cfg.RunInterval != 1800 {
						t.Errorf("RunInterval = %v, want 1800", cfg.RunInterval)
					}
				} else {
					if cfg.RunInterval != 3600 {
						t.Errorf("RunInterval = %v, want 3600", cfg.RunInterval)
					}
				}
			}

			// Clean up
			for key := range tt.env {
				os.Unsetenv(key)
			}
		})
	}
}

