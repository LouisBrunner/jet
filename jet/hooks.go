package jet

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/slack-go/slack"
)

type UseStateSetter[T any] func(newValue T) error

type asyncStateData struct {
	TeamID string
	UserID string

	IsHome bool
	// when not home
	ChannelID string
	MessageTS string

	HookID int
}

type UseStateAsyncData[T any] struct {
	P asyncStateData
}

type UseStateSetters[T any] interface {
	SetSync(newValue T) error
	SetAsync(newValue T) error
	GetAsyncData() UseStateAsyncData[T]
}

func UseState[T any](ctx RenderContext, initialValue T) (T, UseStateSetter[T], error) {
	return UseStateFn(ctx, func() T {
		return initialValue
	})
}

func UseStateFn[T any](ctx RenderContext, initializer func() T) (T, UseStateSetter[T], error) {
	value, setters, err := UseStateAdvanced(ctx, initializer)
	if err != nil {
		return value, nil, err
	}
	return value, setters.SetSync, err
}

type useStateSetters[T any] struct {
	ctx  RenderContext
	sync UseStateSetter[T]
	data UseStateAsyncData[T]
}

func (me *useStateSetters[T]) SetSync(newValue T) error {
	return me.sync(newValue)
}

func (me *useStateSetters[T]) SetAsync(newValue T) error {
	return ProcessAsyncData(me.ctx, nil, me.data, newValue)
}

func (me *useStateSetters[T]) GetAsyncData() UseStateAsyncData[T] {
	return me.data
}

func UseStateAdvanced[T any](ctx RenderContext, initializer func() T) (T, UseStateSetters[T], error) {
	var dummyValue T
	id, valueRaw, setRaw, err := ctx.addState(func() (json.RawMessage, error) {
		return json.Marshal(initializer())
	})
	if err != nil {
		return dummyValue, nil, err
	}

	var value T
	err = json.Unmarshal(valueRaw, &value)
	if err != nil {
		return dummyValue, nil, fmt.Errorf("invalid state type: %w", err)
	}
	set := func(newValue T) error {
		newValueRaw, err := json.Marshal(newValue)
		if err != nil {
			return err
		}
		setRaw(newValueRaw)
		return nil
	}

	async := ctx.getAsyncData()
	async.HookID = id
	return value, &useStateSetters[T]{
		ctx:  ctx,
		sync: set,
		data: UseStateAsyncData[T]{
			P: async,
		},
	}, nil
}

type Callback func(ctx context.Context, args slack.BlockAction) error

func UseCallback(ctx RenderContext, callback Callback) (string, error) {
	return ctx.addCallback(callback)
}

func ProcessAsyncData[T any](ctx context.Context, app App, async UseStateAsyncData[T], value T) error {
	valueRaw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return app.handleAsyncData(ctx, async.P, valueRaw)
}
