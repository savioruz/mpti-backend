package mail

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMailService_Templates(t *testing.T) {
	// Set up environment variable for testing
	os.Setenv("APP_URL", "http://localhost:3000")
	defer os.Unsetenv("APP_URL")

	config := Config{
		SMTPHost:     "localhost",
		SMTPPort:     587,
		SMTPUsername: "test@example.com",
		SMTPPassword: "password",
		FromEmail:    "test@example.com",
		FromName:     "Test Service",
		TemplatePath: "../../template", // Relative to the test directory
	}

	mailService := New(config)

	t.Run("templates are loaded correctly", func(t *testing.T) {
		s := mailService.(*service)

		require.NotNil(t, s.verificationTemplate)
		require.NotNil(t, s.passwordResetTemplate)
	})
}
