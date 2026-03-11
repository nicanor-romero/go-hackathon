# 🛡️ Command Risk Shield

A terminal TUI that intercepts shell commands, asks Claude (Haiku) to rate their risk before execution, and automatically explains failures when a command goes wrong.

## Prerequisites

- Go 1.21+
- An [Anthropic API key](https://console.anthropic.com/)

## Setup

```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

The app works without the key but analysis will be skipped.

## Build & Run

```bash
make all          # compiles → bin/shield
make run          # build + launch immediately
make clean        # remove bin/shield
```

Or run directly without building:

```bash
go run ./shield
```

## Usage

```
🛡️  Command Risk Shield

Command:
╭──────────────────────────────────────────────────────────╮
│ type a shell command...                                  │
╰──────────────────────────────────────────────────────────╯

⏎ analyze  •  ctrl+c quit
```

1. **Type** any shell command and press `Enter`.
2. Claude analyzes it and shows a risk card:

| Level    | Color  | Meaning                                          |
|----------|--------|--------------------------------------------------|
| LOW      | green  | Routine read-only or safe operation              |
| MEDIUM   | yellow | Writes files or modifies state                   |
| HIGH     | orange | Potentially destructive or hard to reverse       |
| CRITICAL | red    | Data loss, security risk, or system-wide damage  |

3. **Confirm or cancel:**
   - `y` or `Enter` — execute the command
   - `n` or `Esc` — go back and edit

4. **After execution:**
   - Success → output shown (up to 10 lines)
   - Failure → error shown + Claude explains why and how to fix it
   - `r` — run another command
   - `q` — quit

## Keyboard Reference

| Key           | State  | Action                    |
|---------------|--------|---------------------------|
| `Enter`       | Input  | Analyze command           |
| `y` / `Enter` | Result | Execute                   |
| `n` / `Esc`   | Result | Cancel, back to input     |
| `r`           | Done   | Run another command       |
| `q`           | Done   | Quit                      |
| `Ctrl+C`      | Any    | Quit immediately          |

## Limitations

- Commands are split on whitespace — quoted arguments like `echo "hello world"` are not supported.
- Requires a real TTY; does not work piped or in CI.
