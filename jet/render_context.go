package jet

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/slack-go/slack"
)

type SourceInfo struct {
	TeamID string
	UserID string
}

type RenderContext interface {
	context.Context
	Source() SourceInfo
	addState(initial func() (json.RawMessage, error)) (int, json.RawMessage, func(newValue json.RawMessage), error)
	addCallback(callback Callback) (string, error)
	addEffect(effect Effect) error
	getAsyncData() *asyncStateData
}

type hookData struct {
	kind string
	// for state
	data json.RawMessage
	// for callback
	callback   Callback
	callbackID string
}

type renderContext struct {
	context.Context
	name                string
	isInitial           bool
	hookIdx             int
	expectedHooks       []*hookData
	addedHooks          []*hookData
	pendingStartEffects []Effect
	props               FlowProps
	source              SourceInfo
	async               *asyncStateData
}

func (me *renderContext) Source() SourceInfo {
	return me.source
}

func (me *renderContext) getAsyncData() *asyncStateData {
	return me.async
}

func newRenderContext(ctx context.Context, name string, props FlowProps, metadata *slackMetadataJet, source SourceInfo, async *asyncStateData) (*renderContext, error) {
	var expectedHooks []*hookData
	if metadata != nil {
		expectedHooks = make([]*hookData, len(metadata.Hooks))
		for i, hook := range metadata.Hooks {
			expectedHooks[i] = &hookData{
				kind:       hook.Kind,
				data:       hook.Data,
				callbackID: hook.CallbackID,
			}
		}
		props = metadata.Props
	}
	return &renderContext{
		Context:       ctx,
		name:          name,
		isInitial:     metadata == nil,
		expectedHooks: expectedHooks,
		props:         props,
		source:        source,
		async:         async,
	}, nil
}

type slackMetadataHook struct {
	Kind       string          `json:"k" mapstructure:"k"`
	Data       json.RawMessage `json:"d,omitempty" mapstructure:"d"`
	CallbackID string          `json:"cb,omitempty" mapstructure:"cb"`
}

func (me *renderContext) serializeHooks() []slackMetadataHook {
	hooks := make([]slackMetadataHook, len(me.expectedHooks))
	for i, hook := range me.expectedHooks {
		hooks[i] = slackMetadataHook{
			Kind:       hook.kind,
			Data:       hook.data,
			CallbackID: hook.callbackID,
		}
	}
	return hooks
}

const (
	hookState       = "state"
	hookCallback    = "callback"
	hookEffectStart = "effect-start"
)

func (me *renderContext) addState(initial func() (json.RawMessage, error)) (int, json.RawMessage, func(newValue json.RawMessage), error) {
	id, prev, err := me.fetchHook(hookState)
	if err != nil {
		return 0, nil, nil, err
	}
	if me.isInitial {
		prev.data, err = initial()
		if err != nil {
			return 0, nil, nil, err
		}
	}
	me.addedHooks = append(me.addedHooks, prev)
	return id, prev.data, func(newValue json.RawMessage) {
		prev.data = newValue
	}, nil
}

func (me *renderContext) updateState(idx int, newValue json.RawMessage) error {
	if idx >= len(me.expectedHooks) {
		return fmt.Errorf("unknown state: %d", idx)
	}
	hookData := me.expectedHooks[idx]
	if hookData.kind != hookState {
		return fmt.Errorf("hook %d is not a state", idx)
	}
	me.expectedHooks[idx].data = newValue
	return nil
}

func (me *renderContext) addCallback(callback Callback) (string, error) {
	id, prev, err := me.fetchHook(hookCallback)
	if err != nil {
		return "", err
	}
	prev.callback = callback
	if me.isInitial {
		prev.callbackID = fmt.Sprintf("jet_%s_cb_%x", me.name, id)
	}
	me.addedHooks = append(me.addedHooks, prev)
	return prev.callbackID, nil
}

func (me *renderContext) triggerCallback(callbackID string, action slack.BlockAction) error {
	for _, hook := range me.expectedHooks {
		if hook.kind != hookCallback || hook.callbackID != callbackID {
			continue
		}
		return hook.callback(me, action)
	}
	return fmt.Errorf("unknown callback: %s", callbackID)
}

func (me *renderContext) addEffect(effect Effect) error {
	_, prev, err := me.fetchHook(hookEffectStart)
	if err != nil {
		return err
	}
	me.addedHooks = append(me.addedHooks, prev)
	me.pendingStartEffects = append(me.pendingStartEffects, effect)
	return nil
}

func (me *renderContext) fetchHook(kind string) (int, *hookData, error) {
	currentIdx := me.hookIdx
	me.hookIdx += 1
	if me.isInitial {
		return currentIdx, &hookData{kind: kind}, nil
	}

	if currentIdx >= len(me.expectedHooks) {
		return 0, nil, fmt.Errorf("must use the same amount and type of hooks in all renders, %d vs %d", currentIdx+1, len(me.expectedHooks))
	}
	expected := me.expectedHooks[currentIdx]
	if expected.kind != kind {
		return 0, nil, fmt.Errorf("must use the same amount and type of hooks in all renders, %d is different: %+v vs %+v", currentIdx+1, expected.kind, kind)
	}
	return currentIdx, expected, nil
}

func (me *renderContext) finish() error {
	if !me.isInitial {
		if len(me.expectedHooks) != len(me.addedHooks) {
			expected := make([]string, len(me.expectedHooks))
			for i, hook := range me.expectedHooks {
				expected[i] = hook.kind
			}
			added := make([]string, len(me.addedHooks))
			for i, hook := range me.addedHooks {
				added[i] = hook.kind
			}
			return fmt.Errorf("must use the same amount and type of hooks in all renders: %+v vs %+v", expected, added)
		}
	}
	me.isInitial = false
	me.expectedHooks = me.addedHooks
	me.addedHooks = nil
	me.hookIdx = 0
	return nil
}
