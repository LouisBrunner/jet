package jet

import (
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

type ErrorFormatter func(error) slack.Msg

type Options struct {
	Credentials    Credentials
	Logger         Logger
	ErrorFormatter ErrorFormatter
}
