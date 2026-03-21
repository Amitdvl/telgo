package telegram

import "time"

// Channel represents a Telegram channel.
type Channel struct {
	ID         int64
	AccessHash int64
	Title      string
	Username   string
}

// Message represents a single Telegram message.
type Message struct {
	ID     int
	Date   time.Time
	Text   string
	Sender string
}
