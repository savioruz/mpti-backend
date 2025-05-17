package oauth

import (
	"golang.org/x/oauth2"
)

//go:generate go run go.uber.org/mock/mockgen -source=google_provider.go -destination=mock/google_mock.go -package=mock github.com/savioruz/goth/pkg/oauth Interface

// GoogleProviderIface defines the methods that a GoogleProvider must implement
type GoogleProviderIface interface {
	GetAuthURL() string
	Exchange(code string) (*oauth2.Token, error)
	GetUserInfo(token *oauth2.Token) (*GoogleUserInfo, error)
}
