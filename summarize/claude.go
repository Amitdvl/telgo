package summarize

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/amitdvl/telgo/telegram"
)

// Summarizer uses Claude to summarize Telegram channel messages.
type Summarizer struct {
	client anthropic.Client
}

// New creates a Summarizer. If apiKey is empty, it uses ANTHROPIC_API_KEY env var.
func New(apiKey string) *Summarizer {
	var opts []option.RequestOption
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	return &Summarizer{
		client: anthropic.NewClient(opts...),
	}
}

// Summarize generates a structured summary of channel messages using Claude.
func (s *Summarizer) Summarize(ctx context.Context, channelTitle string, messages []telegram.Message) (string, error) {
	if len(messages) == 0 {
		return "No messages to summarize.", nil
	}

	prompt := formatMessages(channelTitle, messages)

	resp, err := s.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6,
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("claude API: %w", err)
	}

	var result strings.Builder
	for _, block := range resp.Content {
		if block.Type == "text" {
			result.WriteString(block.Text)
		}
	}
	return result.String(), nil
}

const systemPrompt = `You are a Telegram channel summarizer for a busy professional.
Given messages from a Telegram channel, produce a concise, structured summary.

Guidelines:
- Lead with the most important/actionable information
- Group by topic/theme, not chronologically
- Use bullet points for clarity
- Call out: key announcements, decisions, action items, deadlines, links to important resources
- Note the time range of messages covered
- Skip trivial messages (greetings, reactions, off-topic chatter)
- If there are recurring themes, note their frequency
- Keep it concise but don't miss anything important`

func formatMessages(channelTitle string, messages []telegram.Message) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Channel: %s\n", channelTitle)
	fmt.Fprintf(&b, "Messages: %d (from %s to %s)\n\n",
		len(messages),
		messages[0].Date.Format("2006-01-02 15:04"),
		messages[len(messages)-1].Date.Format("2006-01-02 15:04"),
	)

	for _, msg := range messages {
		fmt.Fprintf(&b, "[%s] %s: %s\n",
			msg.Date.Format("Jan 02 15:04"),
			msg.Sender,
			msg.Text,
		)
	}
	return b.String()
}
