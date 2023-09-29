package jet

import "github.com/slack-go/slack"

type SlashCommandHandler func(ctx Context, args slack.SlashCommand) (*slack.Msg, error)

type SlashCommand struct {
	Handler SlashCommandHandler
}
