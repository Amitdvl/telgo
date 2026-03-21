package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/gotd/td/tgerr"
	"github.com/gotd/td/tg"
)

// ListChannels returns all accessible channels/supergroups from the user's dialogs.
func ListChannels(ctx context.Context, api *tg.Client) ([]Channel, error) {
	dialogs, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      200,
	})
	if err != nil {
		return nil, fmt.Errorf("get dialogs: %w", err)
	}

	var chats []tg.ChatClass
	switch d := dialogs.(type) {
	case *tg.MessagesDialogs:
		chats = d.Chats
	case *tg.MessagesDialogsSlice:
		chats = d.Chats
	case *tg.MessagesDialogsNotModified:
		// no changes since last fetch — return empty, not an error
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected dialogs type: %T", dialogs)
	}

	var channels []Channel
	for _, chat := range chats {
		ch, ok := chat.(*tg.Channel)
		if !ok {
			continue
		}
		channels = append(channels, Channel{
			ID:         ch.ID,
			AccessHash: ch.AccessHash,
			Title:      ch.Title,
			Username:   ch.Username,
		})
	}
	return channels, nil
}

// ResolveChannel resolves a channel by username or title substring.
// It first tries username resolution via the API, then falls back to searching dialogs.
func ResolveChannel(ctx context.Context, api *tg.Client, query string) (*Channel, error) {
	// Strip leading @ if present
	query = strings.TrimPrefix(query, "@")

	// Try resolving by username first.
	resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: query,
	})
	if err != nil && !tgerr.IsCode(err, 400) {
		// Propagate real errors (network, auth, flood-wait, etc.).
		// 400 means the username wasn't found or was invalid — fall through to dialog search.
		return nil, fmt.Errorf("resolve username %q: %w", query, err)
	}
	if err == nil {
		for _, chat := range resolved.Chats {
			ch, ok := chat.(*tg.Channel)
			if !ok {
				continue
			}
			return &Channel{
				ID:         ch.ID,
				AccessHash: ch.AccessHash,
				Title:      ch.Title,
				Username:   ch.Username,
			}, nil
		}
	}

	// Fallback: search dialogs by title or username match.
	channels, err := ListChannels(ctx, api)
	if err != nil {
		return nil, fmt.Errorf("resolve channel %q: %w", query, err)
	}

	queryLower := strings.ToLower(query)
	for _, ch := range channels {
		if strings.ToLower(ch.Username) == queryLower || strings.Contains(strings.ToLower(ch.Title), queryLower) {
			return &ch, nil
		}
	}

	return nil, fmt.Errorf("channel %q not found", query)
}
