package jet

import (
	"fmt"
)

var ErrDuplicateFlowHandle = fmt.Errorf("duplicate flow name")

type AppBuilder interface {
	AddFlow(flow Flow) (*FlowHandle, error)
	AddSlash(name string, handler SlashCommandHandler) AppBuilder
	HandleUnknownSlash(handler SlashCommandHandler) AppBuilder
	AddGlobalShortcut(name string, handler ShortcutHandler) AppBuilder
	AddMessageShortcut(name string, handler ShortcutHandler) AppBuilder
	HandleUnknownShortcut(handler ShortcutHandler) AppBuilder

	Build(opts Options) App
}

type appBuilder struct {
	flows            map[FlowHandle]*Flow
	slashes          map[string]SlashCommandHandler
	unknownSlash     SlashCommandHandler
	globalShortcuts  map[string]ShortcutHandler
	messageShortcuts map[string]ShortcutHandler
	unknownShortcut  ShortcutHandler
}

func NewBuilder() AppBuilder {
	return &appBuilder{
		flows:            make(map[FlowHandle]*Flow),
		slashes:          make(map[string]SlashCommandHandler),
		globalShortcuts:  make(map[string]ShortcutHandler),
		messageShortcuts: make(map[string]ShortcutHandler),
	}
}

func (me *appBuilder) AddFlow(f Flow) (*FlowHandle, error) {
	fh := FlowHandle{
		id: f.name,
	}
	_, found := me.flows[fh]
	if found {
		return nil, ErrDuplicateFlowHandle
	}
	me.flows[fh] = &f
	return &fh, nil
}

func (me *appBuilder) AddSlash(cmd string, handler SlashCommandHandler) AppBuilder {
	me.slashes[cmd] = handler
	return me
}

func (me *appBuilder) HandleUnknownSlash(handler SlashCommandHandler) AppBuilder {
	me.unknownSlash = handler
	return me
}

func (me *appBuilder) AddGlobalShortcut(cmd string, handler ShortcutHandler) AppBuilder {
	me.globalShortcuts[cmd] = handler
	return me
}

func (me *appBuilder) AddMessageShortcut(cmd string, handler ShortcutHandler) AppBuilder {
	me.messageShortcuts[cmd] = handler
	return me
}

func (me *appBuilder) HandleUnknownShortcut(handler ShortcutHandler) AppBuilder {
	me.unknownShortcut = handler
	return me
}

func (me *appBuilder) Build(opts Options) App {
	return &app{
		flows:            me.flows,
		slashes:          me.slashes,
		unknownSlash:     me.unknownSlash,
		globalShortcuts:  me.globalShortcuts,
		messageShortcuts: me.messageShortcuts,
		unknownShortcut:  me.unknownShortcut,
		opts:             opts,
	}
}
