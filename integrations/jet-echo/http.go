package jetecho

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/LouisBrunner/jet/integrations/common"
	"github.com/LouisBrunner/jet/jet"
	"github.com/labstack/echo/v4"
	"github.com/slack-go/slack"
)

func signingVerifyMiddleware(app jet.App) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			body, err := io.ReadAll(c.Request().Body)
			if err != nil {
				return c.String(http.StatusBadRequest, "")
			}

			app.LogDebugf("body: %+v", string(body))
			c.Request().Body.Close()
			c.Request().Body = io.NopCloser(bytes.NewBuffer(body))

			sv, err := slack.NewSecretsVerifier(c.Request().Header, app.Options().Credentials.SigningSecret)
			if err != nil {
				return c.String(http.StatusBadRequest, "")
			}

			if _, err := sv.Write(body); err != nil {
				return c.String(http.StatusInternalServerError, "")
			}

			if err := sv.Ensure(); err != nil {
				return c.String(http.StatusUnauthorized, "")
			}

			return next(c)
		}
	}
}

type EchoRoutes interface {
	GET(path string, handler echo.HandlerFunc, middleware ...echo.MiddlewareFunc) *echo.Route
	POST(path string, handler echo.HandlerFunc, middleware ...echo.MiddlewareFunc) *echo.Route
}

type EchoAdder func(e EchoRoutes, path string, middlewars ...echo.MiddlewareFunc) *echo.Route

type Handlers common.Handlers[EchoAdder]

func New(app jet.App) Handlers {
	middlewares := []echo.MiddlewareFunc{
		signingVerifyMiddleware(app),
	}

	return Handlers{
		SlashCommands: handleSlashCommands(app, middlewares),
		Interactivity: handleInteractivity(app, middlewares),
		SelectMenus:   handleSelectMenus(app, middlewares),
		OAuth:         handleOAuth(app),
	}
}

func handleSlashCommands(app jet.App, middlewares []echo.MiddlewareFunc) EchoAdder {
	return func(e EchoRoutes, path string, theirMiddlewares ...echo.MiddlewareFunc) *echo.Route {
		return e.POST(path, func(c echo.Context) error {
			app.LogDebugf("slash command: %+v", c)

			args, err := slack.SlashCommandParse(c.Request())
			if err != nil {
				app.LogErrorf("failed to parse slash command: %+v", err)
				return c.String(http.StatusBadRequest, "")
			}

			res := app.HandleSlashCommand(c.Request().Context(), args)
			if res == nil {
				app.LogDebugf("slash response: nil")
				return c.String(http.StatusOK, "")
			}

			app.LogDebugf("slash response: %+v", res)
			return c.JSON(http.StatusOK, res)
		}, append(theirMiddlewares, middlewares...)...)
	}
}

func handleInteractivity(app jet.App, middlewares []echo.MiddlewareFunc) EchoAdder {
	return func(e EchoRoutes, path string, theirMiddlewares ...echo.MiddlewareFunc) *echo.Route {
		return e.POST(path, func(c echo.Context) error {
			app.LogDebugf("interactivity: %+v", c)

			payload := c.FormValue("payload")
			if payload == "" {
				app.LogErrorf("missing payload in interactivity request")
				return c.String(http.StatusBadRequest, "")
			}

			var args slack.InteractionCallback
			err := json.Unmarshal([]byte(payload), &args)
			if err != nil {
				app.LogErrorf("failed to unmarshal interactivity payload: %+v", err)
				return c.String(http.StatusBadRequest, "")
			}

			err = app.HandleInteraction(c.Request().Context(), args)
			if err != nil {
				app.LogErrorf("failed to handle interactivity: %+v", err)
				return c.String(http.StatusInternalServerError, "")
			}

			return c.String(http.StatusOK, "")
		}, append(theirMiddlewares, middlewares...)...)
	}
}

func handleSelectMenus(app jet.App, middlewares []echo.MiddlewareFunc) EchoAdder {
	return func(e EchoRoutes, path string, theirMiddlewares ...echo.MiddlewareFunc) *echo.Route {
		return e.POST(path, func(c echo.Context) error {
			app.LogDebugf("select menu: %+v", c)

			// TODO: finish
			a, _ := io.ReadAll(c.Request().Body)
			fmt.Printf("select menu: %+v\n", string(a))
			return fmt.Errorf("not implemented")
		}, append(theirMiddlewares, middlewares...)...)
	}
}

func handleOAuth(app jet.App) EchoAdder {
	if app.Options().OAuthConfig == nil {
		return nil
	}

	return func(e EchoRoutes, path string, middlewares ...echo.MiddlewareFunc) *echo.Route {
		return e.GET(path, func(c echo.Context) error {
			app.LogDebugf("oauth: %+v", c)

			state := c.QueryParam("state")
			code := c.QueryParam("code")

			handler := app.FinalizeOAuth(c.Request().Context(), code, state)
			handler.ServeHTTP(c.Response(), c.Request())
			return nil
		}, middlewares...)
	}
}
