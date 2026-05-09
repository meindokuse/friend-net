package event

// Event is a domain event to be published to a message bus.
type Event struct {
	Topic   string
	Key     string
	Payload []byte
	Headers map[string]string
}

// Events is a slice of Event.
type Events []*Event
