package jet

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/slack-go/slack"
)

type UseStateSetter[T any] func(newValue T) error

func UseState[T any](ctx RenderContext, initialValue T) (T, UseStateSetter[T], error) {
	valueRaw, setRaw, err := ctx.addState(func() (json.RawMessage, error) {
		return json.Marshal(initialValue)
	})
	if err != nil {
		return initialValue, nil, err
	}

	var value T
	err = json.Unmarshal(valueRaw, &value)
	if err != nil {
		return initialValue, nil, fmt.Errorf("invalid state type: %w", err)
	}
	set := func(newValue T) error {
		newValueRaw, err := json.Marshal(newValue)
		if err != nil {
			return err
		}
		setRaw(newValueRaw)
		return nil
	}
	return value, set, nil
}

type Callback func(ctx context.Context, args slack.BlockAction) error

func UseCallback(ctx RenderContext, callback Callback) (string, error) {
	return ctx.addCallback(callback)
}
