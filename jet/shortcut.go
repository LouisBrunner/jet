package jet

import "github.com/slack-go/slack"

type ShortcutHandler func(ctx Context, args slack.InteractionCallback) error

type Shortcut struct {
	Handler ShortcutHandler
}
