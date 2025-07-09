package mail

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"gopkg.in/gomail.v2"
)

//go:generate go run go.uber.org/mock/mockgen -source=mail.go -destination=mock/mail_mock.go -package=mock github.com/savioruz/goth/pkg/mail Interface

type Config struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
	TemplatePath string // Path to the templates directory
}

type Service interface {
	SendVerificationEmail(to, name, token string) error
	SendPasswordResetEmail(to, name, token string) error
}

type service struct {
	config                Config
	verificationTemplate  *template.Template
	passwordResetTemplate *template.Template
}

func New(config Config) Service {
	// Default template path if not provided
	templatePath := config.TemplatePath
	if templatePath == "" {
		templatePath = "template"
	}

	// Parse templates
	verificationTemplate, err := template.ParseFiles(filepath.Join(templatePath, "email_verification.html"))
	if err != nil {
		panic(fmt.Sprintf("failed to parse email verification template: %v", err))
	}

	passwordResetTemplate, err := template.ParseFiles(filepath.Join(templatePath, "password_reset.html"))
	if err != nil {
		panic(fmt.Sprintf("failed to parse password reset template: %v", err))
	}

	return &service{
		config:                config,
		verificationTemplate:  verificationTemplate,
		passwordResetTemplate: passwordResetTemplate,
	}
}

func (s *service) SendVerificationEmail(to, name, token string) error {
	subject := "Verify Your Email Address"
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", os.Getenv("APP_URL"), token)

	// Template data
	data := struct {
		Name      string
		VerifyURL string
	}{
		Name:      name,
		VerifyURL: verifyURL,
	}

	// Execute template
	var body bytes.Buffer
	if err := s.verificationTemplate.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute email verification template: %w", err)
	}

	return s.sendEmail(to, subject, body.String())
}

func (s *service) SendPasswordResetEmail(to, name, token string) error {
	subject := "Reset Your Password"
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", os.Getenv("APP_URL"), token)

	// Template data
	data := struct {
		Name     string
		ResetURL string
	}{
		Name:     name,
		ResetURL: resetURL,
	}

	// Execute template
	var body bytes.Buffer
	if err := s.passwordResetTemplate.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute password reset template: %w", err)
	}

	return s.sendEmail(to, subject, body.String())
}

func (s *service) sendEmail(to, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromEmail))
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(s.config.SMTPHost, s.config.SMTPPort, s.config.SMTPUsername, s.config.SMTPPassword)

	return d.DialAndSend(m)
}
