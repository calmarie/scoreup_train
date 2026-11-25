package outbox

import "time"

type Event struct {
	ID          int64
	Topic       string
	EventKey    string
	Payload     []byte
	PublishedAt *time.Time
}
