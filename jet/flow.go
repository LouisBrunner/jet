package jet

import (
	"context"
	"maps"

	"github.com/slack-go/slack"
)

type FlowHandle struct {
	id string
}

type FlowRenderer func(ctx RenderContext, props FlowProps) (*RenderedFlow, error)

type FlowOptions struct {
	CanUpdateWithoutInteraction bool
}

type Flow struct {
	name                           string
	canUpdateWithoutInteractionOpt bool
	renderFn                       FlowRenderer
}

type RenderedFlow struct {
	Blocks   slack.Blocks
	Text     string
	Metadata *slack.SlackMetadata
}

func NewFlow(name string, render FlowRenderer, opts *FlowOptions) Flow {
	opt := FlowOptions{}
	if opts != nil {
		opt = *opts
	}
	return Flow{
		name:                           name,
		canUpdateWithoutInteractionOpt: opt.CanUpdateWithoutInteraction,
		renderFn:                       render,
	}
}

func (me *Flow) canUpdateWithoutInteraction() bool {
	return me.canUpdateWithoutInteractionOpt
}

func (me *Flow) renderFresh(ctx context.Context, props FlowProps, source SourceInfo, msgOpts messageOptions, isHome bool) (*slack.Msg, error) {
	rctx, err := newRenderContext(ctx, me.name, props, nil, source, asyncStateData{
		TeamID:    source.TeamID,
		UserID:    source.UserID,
		IsHome:    isHome,
		ChannelID: msgOpts.ChannelID,
		MessageTS: msgOpts.MessageTS,
	})
	if err != nil {
		return nil, err
	}
	return me.renderWith(rctx, nil)
}

func (me *Flow) renderWith(rctx *renderContext, metadata *slackMetadataJet) (*slack.Msg, error) {
	rendered, err := me.renderBlocks(rctx)
	if err != nil {
		return nil, err
	}
	var finalMetadata *slack.SlackMetadata
	if metadata != nil {
		finalMetadata = &metadata.Original
	}
	if rendered.Metadata != nil {
		if finalMetadata == nil {
			finalMetadata = &slack.SlackMetadata{}
		}

		if rendered.Metadata.EventType != "" {
			finalMetadata.EventType = rendered.Metadata.EventType
		}
		if rendered.Metadata.EventPayload != nil {
			for k, v := range rendered.Metadata.EventPayload {
				finalMetadata.EventPayload[k] = v
			}
		}
	}
	return &slack.Msg{
		ResponseType:    slack.ResponseTypeInChannel,
		ReplaceOriginal: true,
		Text:            rendered.Text,
		Blocks:          rendered.Blocks,
		Metadata:        serializeMetadata(finalMetadata, me.name, rctx),
	}, nil
}

func (me *Flow) renderBlocks(rctx *renderContext) (*RenderedFlow, error) {
	props := maps.Clone(rctx.props)
	if props == nil {
		props = make(FlowProps)
	}
	rendered, err := me.renderFn(rctx, props)
	if err != nil {
		return nil, err
	}
	err = rctx.finish()
	if err != nil {
		return nil, err
	}
	return rendered, nil
}
