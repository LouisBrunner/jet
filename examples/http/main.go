package main

import (
	"net/http"
	"os"

	jethttp "github.com/LouisBrunner/slack-jet/integrations/jet-http"
	"github.com/LouisBrunner/slack-jet/jet"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

func exampleFlow(ctx jet.Context) (*slack.Blocks, error) {
	return &slack.Blocks{
		BlockSet: []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", "Hello, world!", false, false),
				nil,
				nil,
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
	f1, err := builder.AddFlow(jet.NewFlow(exampleFlow))
	if err != nil {
		return err
	}
	app := builder.
		AddSlash("/test-jet", func(ctx jet.Context, args slack.SlashCommand) (*slack.Msg, error) {
			return ctx.StartFlow(f1, jet.MessageOptions{
				TeamID:      args.TeamID,
				ChannelID:   args.ChannelID,
				ResponseURL: args.ResponseURL,
			})
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
