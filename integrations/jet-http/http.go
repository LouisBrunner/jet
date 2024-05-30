package jethttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/LouisBrunner/jet/integrations/common"
	"github.com/LouisBrunner/jet/jet"
	"github.com/slack-go/slack"
)

func verifyRequest(app jet.App, w http.ResponseWriter, r *http.Request) bool {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return false
	}
	app.LogDebugf("body: %+v", string(body))
	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	sv, err := slack.NewSecretsVerifier(r.Header, app.Options().Credentials.SigningSecret)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return false
	}

	if _, err := sv.Write(body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return false
	}

	if err := sv.Ensure(); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}

	return true
}

type Handlers common.Handlers[http.Handler]

func New(app jet.App) Handlers {
	return Handlers{
		SlashCommands: handleSlashCommands(app),
		Interactivity: handleInteractivity(app),
		SelectMenus:   handleSelectMenus(app),
		OAuth:         handleOAuth(app),
	}
}

func handleSlashCommands(app jet.App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.LogDebugf("slash command: %+v", r.Header)
		if !verifyRequest(app, w, r) {
			return
		}

		args, err := slack.SlashCommandParse(r)
		if err != nil {
			app.LogErrorf("failed to parse slash command: %+v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		res := app.HandleSlashCommand(r.Context(), args)
		if res == nil {
			app.LogDebugf("slash response: nil")
			w.WriteHeader(http.StatusOK)
			return
		}
		app.LogDebugf("slash response: %+v", res)

		json, err := json.Marshal(res)
		if err != nil {
			app.LogErrorf("failed to marshal slash response: %+v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		app.LogDebugf("slash response json: %+v", string(json))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(json))
		if err != nil {
			app.LogErrorf("failed to write slash response: %+v", err)
			return
		}
	})
}

func handleInteractivity(app jet.App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.LogDebugf("interactivity: %+v", r.Header)
		if !verifyRequest(app, w, r) {
			return
		}

		payload := r.PostFormValue("payload")
		if payload == "" {
			app.LogErrorf("missing payload in interactivity request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var args slack.InteractionCallback
		err := json.Unmarshal([]byte(payload), &args)
		if err != nil {
			app.LogErrorf("failed to unmarshal interactivity payload: %+v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = app.HandleInteraction(r.Context(), args)
		if err != nil {
			app.LogErrorf("failed to handle interactivity: %+v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}

func handleSelectMenus(app jet.App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.LogDebugf("select menu: %+v", r.Header)
		if !verifyRequest(app, w, r) {
			return
		}

		// TODO: finish
		a, _ := io.ReadAll(r.Body)
		fmt.Printf("select menu: %+v\n", r.Header)
		fmt.Printf("select menu: %+v\n", string(a))
	})
}

func handleOAuth(app jet.App) http.Handler {
	if app.Options().OAuthConfig == nil {
		return nil
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.LogDebugf("oauth: %+v", r.Header)

		state := r.URL.Query().Get("state")
		code := r.URL.Query().Get("code")

		app.FinalizeOAuth(r.Context(), code, state).ServeHTTP(w, r)
	})
}
