# telgo

<img width="1536" height="1024" alt="telgo" src="https://github.com/user-attachments/assets/f13ccc6b-a92f-48d8-9256-c0163b49ea4b" />

A CLI tool for reading and summarizing Telegram channels using your personal account (MTProto, not a bot).

Fetches message history and produces clean, agent-ready summaries via Claude.

---

## Prerequisites

- Go 1.21+
- Telegram API credentials — get them at [my.telegram.org](https://my.telegram.org) → *API development tools*
- Anthropic API key — only needed for the `summarize` command

---

## Installation

```sh
git clone https://github.com/amitdvl/telgo
cd telgo
go build -o telgo .
```

Or install directly:

```sh
go install github.com/amitdvl/telgo@latest
```

---

## Setup

Copy `.env.example` to `.env` and fill in your credentials:

```sh
cp .env.example .env
```

```env
TELEGRAM_APP_ID=12345678
TELEGRAM_APP_HASH=your_app_hash_here
ANTHROPIC_API_KEY=sk-ant-...
```

Load the env before running (or use [direnv](https://direnv.net/)):

```sh
export $(cat .env | grep -v '#' | xargs)
```

---

## Commands

### `auth` — Authenticate with Telegram

Interactive one-time login using your phone number. The session is saved to `~/.telgo/session.json` and reused for all subsequent commands.

```sh
telgo auth
```

You will be prompted for:
1. Your phone number (e.g. `+1234567890`)
2. The login code sent to your Telegram app
3. Your 2FA password, if enabled

You only need to do this once per machine.

---

### `channels` — List accessible channels

Lists all channels and supergroups your account can access.

```sh
telgo channels
```

Example output:

```
TITLE               USERNAME          ID
Hacker News         @hackernewsfeed   1234567890
Go Blog             @golang           9876543210
My Private Channel  -                 1122334455
```

Use the `USERNAME` or `ID` value as the `<channel>` argument in other commands.

---

### `read` — Fetch and print messages

Fetches recent messages from a channel and prints them chronologically.

```sh
telgo read <channel> [-limit N]
```

| Argument | Description |
|---|---|
| `<channel>` | Username (e.g. `golang` or `@golang`) or numeric channel ID |
| `-limit N` | Number of messages to fetch (default: `200`) |

**Examples:**

```sh
# Read the 200 most recent messages
telgo read golang

# Read the 500 most recent messages
telgo read @golang -limit 500

# Read from a private channel using its ID
telgo read 1122334455 -limit 100
```

Example output:

```
Fetching 200 messages from "Go Blog"...
Got 200 messages

[Mar 18 14:32] Go Team: Go 1.25 is released...
[Mar 17 09:15] Go Team: Proposal: add iter.Zip to the standard library...
```

---

### `summarize` — Fetch and summarize messages

Fetches recent messages and generates a structured summary using Claude. Designed to be useful for both humans and agents.

```sh
telgo summarize <channel> [-limit N]
```

Requires `ANTHROPIC_API_KEY` to be set.

| Argument | Description |
|---|---|
| `<channel>` | Username or numeric channel ID |
| `-limit N` | Number of messages to summarize (default: `200`) |

**Examples:**

```sh
# Summarize the last 200 messages
telgo summarize golang

# Summarize more history
telgo summarize @hackernewsfeed -limit 500

# Pipe the summary to a file
telgo summarize golang > summary.txt
```

Example output:

```
## Go Blog — Summary (200 messages)
Period: Mar 10 – Mar 18, 2026

### Key Announcements
- Go 1.25 released with improved range-over-func support and reduced binary sizes.
- New proposal accepted: `iter.Zip` added to the standard library.

### Notable Discussions
- Community debate on structured logging best practices.
- Several contributors discussed improving the gopls performance on large monorepos.

### Links & Resources
- https://go.dev/blog/go1.25
- https://github.com/golang/go/issues/68426
```

---

## Session Storage

Sessions are stored at `~/.telgo/session.json` by default. Override with:

```sh
export TELGO_SESSION_DIR=/path/to/dir
```

The directory is created with `0700` permissions. Keep it private — it contains your Telegram auth key.

---

## Architecture

```
telgo/
├── main.go              # CLI entry point (auth, channels, read, summarize)
├── config/
│   └── config.go        # Env-based configuration
├── telegram/
│   ├── auth.go          # Interactive MTProto auth flow
│   ├── channels.go      # Channel listing and resolution
│   ├── messages.go      # Message history fetching with pagination
│   └── types.go         # Shared types (Channel, Message)
└── summarize/
    └── claude.go        # Claude API summarization
```

**Libraries:**
- [`gotd/td`](https://github.com/gotd/td) — MTProto client (personal account, not bot API)
- [`anthropics/anthropic-sdk-go`](https://github.com/anthropics/anthropic-sdk-go) — Claude summarization

The package boundaries are clean and the `telegram/` and `summarize/` packages are independently usable — straightforward to wrap as an MCP server later.

---

## License

MIT
