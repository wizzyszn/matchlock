package email

import (
	"context"
	"fmt"
	"strings"

	brevo "github.com/getbrevo/brevo-go/lib"
)

// Config holds Brevo transactional email settings.
type Config struct {
	APIKey string
	From   string
}

// Mailer sends transactional email via Brevo.
type Mailer struct {
	api *brevo.TransactionalEmailsApiService
	cfg Config
}

func NewMailer(cfg Config) *Mailer {
	clientCfg := brevo.NewConfiguration()
	clientCfg.AddDefaultHeader("api-key", cfg.APIKey)
	client := brevo.NewAPIClient(clientCfg)
	return &Mailer{api: client.TransactionalEmailsApi, cfg: cfg}
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
	if strings.TrimSpace(m.cfg.APIKey) == "" {
		return fmt.Errorf("brevo api key not configured")
	}

	fromName, fromEmail := parseFrom(m.cfg.From)
	if fromEmail == "" {
		fromName = "Matchlock"
		fromEmail = "noreply@matchlock.dev"
	}

	message := brevo.SendSmtpEmail{
		Sender: &brevo.SendSmtpEmailSender{
			Name:  fromName,
			Email: fromEmail,
		},
		To: []brevo.SendSmtpEmailTo{
			{Email: to},
		},
		Subject:     subject,
		TextContent: body,
	}

	if _, _, err := m.api.SendTransacEmail(context.Background(), message); err != nil {
		return fmt.Errorf("brevo send: %w", err)
	}
	return nil
}

func parseFrom(from string) (name, email string) {
	from = strings.TrimSpace(from)
	if from == "" {
		return "", ""
	}
	if i := strings.Index(from, "<"); i >= 0 {
		end := strings.Index(from, ">")
		if end > i {
			name = strings.TrimSpace(from[:i])
			email = strings.TrimSpace(from[i+1 : end])
			name = strings.Trim(name, "\"")
			return name, email
		}
	}
	return "", from
}
