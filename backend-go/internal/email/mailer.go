package email

import (
	"fmt"
	"net/smtp"
	"strings"
)

type loginAuth struct {
	username, password string
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	msg := strings.ToLower(string(fromServer))
	switch {
	case strings.Contains(msg, "username"):
		return []byte(a.username), nil
	case strings.Contains(msg, "password"):
		return []byte(a.password), nil
	default:
		return nil, fmt.Errorf("unexpected server challenge: %s", fromServer)
	}
}

func chooseAuth(cfg Config) smtp.Auth {
	if cfg.Username == "" {
		return nil
	}
	return &loginAuth{cfg.Username, cfg.Password}
}

// Config holds Mailtrap / SMTP settings.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// Mailer sends transactional email via SMTP (Mailtrap in dev).
type Mailer struct {
	cfg Config
}

func NewMailer(cfg Config) *Mailer {
	return &Mailer{cfg: cfg}
}

func (m *Mailer) SendMagicLink(to, link string) error {
	subject := "Sign in to Matchlock"
	body := strings.Join([]string{
		"Sign in to Matchlock",
		"",
		"Click the link below to sign in. This link expires in 15 minutes and can only be used once.",
		"",
		link,
		"",
		"If you did not request this email, you can ignore it.",
	}, "\n")
	return m.send(to, subject, body)
}

func (m *Mailer) SendWagerInvite(to, makerEmail, matchLabel, inviteURL string) error {
	subject := fmt.Sprintf("%s challenged you on Matchlock", makerEmail)
	body := strings.Join([]string{
		fmt.Sprintf("%s sent you a head-to-head challenge.", makerEmail),
		"",
		fmt.Sprintf("Match: %s", matchLabel),
		"",
		"View and accept the challenge:",
		inviteURL,
		"",
		"Sign in with the same email address to see this invite.",
	}, "\n")
	return m.send(to, subject, body)
}

func (m *Mailer) send(to, subject, body string) error {
	if m.cfg.Host == "" {
		return fmt.Errorf("smtp host not configured")
	}
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)

	envelopeFrom := m.cfg.Username
	displayFrom := m.cfg.From
	if envelopeFrom != "" {
		displayFrom = envelopeFrom
	} else if displayFrom == "" {
		displayFrom = "Matchlock <noreply@matchlock.dev>"
		envelopeFrom = "noreply@matchlock.dev"
	}

	msg := strings.Join([]string{
		fmt.Sprintf("From: %s", displayFrom),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	auth := chooseAuth(m.cfg)
	if err := smtp.SendMail(addr, auth, envelopeFrom, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}

func extractEmail(from string) string {
	if i := strings.Index(from, "<"); i >= 0 {
		end := strings.Index(from, ">")
		if end > i {
			return strings.TrimSpace(from[i+1 : end])
		}
	}
	return strings.TrimSpace(from)
}
