package jet

import (
	"context"
	"errors"

	"github.com/slack-go/slack"
)

type FlowProps map[string]interface{}

type Context interface {
	context.Context
	StartFlow(flow *FlowHandle, props FlowProps) (*slack.Msg, error)
	StartFlowAndPost(flow *FlowHandle, props FlowProps) error
	SlackAPI(teamID string) (*slack.Client, error)
}

type appContext struct {
	context.Context
	app     *app
	msgOpts messageOptions
	source  SourceInfo
}

type messageOptions struct {
	TeamID string
	// when using {post|update}Message (initial messages/background updates)
	ChannelID string
	MessageTS string
	// when using interactivity
	ResponseURL string
	// when using home
	UserID string
}

func (me *appContext) renderFlow(flow *FlowHandle, props FlowProps) (*Flow, *slack.Msg, error) {
	f, ok := me.app.flows[*flow]
	if !ok {
		return nil, nil, errors.New("unknown flow")
	}
	msg, err := f.renderFresh(me.Context, props, me.source)
	return f, msg, err
}

func (me *appContext) StartFlow(flow *FlowHandle, props FlowProps) (*slack.Msg, error) {
	f, msg, err := me.renderFlow(flow, props)
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

func (me *appContext) StartFlowAndPost(flow *FlowHandle, props FlowProps) error {
	_, msg, err := me.renderFlow(flow, props)
	if err != nil {
		return err
	}
	_, err = me.app.createMessage(me.Context, msg, me.msgOpts)
	return err
}

func (me *appContext) SlackAPI(teamID string) (*slack.Client, error) {
	return me.app.makeClientFor(teamID)
}
