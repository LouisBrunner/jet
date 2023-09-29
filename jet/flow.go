package jet

import (
	"context"

	"github.com/slack-go/slack"
)

type FlowHandle struct {
	id string
}

type FlowRenderer func(ctx RenderContext) (*slack.Blocks, error)

type FlowOptions struct {
	CanUpdateWithoutInteraction bool
}

type Flow struct {
	name                           string
	canUpdateWithoutInteractionOpt bool
	renderFn                       FlowRenderer
}

func NewFlow(name string, render FlowRenderer, opts ...FlowOptions) Flow {
	opt := FlowOptions{}
	if len(opts) > 0 {
		opt = opts[0]
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

func (me *Flow) renderFresh(ctx context.Context) (*slack.Msg, error) {
	rctx, err := newRenderContext(ctx, me.name, nil)
	if err != nil {
		return nil, err
	}
	return me.renderWith(rctx, nil)
}

func (me *Flow) renderWith(rctx *renderContext, metadata *slackMetadataJet) (*slack.Msg, error) {
	blocks, err := me.renderBlocks(rctx)
	if err != nil {
		return nil, err
	}
	return &slack.Msg{
		ResponseType:    slack.ResponseTypeInChannel,
		ReplaceOriginal: true,
		Blocks:          *blocks,
		Metadata:        serializeMetadata(metadata, me.name, rctx.serialize()),
	}, nil
}

func (me *Flow) renderBlocks(rctx *renderContext) (*slack.Blocks, error) {
	blocks, err := me.renderFn(rctx)
	if err != nil {
		return nil, err
	}
	err = rctx.finish()
	if err != nil {
		return nil, err
	}
	return blocks, nil
}
