package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/gotd/td/tg"
)

// FetchMessages fetches the most recent messages from a channel, handling pagination.
// It returns messages in chronological order (oldest first).
func FetchMessages(ctx context.Context, api *tg.Client, ch *Channel, limit int) ([]Message, error) {
	inputPeer := &tg.InputPeerChannel{
		ChannelID:  ch.ID,
		AccessHash: ch.AccessHash,
	}

	var all []Message
	offsetID := 0

	for len(all) < limit {
		batchSize := limit - len(all)
		if batchSize > 100 {
			batchSize = 100
		}

		result, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
			Peer:     inputPeer,
			OffsetID: offsetID,
			Limit:    batchSize,
		})
		if err != nil {
			return nil, fmt.Errorf("get history: %w", err)
		}

		msgs := extractMessages(result)
		if len(msgs) == 0 {
			break
		}

		all = append(all, msgs...)
		// Set offset to the last (oldest) message ID for next page
		offsetID = msgs[len(msgs)-1].ID
	}

	if len(all) > limit {
		all = all[:limit]
	}

	// Reverse to chronological order (oldest first)
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}

	return all, nil
}

func extractMessages(result tg.MessagesMessagesClass) []Message {
	var rawMsgs []tg.MessageClass
	var users []tg.UserClass

	switch v := result.(type) {
	case *tg.MessagesMessages:
		rawMsgs = v.Messages
		users = v.Users
	case *tg.MessagesMessagesSlice:
		rawMsgs = v.Messages
		users = v.Users
	case *tg.MessagesChannelMessages:
		rawMsgs = v.Messages
		users = v.Users
	default:
		return nil
	}

	userMap := buildUserMap(users)

	var msgs []Message
	for _, raw := range rawMsgs {
		msg, ok := raw.(*tg.Message)
		if !ok || msg.Message == "" {
			continue
		}

		sender := resolveSender(msg, userMap)
		msgs = append(msgs, Message{
			ID:     msg.ID,
			Date:   time.Unix(int64(msg.Date), 0),
			Text:   msg.Message,
			Sender: sender,
		})
	}
	return msgs
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
