package common

type Handlers[T any] struct {
	SlashCommands T
	Interactivity T
	SelectMenus   T
}
