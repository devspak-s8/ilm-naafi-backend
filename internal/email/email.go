package email

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"

	"github.com/ilmnafi/backend/internal/config"
)

type EmailService struct {
	config *config.Config
}

type EmailData struct {
	AppName string
	URL     string
	Token   string
	Email   string
	Name    string
}

func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{config: cfg}
}

func (e *EmailService) Send(to, subject, body string) error {
	switch e.config.Email.Provider {
	case "sendgrid":
		return e.sendSendGrid(to, subject, body)
	case "mailgun":
		return e.sendMailgun(to, subject, body)
	default:
		return e.sendSMTP(to, subject, body)
	}
}

func (e *EmailService) SendVerificationEmail(to, token string) error {
	subject := "Verify your email address"
	tmpl := e.getTemplate("verification")
	body, err := e.renderTemplate(tmpl, EmailData{
		AppName: "Ilm Nafi",
		URL:     fmt.Sprintf("%s/auth/verify-email?token=%s", e.config.AppURL, token),
		Token:   token,
		Email:   to,
	})
	if err != nil {
		return err
	}
	return e.Send(to, subject, body)
}

func (e *EmailService) SendPasswordResetEmail(to, token string) error {
	subject := "Reset your password"
	tmpl := e.getTemplate("reset")
	body, err := e.renderTemplate(tmpl, EmailData{
		AppName: "Ilm Nafi",
		URL:     fmt.Sprintf("%s/auth/reset-password?token=%s", e.config.AppURL, token),
		Token:   token,
		Email:   to,
	})
	if err != nil {
		return err
	}
	return e.Send(to, subject, body)
}

func (e *EmailService) SendSecurityNotification(to, event string) error {
	subject := "Security notification"
	tmpl := e.getTemplate("security")
	body, err := e.renderTemplate(tmpl, EmailData{
		AppName: "Ilm Nafi",
		Email:   to,
	})
	if err != nil {
		return err
	}
	body = strings.ReplaceAll(body, "{{event}}", event)
	return e.Send(to, subject, body)
}

func (e *EmailService) sendSMTP(to, subject, body string) error {
	auth := smtp.PlainAuth("", e.config.Email.SMTPUser, e.config.Email.SMTPPass, e.config.Email.SMTPHost)
	addr := fmt.Sprintf("%s:%d", e.config.Email.SMTPHost, e.config.Email.SMTPPort)

	msg := fmt.Sprintf("From: %s <%s>\r\n", e.config.Email.FromName, e.config.Email.From)
	msg += fmt.Sprintf("To: %s\r\n", to)
	msg += "Subject: " + subject + "\r\n"
	msg += "MIME-version: 1.0\r\n"
	msg += "Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n"
	msg += body

	return smtp.SendMail(addr, auth, e.config.Email.From, []string{to}, []byte(msg))
}

func (e *EmailService) sendSendGrid(to, subject, body string) error {
	return fmt.Errorf("SendGrid provider not yet implemented")
}

func (e *EmailService) sendMailgun(to, subject, body string) error {
	return fmt.Errorf("Mailgun provider not yet implemented")
}

func (e *EmailService) getTemplate(name string) *template.Template {
	// In production, load from files. For now, use inline templates.
	var tmplStr string
	switch name {
	case "verification":
		tmplStr = `<html><body><p>Hello,</p><p>Please verify your email by clicking the link below:</p><a href="{{.URL}}">Verify Email</a><p>This link will expire in 24 hours.</p></body></html>`
	case "reset":
		tmplStr = `<html><body><p>Hello,</p><p>You requested a password reset. Click the link below:</p><a href="{{.URL}}">Reset Password</a><p>This link will expire in 1 hour.</p></body></html>`
	case "security":
		tmplStr = `<html><body><p>Hello,</p><p>A security event occurred: {{event}}</p></body></html>`
	default:
		tmplStr = `<html><body><p>{{.Body}}</p></body></html>`
	}
	return template.Must(template.New(name).Parse(tmplStr))
}

func (e *EmailService) renderTemplate(tmpl *template.Template, data EmailData) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
