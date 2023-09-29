package jet

import (
	"context"
	"errors"

	"github.com/slack-go/slack"
)

type App interface {
	HandleSlashCommand(ctx context.Context, slash slack.SlashCommand) *slack.Msg
	HandleInteraction(ctx context.Context, interaction slack.InteractionCallback) error
	// TODO: select menu
	// TODO: workflow step
	// TODO: bot events (e.g. reaction)
	Options() Options

	LogDebugf(format string, v ...interface{})
	LogErrorf(format string, v ...interface{})
}

type app struct {
	flows            map[*FlowHandle]Flow
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
		panic("not implemented") // TODO: finish
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
