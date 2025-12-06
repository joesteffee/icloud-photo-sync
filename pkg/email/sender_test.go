package email

import (
	"testing"

	"github.com/jsteffee/icloud-photo-sync/pkg/config"
)

func TestNewSender(t *testing.T) {
	smtpConfig := &config.SMTPConfig{
		Server:   "smtp.example.com",
		Port:     587,
		Username: "test@example.com",
		Password: "password",
	}

	sender, err := NewSender(smtpConfig)
	if err != nil {
		t.Fatalf("NewSender() error = %v", err)
	}

	if sender == nil {
		t.Fatal("NewSender() returned nil")
	}

	if sender.smtpConfig != smtpConfig {
		t.Error("NewSender() did not set smtpConfig correctly")
	}
}

// Note: Testing SendImage requires a real SMTP server or a mock
// For unit tests, we would typically use a mock SMTP server
// This is a placeholder that can be expanded with actual SMTP mocking
func TestSender_SendImage(t *testing.T) {
	t.Skip("SendImage test requires SMTP server or mock - implement with test SMTP server")
	
	// Example test structure:
	// 1. Set up mock SMTP server
	// 2. Create sender with mock server config
	// 3. Create test image file
	// 4. Call SendImage
	// 5. Verify email was sent correctly
}

