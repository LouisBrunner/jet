package jet

import (
	"net/http"

	"github.com/slack-go/slack"
)

type AccessTokenRetriever func(teamID string) (string, error)

type Credentials struct {
	// The signing secret used to verify incoming webhooks.
	// This can be found in your app settings page.
	SigningSecret string

	// This function is used to request the access token for a given team.
	// This will be called every time a request is made to a particular team.
	// It is your job to cache the token for future use if you want to.
	// If you return an error, the request will be aborted.
	GetAccessToken AccessTokenRetriever
}

type OAuthSuccessData struct {
	State       string
	TeamID      string
	AccessToken string
}

type OAuthSuccessHandler func(data OAuthSuccessData) error

type OAuthRenderError func(err error) http.Handler

type OAuthConfig struct {
	// used to exchange the temporary code for an access token
	ClientID string
	// same as above
	ClientSecret string

	// called if the OAuth flow is successful
	OnSuccess OAuthSuccessHandler

	// render the page that will be displayed to the user after the OAuth flow is successful
	RenderSuccessPage http.Handler
	// render the page that will be displayed to the user if the OAuth flow fails for any reason
	RenderErrorPage OAuthRenderError
}

type ErrorFormatter func(error) slack.Msg

type Options struct {
	Credentials    Credentials
	OAuthConfig    *OAuthConfig
	Logger         Logger
	ErrorFormatter ErrorFormatter
}
