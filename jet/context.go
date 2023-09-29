package jet

import (
	"context"
	"errors"

	"github.com/slack-go/slack"
)

type Context interface {
	context.Context
	StartFlow(flow *FlowHandle, in MessageOptions) (*slack.Msg, error)
	StartFlowWithPost(flow *FlowHandle, in MessageOptions) error
}

type appContext struct {
	context.Context
	app *app
}

type MessageOptions struct {
	TeamID      string
	ChannelID   string
	ResponseURL string
}

func (me *appContext) renderFlow(flow *FlowHandle) (Flow, *slack.Msg, error) {
	f, ok := me.app.flows[flow]
	if !ok {
		return nil, nil, errors.New("unknown flow")
	}
	msg, err := f.Render(me)
	return f, msg, err
}

func (me *appContext) StartFlow(flow *FlowHandle, in MessageOptions) (*slack.Msg, error) {
	f, msg, err := me.renderFlow(flow)
	if err != nil {
		return nil, err
	}
	if f.CanUpdateWithoutInteraction() {
		if in.ChannelID == "" {
			return nil, errors.New("missing ChannelID when using CanUpdateWithoutInteraction")
		}
		in.ResponseURL = ""
		return nil, me.app.createMessage(me.Context, msg, in)
	}
	return msg, nil
}

func (me *appContext) StartFlowWithPost(flow *FlowHandle, in MessageOptions) error {
	_, msg, err := me.renderFlow(flow)
	if err != nil {
		return err
	}
	return me.app.createMessage(me.Context, msg, in)
}
