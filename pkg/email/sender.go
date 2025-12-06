package email

import (
	"fmt"
	"path/filepath"

	"github.com/jsteffee/icloud-photo-sync/pkg/config"
	"gopkg.in/mail.v2"
)

// Sender handles sending emails with image attachments
type Sender struct {
	smtpConfig *config.SMTPConfig
}

// NewSender creates a new email sender
func NewSender(smtpConfig *config.SMTPConfig) (*Sender, error) {
	return &Sender{
		smtpConfig: smtpConfig,
	}, nil
}

// SendImage sends an email with an image attachment
func (s *Sender) SendImage(imagePath string, destination string) error {
	m := mail.NewMessage()
	m.SetHeader("From", s.smtpConfig.Username)
	m.SetHeader("To", destination)
	m.SetHeader("Subject", "New Photo from iCloud Album")
	m.SetBody("text/plain", "A new photo has been added to the shared album.")

	// Attach the image
	filename := filepath.Base(imagePath)
	m.Attach(imagePath, mail.Rename(filename))

	// Create dialer
	d := mail.NewDialer(s.smtpConfig.Server, s.smtpConfig.Port, s.smtpConfig.Username, s.smtpConfig.Password)
	d.StartTLSPolicy = mail.MandatoryStartTLS

	// Send email
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

