package jet

import (
	"context"
	"errors"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/slack-go/slack"
)

type structLike any

type FlowProps map[string]interface{}

func UnmarshalProps[T structLike](props FlowProps, data T) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:  data,
		TagName: "json",
	})
	if err != nil {
		return err
	}
	return decoder.Decode(props)
}

func MarshalProps[T structLike](data T) (FlowProps, error) {
	var props FlowProps
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:  &props,
		TagName: "json",
	})
	if err != nil {
		return nil, err
	}
	err = decoder.Decode(data)
	if err != nil {
		return nil, err
	}
	return props, err
}

type Context interface {
	context.Context
	StartFlow(flow *FlowHandle, props FlowProps) (*slack.Msg, error)
	StartFlowAndPost(flow *FlowHandle, props FlowProps) error
	App() App
}

type appContext struct {
	context.Context
	app     *app
	msgOpts messageOptions
	source  SourceInfo
	isHome  bool
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

func (me *appContext) renderFlow(flow *FlowHandle, props FlowProps) (*Flow, *slack.Msg, postCreateFlowFn, error) {
	f, ok := me.app.flows[*flow]
	if !ok {
		return nil, nil, nil, errors.New("unknown flow")
	}
	msg, post, err := f.renderFresh(me.Context, props, me.source, me.msgOpts, me.isHome)
	return f, msg, post, err
}

func (me *appContext) StartFlow(flow *FlowHandle, props FlowProps) (*slack.Msg, error) {
	f, msg, post, err := me.renderFlow(flow, props)
	if err != nil {
		return nil, err
	}
	if f.canUpdateWithoutInteraction() {
		if me.msgOpts.ChannelID == "" {
			return nil, errors.New("missing ChannelID when using CanUpdateWithoutInteraction")
		}
		me.msgOpts.ResponseURL = ""
		return nil, me.createWithPost(f, msg, post)
	} else if post != nil {
		return nil, fmt.Errorf("cannot use UseEffectAtStart without CanUpdateWithoutInteraction")
	}
	return msg, nil
}

func (me *appContext) createWithPost(f *Flow, msg *slack.Msg, post postCreateFlowFn) error {
	ts, err := me.app.createMessage(me.Context, msg, me.msgOpts)
	if err != nil {
		return err
	}
	if post != nil {
		go func() {
			err := me.processPostFlow(context.Background(), post, msg, &asyncStateData{
				TeamID:    me.msgOpts.TeamID,
				UserID:    me.msgOpts.UserID,
				ChannelID: me.msgOpts.ChannelID,
				MessageTS: ts,
			})
			if err != nil {
				me.app.LogErrorf("failed to process post flow: %v", err)
			}
		}()
	}
	return nil
}

func (me *appContext) processPostFlow(ctx context.Context, post postCreateFlowFn, orig *slack.Msg, async *asyncStateData) error {
	meta, err := deserializeMetadata(&orig.Metadata, "")
	if err != nil {
		return err
	}
	msg, err := post(ctx, meta, async)
	if err != nil {
		return err
	}
	return me.app.updateMessage(ctx, msg, messageOptions{
		TeamID:    async.TeamID,
		ChannelID: async.ChannelID,
		MessageTS: async.MessageTS,
	})
}

func (me *appContext) StartFlowAndPost(flow *FlowHandle, props FlowProps) error {
	f, msg, post, err := me.renderFlow(flow, props)
	if err != nil {
		return err
	}
	return me.createWithPost(f, msg, post)
}

func (me *appContext) App() App {
	return me.app
}

func StartFlow[T structLike](ctx Context, flow *FlowHandle, props T) (*slack.Msg, error) {
	propsMap, err := MarshalProps(props)
	if err != nil {
		return nil, err
	}
	return ctx.StartFlow(flow, propsMap)
}

func StartFlowAndPost[T structLike](ctx Context, flow *FlowHandle, props T) error {
	propsMap, err := MarshalProps(props)
	if err != nil {
		return err
	}
	return ctx.StartFlowAndPost(flow, propsMap)
}
