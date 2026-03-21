package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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

	// setup runs before config.Load() since credentials may not exist yet.
	if cmd == "setup" {
		if err := runSetup(); err != nil {
			fmt.Fprintf(os.Stderr, "setup error: %v\n", err)
			os.Exit(1)
		}
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
  setup                 Interactive first-time setup (saves credentials to ~/.telgo/.env)
  auth                  Authenticate with Telegram (interactive)
  channels              List accessible channels
  read <channel>        Read messages from a channel
  summarize <channel>   Summarize messages from a channel

Options for read/summarize:
  -limit N    Number of messages to fetch (default: 200)`)
}

func runSetup() error {
	dir := config.DefaultDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("cannot create %s: %w", dir, err)
	}
	envPath := filepath.Join(dir, ".env")

	if _, err := os.Stat(envPath); err == nil {
		fmt.Printf("Existing config found at %s\n", envPath)
		fmt.Print("Overwrite? [y/N]: ")
		var ans string
		fmt.Scanln(&ans)
		if strings.ToLower(strings.TrimSpace(ans)) != "y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	r := bufio.NewReader(os.Stdin)
	prompt := func(label string) (string, error) {
		fmt.Print(label)
		v, err := r.ReadString('\n')
		return strings.TrimSpace(v), err
	}

	fmt.Println("Get your API credentials at https://my.telegram.org → API development tools")
	fmt.Println()

	appID, err := prompt("App api_id:   ")
	if err != nil || appID == "" {
		return fmt.Errorf("app ID is required")
	}
	appHash, err := prompt("App api_hash: ")
	if err != nil || appHash == "" {
		return fmt.Errorf("app hash is required")
	}

	fmt.Println()
	fmt.Println("Anthropic API key is used by 'telgo summarize' to generate summaries via Claude.")
	apiKey, err := prompt("ANTHROPIC_API_KEY (leave blank to skip): ")
	if err != nil {
		return err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "TELEGRAM_APP_ID=%s\n", appID)
	fmt.Fprintf(&b, "TELEGRAM_APP_HASH=%s\n", appHash)
	if apiKey != "" {
		fmt.Fprintf(&b, "ANTHROPIC_API_KEY=%s\n", apiKey)
	}

	if err := os.WriteFile(envPath, []byte(b.String()), 0600); err != nil {
		return fmt.Errorf("write %s: %w", envPath, err)
	}

	fmt.Printf("\nSaved to %s\n", envPath)
	fmt.Println("Run 'telgo auth' next to authenticate with your Telegram account.")
	return nil
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

// reorderArgs moves flag arguments before positional arguments so that
// Go's flag package (which stops at the first non-flag arg) parses them.
// It treats the argument after a flag starting with "-" as that flag's value.
func reorderArgs(args []string) []string {
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			flags = append(flags, args[i])
			if !strings.Contains(args[i], "=") && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
		} else {
			positional = append(positional, args[i])
		}
	}
	return append(flags, positional...)
}

func runRead(ctx context.Context, cfg *config.Config, args []string) error {
	fs := flag.NewFlagSet("read", flag.ExitOnError)
	limit := fs.Int("limit", 200, "number of messages to fetch")
	if err := fs.Parse(reorderArgs(args)); err != nil {
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
	if err := fs.Parse(reorderArgs(args)); err != nil {
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
