package jet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/slack-go/slack"
)

func (me *app) makeClientFor(teamID string) (*slack.Client, error) {
	if me.opts.Credentials.GetAccessToken == nil {
		return nil, errors.New("missing GetAccessToken function")
	}
	token, err := me.opts.Credentials.GetAccessToken(teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token for %q: %w", teamID, err)
	}
	return slack.New(token), nil
}

func prepareMessage(msg *slack.Msg, in MessageOptions) []slack.MsgOption {
	options := []slack.MsgOption{
		slack.MsgOptionBlocks(msg.Blocks.BlockSet...),
		slack.MsgOptionMetadata(msg.Metadata),
		slack.MsgOptionText(msg.Text, false),
	}

	if msg.ReplaceOriginal {
		options = append(options, slack.MsgOptionReplaceOriginal(in.ResponseURL))
	}
	if msg.DeleteOriginal {
		options = append(options, slack.MsgOptionDeleteOriginal(in.ResponseURL))
	}
	options = append(options, slack.MsgOptionResponseURL(in.ResponseURL, msg.ResponseType))

	if in.ResponseURL == "" {
		options = append(options, slack.MsgOptionPost())
	}

	return options
}

func (me *app) createMessage(ctx context.Context, msg *slack.Msg, in MessageOptions) (string, error) {
	client, err := me.makeClientFor(in.TeamID)
	if err != nil {
		return "", err
	}

	me.LogDebugf("creating message: %+v", msg)
	_, ts, err := client.PostMessageContext(ctx, in.ChannelID,
		prepareMessage(msg, in)...,
	)
	return ts, err
}

func (me *app) updateMessage(ctx context.Context, msg *slack.Msg, in MessageOptions) error {
	client, err := me.makeClientFor(in.TeamID)
	if err != nil {
		return err
	}

	me.LogDebugf("updating message: %+v", msg)
	_, _, _, err = client.UpdateMessageContext(ctx, in.ChannelID, in.MessageTS,
		prepareMessage(msg, in)...,
	)
	return err
}

func (me *app) publishView(ctx context.Context, msg *slack.Msg, in MessageOptions) error {
	client, err := me.makeClientFor(in.TeamID)
	if err != nil {
		return err
	}

	meta, err := json.Marshal(msg.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	me.LogDebugf("publishing view: %+v", msg)
	_, err = client.PublishViewContext(ctx, in.UserID, slack.HomeTabViewRequest{
		Type:            slack.VTHomeTab,
		Blocks:          msg.Blocks,
		PrivateMetadata: string(meta),
	}, "")
	return err
}

func (me *app) tokenExchange(ctx context.Context, code, clientID, clientSecret string) (*slack.OAuthV2Response, error) {
	resp, err := slack.GetOAuthV2Response(http.DefaultClient, clientID, clientSecret, code, "")
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}
	if !resp.Ok {
		return nil, fmt.Errorf("failed to exchange code for token: %s", resp.Error)
	}
	return resp, nil
}
