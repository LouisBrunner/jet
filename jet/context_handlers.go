package jet

import (
	"context"
	"errors"

	"github.com/slack-go/slack"
)

type Context interface {
	context.Context
	StartFlow(flow *FlowHandle) (*slack.Msg, error)
	StartFlowWithPost(flow *FlowHandle) error
	SlackAPI(teamID string) (*slack.Client, error)
}

type appContext struct {
	context.Context
	app     *app
	msgOpts MessageOptions
}

type MessageOptions struct {
	TeamID string
	// when using {post|update}Message (initial messages/background updates)
	ChannelID string
	MessageTS string
	// when using interactivity
	ResponseURL string
	// when using home
	UserID string
}

func (me *appContext) renderFlow(flow *FlowHandle) (*Flow, *slack.Msg, error) {
	f, ok := me.app.flows[*flow]
	if !ok {
		return nil, nil, errors.New("unknown flow")
	}
	msg, err := f.renderFresh(me.Context)
	return f, msg, err
}

func (me *appContext) StartFlow(flow *FlowHandle) (*slack.Msg, error) {
	f, msg, err := me.renderFlow(flow)
	if err != nil {
		return nil, err
	}
	if f.canUpdateWithoutInteraction() {
		if me.msgOpts.ChannelID == "" {
			return nil, errors.New("missing ChannelID when using CanUpdateWithoutInteraction")
		}
		me.msgOpts.ResponseURL = ""
		_, err := me.app.createMessage(me.Context, msg, me.msgOpts)
		return nil, err
	}
	return msg, nil
}

func (me *appContext) StartFlowWithPost(flow *FlowHandle) error {
	_, msg, err := me.renderFlow(flow)
	if err != nil {
		return err
	}
	_, err = me.app.createMessage(me.Context, msg, me.msgOpts)
	return err
}

func (me *appContext) SlackAPI(teamID string) (*slack.Client, error) {
	return me.app.makeClientFor(teamID)
}
