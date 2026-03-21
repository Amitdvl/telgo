package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/gotd/td/tg"
)

// FetchMessages fetches the most recent messages from a channel, handling pagination.
// It returns messages in chronological order (oldest first).
// Pagination uses raw message IDs so that batches consisting entirely of
// service/empty messages do not cause premature termination.
func FetchMessages(ctx context.Context, api *tg.Client, ch *Channel, limit int) ([]Message, error) {
	inputPeer := &tg.InputPeerChannel{
		ChannelID:  ch.ID,
		AccessHash: ch.AccessHash,
	}

	var all []Message
	offsetID := 0

	for len(all) < limit {
		result, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
			Peer:     inputPeer,
			OffsetID: offsetID,
			Limit:    100,
		})
		if err != nil {
			return nil, fmt.Errorf("get history: %w", err)
		}

		raw, users := extractRaw(result)
		if len(raw) == 0 {
			break // no more messages in channel
		}

		// Always advance offset using the oldest raw message ID,
		// regardless of whether it carried text or not.
		offsetID = rawLastID(raw)

		userMap := buildUserMap(users)
		for _, msgClass := range raw {
			msg, ok := msgClass.(*tg.Message)
			if !ok || msg.Message == "" {
				continue
			}
			all = append(all, Message{
				ID:     msg.ID,
				Date:   time.Unix(int64(msg.Date), 0),
				Text:   msg.Message,
				Sender: resolveSender(msg, userMap),
			})
		}

		if len(raw) < 100 {
			break // reached the end of channel history
		}
	}

	if len(all) > limit {
		all = all[:limit]
	}

	// Reverse to chronological order (oldest first).
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}

	return all, nil
}

// extractRaw unpacks the union type returned by MessagesGetHistory.
func extractRaw(result tg.MessagesMessagesClass) ([]tg.MessageClass, []tg.UserClass) {
	switch v := result.(type) {
	case *tg.MessagesMessages:
		return v.Messages, v.Users
	case *tg.MessagesMessagesSlice:
		return v.Messages, v.Users
	case *tg.MessagesChannelMessages:
		return v.Messages, v.Users
	default:
		return nil, nil
	}
}

// rawLastID returns the ID of the last (oldest) message in a raw batch.
func rawLastID(msgs []tg.MessageClass) int {
	if len(msgs) == 0 {
		return 0
	}
	switch m := msgs[len(msgs)-1].(type) {
	case *tg.Message:
		return m.ID
	case *tg.MessageService:
		return m.ID
	case *tg.MessageEmpty:
		return m.ID
	}
	return 0
}

func buildUserMap(users []tg.UserClass) map[int64]string {
	m := make(map[int64]string)
	for _, u := range users {
		user, ok := u.(*tg.User)
		if !ok {
			continue
		}
		name := user.FirstName
		if user.LastName != "" {
			name += " " + user.LastName
		}
		if name == "" {
			name = user.Username
		}
		m[user.ID] = name
	}
	return m
}

func resolveSender(msg *tg.Message, userMap map[int64]string) string {
	if msg.FromID == nil {
		return "channel"
	}
	switch p := msg.FromID.(type) {
	case *tg.PeerUser:
		if name, ok := userMap[p.UserID]; ok {
			return name
		}
		return fmt.Sprintf("user:%d", p.UserID)
	case *tg.PeerChannel:
		return fmt.Sprintf("channel:%d", p.ChannelID)
	default:
		return "unknown"
	}
}
