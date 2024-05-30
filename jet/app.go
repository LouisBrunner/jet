package jet

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/slack-go/slack"
)

type App interface {
	HandleSlashCommand(ctx context.Context, slash slack.SlashCommand) *slack.Msg
	HandleInteraction(ctx context.Context, interaction slack.InteractionCallback) error
	// TODO: select menu
	// TODO: workflow step
	// TODO: bot events (e.g. reaction)
	Options() Options

	FinalizeOAuth(ctx context.Context, code, state string) http.Handler

	LogDebugf(format string, v ...interface{})
	LogErrorf(format string, v ...interface{})
}

type app struct {
	flows            map[FlowHandle]*Flow
	slashes          map[string]SlashCommandHandler
	unknownSlash     SlashCommandHandler
	globalShortcuts  map[string]ShortcutHandler
	messageShortcuts map[string]ShortcutHandler
	unknownShortcut  ShortcutHandler
	opts             Options
}

func (me *app) Options() Options {
	return me.opts
}

func (me *app) HandleSlashCommand(ctx context.Context, slash slack.SlashCommand) *slack.Msg {
	me.LogDebugf("handling slash command: %+v", slash)

	appCtx := &appContext{
		Context: ctx,
		app:     me,
	}

	var res *slack.Msg
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
			res = &slack.Msg{
				ResponseType: slack.ResponseTypeEphemeral,
				Text:         err.Error(),
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
		panic("not implemented") // TODO: finish
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
	meta, err := deserializeMetadata(&interaction.Message.Metadata)
	if err != nil {
		return err
	}
	me.LogDebugf("using meta: %+v", meta)

	flow, ok := me.flows[FlowHandle{
		id: meta.Flow,
	}]
	if !ok {
		return errors.New("unknown flow")
	}

	rctx, err := newRenderContext(ctx, flow.name, meta)
	if err != nil {
		return err
	}
	// first, we populate the render context
	me.LogDebugf("first-pass rendering")
	_, err = flow.renderBlocks(rctx)
	if err != nil {
		return err
	}

	// then, we trigger the callbacks, updating the internal state
	for _, action := range interaction.ActionCallback.BlockActions {
		me.LogDebugf("triggering callback: %s (%+v)", action.ActionID, action)
		err = rctx.triggerCallback(action.ActionID, *action)
		if err != nil {
			return err
		}
	}

	// finally, we render the blocks again
	me.LogDebugf("second-pass rendering")
	msg, err := flow.renderWith(rctx, meta)
	if err != nil {
		return err
	}

	return me.updateMessage(ctx, msg, MessageOptions{
		TeamID:      interaction.Team.ID,
		ResponseURL: interaction.ResponseURL,
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
