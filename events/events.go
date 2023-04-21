package events

type EventType string

const (
	ConfigUpdated EventType = "ConfigUpdated"
)

type Event struct {
	Type EventType
	Data any
}

type Sender interface {
	SendEvent(event Event) error
}
