package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"text/tabwriter"

	gotelegram "github.com/gotd/td/telegram"
	"github.com/gotd/td/session"

	"github.com/amitdvl/telgo/config"
	"github.com/amitdvl/telgo/summarize"
	"github.com/amitdvl/telgo/telegram"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	if cmd == "help" || cmd == "-h" || cmd == "--help" {
		printUsage()
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	switch cmd {
	case "auth":
		err = runAuth(ctx, cfg)
	case "channels":
		err = runChannels(ctx, cfg)
	case "read":
		err = runRead(ctx, cfg, os.Args[2:])
	case "summarize":
		err = runSummarize(ctx, cfg, os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`telgo - Telegram channel reader & summarizer

Usage: telgo <command> [options]

Commands:
  auth                  Authenticate with Telegram (interactive)
  channels              List accessible channels
  read <channel>        Read messages from a channel
  summarize <channel>   Summarize messages from a channel

Options for read/summarize:
  -limit N    Number of messages to fetch (default: 200)

Environment:
  TELEGRAM_APP_ID       Telegram API app ID (from my.telegram.org)
  TELEGRAM_APP_HASH     Telegram API app hash
  ANTHROPIC_API_KEY     Anthropic API key (for summarize command)`)
}

func newClient(cfg *config.Config) *gotelegram.Client {
	return gotelegram.NewClient(cfg.TelegramAppID, cfg.TelegramAppHash, gotelegram.Options{
		SessionStorage: &session.FileStorage{
			Path: filepath.Join(cfg.SessionDir, "session.json"),
		},
	})
}

func runAuth(ctx context.Context, cfg *config.Config) error {
	client := newClient(cfg)
	return client.Run(ctx, func(ctx context.Context) error {
		flow := telegram.NewAuthFlow()
		if err := client.Auth().IfNecessary(ctx, flow); err != nil {
			return err
		}
		user, err := client.Self(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("Authenticated as: %s %s (@%s)\n", user.FirstName, user.LastName, user.Username)
		return nil
	})
}

func runChannels(ctx context.Context, cfg *config.Config) error {
	client := newClient(cfg)
	return client.Run(ctx, func(ctx context.Context) error {
		flow := telegram.NewAuthFlow()
		if err := client.Auth().IfNecessary(ctx, flow); err != nil {
			return err
		}

		channels, err := telegram.ListChannels(ctx, client.API())
		if err != nil {
			return err
		}

		if len(channels) == 0 {
			fmt.Println("No channels found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TITLE\tUSERNAME\tID")
		for _, ch := range channels {
			username := ch.Username
			if username == "" {
				username = "-"
			}
			fmt.Fprintf(w, "%s\t@%s\t%d\n", ch.Title, username, ch.ID)
		}
		return w.Flush()
	})
}

func runRead(ctx context.Context, cfg *config.Config, args []string) error {
	fs := flag.NewFlagSet("read", flag.ExitOnError)
	limit := fs.Int("limit", 200, "number of messages to fetch")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: telgo read <channel> [-limit N]")
	}
	channelQuery := fs.Arg(0)

	client := newClient(cfg)
	return client.Run(ctx, func(ctx context.Context) error {
		flow := telegram.NewAuthFlow()
		if err := client.Auth().IfNecessary(ctx, flow); err != nil {
			return err
		}

		api := client.API()
		ch, err := telegram.ResolveChannel(ctx, api, channelQuery)
		if err != nil {
			return err
		}

		fmt.Printf("Fetching %d messages from \"%s\"...\n", *limit, ch.Title)
		messages, err := telegram.FetchMessages(ctx, api, ch, *limit)
		if err != nil {
			return err
		}

		fmt.Printf("Got %d messages\n\n", len(messages))
		for _, msg := range messages {
			fmt.Printf("[%s] %s: %s\n", msg.Date.Format("Jan 02 15:04"), msg.Sender, msg.Text)
		}
		return nil
	})
}

func runSummarize(ctx context.Context, cfg *config.Config, args []string) error {
	fs := flag.NewFlagSet("summarize", flag.ExitOnError)
	limit := fs.Int("limit", 200, "number of messages to fetch")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: telgo summarize <channel> [-limit N]")
	}
	channelQuery := fs.Arg(0)

	if cfg.AnthropicAPIKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY is required for summarize command")
	}

	client := newClient(cfg)
	return client.Run(ctx, func(ctx context.Context) error {
		flow := telegram.NewAuthFlow()
		if err := client.Auth().IfNecessary(ctx, flow); err != nil {
			return err
		}

		api := client.API()
		ch, err := telegram.ResolveChannel(ctx, api, channelQuery)
		if err != nil {
			return err
		}

		fmt.Printf("Fetching %d messages from \"%s\"...\n", *limit, ch.Title)
		messages, err := telegram.FetchMessages(ctx, api, ch, *limit)
		if err != nil {
			return err
		}
		fmt.Printf("Got %d messages, summarizing...\n\n", len(messages))

		sum := summarize.New(cfg.AnthropicAPIKey)
		summary, err := sum.Summarize(ctx, ch.Title, messages)
		if err != nil {
			return err
		}

		fmt.Println(summary)
		return nil
	})
}
