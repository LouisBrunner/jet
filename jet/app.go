package jet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/slack-go/slack"
)

type HomeUpdater = func(ctx Context) (*Message, error)

type App interface {
	HandleSlashCommand(ctx context.Context, slash slack.SlashCommand) *Message
	HandleInteraction(ctx context.Context, interaction slack.InteractionCallback) error
	// TODO: select menu
	// TODO: workflow step
	// TODO: bot events (e.g. reaction)
	UpdateHome(ctx context.Context, workspaceID, userID string, updater HomeUpdater) error
	SlackAPI(teamID string) (*slack.Client, error)
	Options() Options

	FinalizeOAuth(ctx context.Context, code, state string) http.Handler

	LogDebugf(format string, v ...interface{})
	LogErrorf(format string, v ...interface{})

	handleAsyncData(ctx context.Context, data asyncStateData, value json.RawMessage) error
}

type app struct {
	flows            map[FlowHandle]*Flow
	slashes          map[string]SlashCommandHandler
	unknownSlash     SlashCommandHandler
	globalShortcuts  map[string]ShortcutHandler
	messageShortcuts map[string]ShortcutHandler
	viewSubmitted    map[string]ViewSubmittedHandler
	unknownShortcut  ShortcutHandler
	opts             Options
}

func (me *app) Options() Options {
	return me.opts
}

func (me *app) HandleSlashCommand(ctx context.Context, slash slack.SlashCommand) *Message {
	me.LogDebugf("handling slash command: %+v", slash)

	appCtx := &appContext{
		Context: ctx,
		app:     me,
		msgOpts: messageOptions{
			TeamID:      slash.TeamID,
			ChannelID:   slash.ChannelID,
			ResponseURL: slash.ResponseURL,
		},
		source: SourceInfo{
			TeamID: slash.TeamID,
			UserID: slash.UserID,
		},
	}

	var res *Message
	var err error
	cmd, ok := me.slashes[slash.Command]
	if !ok {
		if me.unknownSlash != nil {
			res, err = me.unknownSlash(appCtx, slash)
		} else {
			err = errors.New("unknown command")
		}
	} else {
		res, err = cmd(appCtx, slash)
	}

	if err != nil {
		if me.opts.ErrorFormatter != nil {
			msg := me.opts.ErrorFormatter(err)
			res = &msg
		} else {
			res = &Message{
				Msg: slack.Msg{
					ResponseType: slack.ResponseTypeEphemeral,
					Text:         err.Error(),
				},
			}
		}
	}

	return res
}

func (me *app) HandleInteraction(ctx context.Context, interaction slack.InteractionCallback) error {
	me.LogDebugf("handling interaction: %+v", interaction)
	switch interaction.Type {
	case slack.InteractionTypeDialogCancellation:
		panic("not implemented") // TODO: finish
	case slack.InteractionTypeDialogSubmission:
		panic("not implemented") // TODO: finish
	case slack.InteractionTypeDialogSuggestion:
		panic("not implemented") // TODO: finish
	case slack.InteractionTypeInteractionMessage:
		panic("not implemented") // TODO: finish
	case slack.InteractionTypeMessageAction:
		return me.handleShortcut(ctx, me.messageShortcuts, interaction)
	case slack.InteractionTypeBlockActions:
		return me.handleBlockActions(ctx, interaction)
	case slack.InteractionTypeBlockSuggestion:
		panic("not implemented") // TODO: finish
	case slack.InteractionTypeViewSubmission:
		return me.handleViewSubmission(ctx, interaction)
	case slack.InteractionTypeViewClosed:
		panic("not implemented") // TODO: finish
	case slack.InteractionTypeShortcut:
		return me.handleShortcut(ctx, me.globalShortcuts, interaction)
	case slack.InteractionTypeWorkflowStepEdit:
		panic("not implemented") // TODO: finish
	default:
		return errors.New("unknown interaction type")
	}
}

func (me *app) handleShortcut(ctx context.Context, shortcuts map[string]ShortcutHandler, interaction slack.InteractionCallback) error {
	appCtx := &appContext{
		Context: ctx,
		app:     me,
		msgOpts: messageOptions{
			TeamID:      interaction.Team.ID,
			ResponseURL: interaction.ResponseURL,
		},
		source: SourceInfo{
			TeamID: interaction.Team.ID,
			UserID: interaction.User.ID,
		},
	}

	var err error
	cmd, ok := shortcuts[interaction.CallbackID]
	if !ok {
		if me.unknownShortcut != nil {
			err = me.unknownShortcut(appCtx, interaction)
		} else {
			err = errors.New("unknown shortcut")
		}
	} else {
		err = cmd(appCtx, interaction)
	}
	return err
}

func (me *app) handleBlockActions(ctx context.Context, interaction slack.InteractionCallback) error {
	meta, err := deserializeMetadata(&interaction.Message.Metadata, interaction.View.PrivateMetadata)
	if err != nil {
		return err
	}

	src := SourceInfo{
		TeamID: interaction.Team.ID,
		UserID: interaction.User.ID,
	}

	return me.multiStageRender(ctx, multiStageOptions{
		meta:   meta,
		src:    src,
		isHome: interaction.View.Type == slack.VTHomeTab,
		msgOpts: messageOptions{
			TeamID:      interaction.Team.ID,
			ResponseURL: interaction.ResponseURL,
		},
		async: asyncStateData{
			ChannelID: interaction.Channel.ID,
			MessageTS: interaction.Message.Timestamp,
		},
		betweenStages: func(rctx *renderContext) error {
			for _, action := range interaction.ActionCallback.BlockActions {
				me.LogDebugf("triggering callback: %s (%+v)", action.ActionID, action)
				err = rctx.triggerCallback(action.ActionID, *action)
				if err != nil {
					return err
				}
			}
			return nil
		},
	})
}

func (me *app) handleViewSubmission(ctx context.Context, interaction slack.InteractionCallback) error {
	meta, err := deserializeMetadata(&interaction.Message.Metadata, interaction.View.PrivateMetadata)
	if err != nil {
		return err
	}

	handler, found := me.viewSubmitted[meta.Flow]
	if !found {
		return errors.New("unknown view submission")
	}

	url := interaction.ResponseURL
	channelID := ""
	for _, action := range interaction.ResponseURLs {
		url = action.ResponseURL
		channelID = action.ChannelID
		break
	}

	return handler(&appContext{
		Context: ctx,
		app:     me,
		msgOpts: messageOptions{
			TeamID:      interaction.Team.ID,
			ChannelID:   channelID,
			ResponseURL: url,
		},
		source: SourceInfo{
			TeamID: interaction.Team.ID,
			UserID: interaction.User.ID,
		},
	}, interaction)
}

type multiStageOptions struct {
	meta          *slackMetadataJet
	src           SourceInfo
	isHome        bool
	msgOpts       messageOptions
	async         asyncStateData
	betweenStages func(rctx *renderContext) error
}

func (me *app) multiStageRender(ctx context.Context, opts multiStageOptions) error {
	me.LogDebugf("using meta: %+v", opts.meta)

	flow, ok := me.flows[FlowHandle{
		id: opts.meta.Flow,
	}]
	if !ok {
		return errors.New("unknown flow")
	}

	msg, err := flow.multiStageRender(ctx, opts.meta, opts.src, &asyncStateData{
		TeamID:      opts.src.TeamID,
		UserID:      opts.src.UserID,
		IsHome:      opts.isHome,
		ChannelID:   opts.async.ChannelID,
		MessageTS:   opts.async.MessageTS,
		ResponseURL: opts.async.ResponseURL,
		Metadata:    opts.async.Metadata,
	}, opts.betweenStages)
	if err != nil {
		return err
	}

	if opts.isHome {
		return me.publishView(ctx, &msg.Msg, messageOptions{
			TeamID: opts.src.TeamID,
			UserID: opts.src.UserID,
		})
	}
	return me.updateMessage(ctx, &msg.Msg, opts.msgOpts)
}

func (me *app) handleAsyncData(ctx context.Context, data asyncStateData, value json.RawMessage) error {
	me.LogDebugf("handling async data: %+v", data)

	var meta *slackMetadataJet
	msg, err := me.getMessage(ctx, data.TeamID, data.ChannelID, data.MessageTS)
	if err != nil {
		if data.Metadata != nil {
			meta, err = deserializeMetadata(data.Metadata, "")
			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		meta, err = deserializeMetadata(&msg.Metadata, "")
		if err != nil {
			return err
		}
	}

	return me.multiStageRender(ctx, multiStageOptions{
		meta: meta,
		src: SourceInfo{
			TeamID: data.TeamID,
			UserID: data.UserID,
		},
		isHome: data.IsHome,
		msgOpts: messageOptions{
			TeamID:      data.TeamID,
			ChannelID:   data.ChannelID,
			MessageTS:   data.MessageTS,
			ResponseURL: data.ResponseURL,
		},
		async: data,
		betweenStages: func(rctx *renderContext) error {
			return rctx.updateState(data.HookID, value)
		},
	})
}

func (me *app) FinalizeOAuth(ctx context.Context, code, state string) http.Handler {
	cfg := me.opts.OAuthConfig

	if cfg == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotImplemented)
			w.Write([]byte("OAuth is not configured correctly, please provide `OAuthConfig` to `jet.NewBuilder().Build()`"))
		})
	}

	if code == "" {
		err := fmt.Errorf("missing code in oauth request (slack issue)")
		me.LogErrorf("failed to handle oauth: %+v", err)
		return cfg.RenderErrorPage(err)
	}

	resp, err := me.tokenExchange(ctx, code, cfg.ClientID, cfg.ClientSecret)
	if err != nil {
		me.LogErrorf("failed to exchange code for token: %+v", err)
		return cfg.RenderErrorPage(err)
	}

	err = cfg.OnSuccess(OAuthSuccessData{
		TeamID:      resp.Team.ID,
		AccessToken: resp.AccessToken,
		State:       state,
	})
	if err != nil {
		me.LogErrorf("failed to handle oauth: %+v", err)
		return cfg.RenderErrorPage(err)
	}

	return cfg.RenderSuccessPage
}

func (me *app) UpdateHome(ctx context.Context, workspaceID, userID string, updater HomeUpdater) error {
	appCtx := &appContext{
		Context: ctx,
		app:     me,
		msgOpts: messageOptions{
			TeamID: workspaceID,
			UserID: userID,
		},
		source: SourceInfo{
			TeamID: workspaceID,
			UserID: userID,
		},
		isHome: true,
	}

	msg, err := updater(appCtx)
	if err != nil {
		return err
	}
	if msg == nil {
		return fmt.Errorf("cannot update Home with a CanUpdateWithoutInteraction flow")
	}

	return me.publishView(ctx, &msg.Msg, appCtx.msgOpts)
}

func (me *app) SlackAPI(teamID string) (*slack.Client, error) {
	return me.makeClientFor(teamID)
}
