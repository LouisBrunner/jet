package jet

import "github.com/slack-go/slack"

type ViewSubmittedHandler func(ctx Context, args slack.InteractionCallback) error

type ViewSubmitted struct {
	Handler ViewSubmittedHandler
}
