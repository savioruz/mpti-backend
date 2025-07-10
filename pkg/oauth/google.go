package oauth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/valyala/fasthttp"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleProvider struct {
	config *oauth2.Config
}

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

func NewGoogleProvider(clientID, clientSecret, redirectURL string) *GoogleProvider {
	return &GoogleProvider{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
	}
}

func (p *GoogleProvider) GetAuthURL(state string) string {
	return p.config.AuthCodeURL(state)
}

func (p *GoogleProvider) Exchange(code string) (*oauth2.Token, error) {
	token, err := p.config.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange authorization code: %w", err)
	}

	return token, nil
}

func (p *GoogleProvider) GetUserInfo(token *oauth2.Token) (*GoogleUserInfo, error) {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	err := fasthttp.Do(req, resp)

	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(resp.Body(), &userInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user info: %w", err)
	}

	return &userInfo, nil
}
