package email

import (
	"crypto/tls"
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
	
	// Some SMTP servers (like ProtonMail Bridge) require the From address to match
	// the authenticated username. Use username as From, but set Reply-To if custom From is specified.
	fromAddr := s.smtpConfig.Username
	replyToAddr := s.smtpConfig.From
	if replyToAddr == "" {
		replyToAddr = s.smtpConfig.Username
	}
	
	// Set From header to authenticated username (required by some SMTP servers)
	m.SetHeader("From", fromAddr)
	// Set Reply-To to the desired sender address if different
	if replyToAddr != fromAddr {
		m.SetHeader("Reply-To", replyToAddr)
	}
	m.SetHeader("To", destination)
	m.SetHeader("Subject", "New Photo from iCloud Album")
	m.SetBody("text/plain", "A new photo has been added to the shared album.")

	// Attach the image
	filename := filepath.Base(imagePath)
	m.Attach(imagePath, mail.Rename(filename))

	// Create dialer
	d := mail.NewDialer(s.smtpConfig.Server, s.smtpConfig.Port, s.smtpConfig.Username, s.smtpConfig.Password)
	
	// Skip certificate verification for self-signed or mismatched certificates
	// This is common with local SMTP servers like ProtonMail Bridge
	d.TLSConfig = &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         s.smtpConfig.Server,
	}

	// For port 25, ProtonMail Bridge typically requires STARTTLS for authentication
	// Try MandatoryStartTLS first (required for authentication on port 25)
	if s.smtpConfig.Port == 25 {
		d.StartTLSPolicy = mail.MandatoryStartTLS
	} else {
		// For other ports, try opportunistic STARTTLS
		d.StartTLSPolicy = mail.OpportunisticStartTLS
	}

	// Send email
	if err := d.DialAndSend(m); err != nil {
		// If MandatoryStartTLS fails on port 25, try OpportunisticStartTLS as fallback
		if s.smtpConfig.Port == 25 && d.StartTLSPolicy == mail.MandatoryStartTLS {
			d.StartTLSPolicy = mail.OpportunisticStartTLS
			if err2 := d.DialAndSend(m); err2 != nil {
				return fmt.Errorf("failed to send email on port 25 (tried MandatoryStartTLS and OpportunisticStartTLS): %w (original: %v)", err2, err)
			}
		} else {
			return fmt.Errorf("failed to send email: %w", err)
		}
	}

	return nil
}

