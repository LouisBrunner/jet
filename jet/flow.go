package jet

import "github.com/slack-go/slack"

type FlowHandle struct {
	id string
}

type Flow interface {
	CanUpdateWithoutInteraction() bool
	Render(ctx Context) (*slack.Msg, error)
}

type FlowRenderer func(ctx Context) (*slack.Blocks, error)

type FlowOptions struct {
	CanUpdateWithoutInteraction bool
}

type flow struct {
	canUpdateWithoutInteraction bool
	render                      FlowRenderer
}

func NewFlow(render FlowRenderer, opts ...FlowOptions) Flow {
	opt := FlowOptions{}
	if len(opts) > 0 {
		opt = opts[0]
	}
	return &flow{
		canUpdateWithoutInteraction: opt.CanUpdateWithoutInteraction,
		render:                      render,
	}
}

func (me *flow) CanUpdateWithoutInteraction() bool {
	return me.canUpdateWithoutInteraction
}

func (me *flow) Render(ctx Context) (*slack.Msg, error) {
	blocks, err := me.render(ctx)
	if err != nil {
		return nil, err
	}
	return &slack.Msg{
		Type:            slack.ResponseTypeInChannel,
		ReplaceOriginal: true,
		Blocks:          *blocks,
	}, nil
}
