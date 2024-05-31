package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	jethttp "github.com/LouisBrunner/jet/integrations/jet-http"
	"github.com/LouisBrunner/jet/jet"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

func exampleFlow(ctx jet.RenderContext, props jet.FlowProps) (*slack.Blocks, error) {
	value, setValue, err := jet.UseState(ctx, 0)
	if err != nil {
		return nil, err
	}
	callback, err := jet.UseCallback(ctx, func(ctx context.Context, args slack.BlockAction) error {
		return setValue(value + 1)
	})
	if err != nil {
		return nil, err
	}

	return &slack.Blocks{
		BlockSet: []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Counter: %d", value), false, false),
				nil,
				slack.NewAccessory(
					slack.NewButtonBlockElement(
						callback,
						"Click Me",
						slack.NewTextBlockObject("plain_text", "Click Me", false, false),
					),
				),
			),
		},
	}, nil
}

func work() error {
	err := godotenv.Load()
	if err != nil {
		return err
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	builder := jet.NewBuilder()
	f1, err := builder.AddFlow(jet.NewFlow("hello", exampleFlow, nil))
	if err != nil {
		return err
	}
	app := builder.
		AddSlash("/test-jet", func(ctx jet.Context, args slack.SlashCommand) (*slack.Msg, error) {
			return ctx.StartFlow(f1, nil)
		}).
		AddGlobalShortcut("jet_global", func(ctx jet.Context, args slack.InteractionCallback) error {
			panic("modal")
		}).
		AddMessageShortcut("jet_message", func(ctx jet.Context, args slack.InteractionCallback) error {
			panic("modal")
		}).
		Build(jet.Options{
			Credentials: jet.Credentials{
				SigningSecret: os.Getenv("SLACK_SIGNING_SECRET"),
				GetAccessToken: func(teamID string) (string, error) {
					return os.Getenv("SLACK_ACCESS_TOKEN"), nil
				},
			},
			Logger: logger,
		})

	handlers := jethttp.New(app)
	http.Handle("/slack", handlers.SlashCommands)
	http.Handle("/slack-interactive", handlers.Interactivity)
	http.Handle("/slack-select", handlers.SelectMenus)
	return http.ListenAndServe("localhost:8080", nil)
}

func main() {
	err := work()
	if err != nil {
		panic(err)
	}
}
