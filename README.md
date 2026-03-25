# poeusage

Monitor your Poe API point balance and usage history from the terminal.

Install with `brew install rgstephens/tap/poeusage`

---

## Command Tree

```sh
poeusage [global flags] <subcommand> [flags]

poeusage balance
poeusage history [flags]
poeusage summary [flags]
poeusage completion <shell>
```

---

## Global Flags

| Flag         | Short | Type   | Default        | Description                                |
| ------------ | ----- | ------ | -------------- | ------------------------------------------ |
| `--api-key`  |       | string | `$POE_API_KEY` | Poe API key (prefer env var)               |
| `--json`     |       | bool   | false          | Output raw JSON to stdout                  |
| `--plain`    |       | bool   | false          | Stable line-based output (script-friendly) |
| `--no-color` |       | bool   | false          | Disable ANSI color (also: `NO_COLOR` env)  |
| `--quiet`    | `-q`  | bool   | false          | Suppress non-essential output              |
| `--verbose`  | `-v`  | bool   | false          | Show HTTP request details on stderr        |
| `--timeout`  |       | int    | 30             | HTTP timeout in seconds                    |
| `--help`     | `-h`  | bool   |                | Show help                                  |
| `--version`  |       | bool   |                | Print version to stdout                    |

---

## Subcommands

### `poeusage balance`

Fetch and display current point balance.

```sh
USAGE: poeusage balance [--json] [--plain]
```

**TTY output:**

```sh
Current balance: 1,500 pts
```

**`--plain` output:**

```sh
1500
```

**`--json` output:**

```json
{"current_point_balance": 1500}
```

---

### `poeusage history`

Fetch usage history. Auto-paginates by default until all records are retrieved (or until `--limit` is reached). Respects the API's 100-per-page max internally.

```sh
USAGE: poeusage history [flags]
```

| Flag            | Short | Type   | Default | Description                                                         |
| --------------- | ----- | ------ | ------- | ------------------------------------------------------------------- |
| `--limit`       | `-n`  | int    | 0 (all) | Max records to fetch total (0 = fetch all)                          |
| `--page-size`   |       | int    | 100     | Records per API call (max 100)                                      |
| `--no-paginate` |       | bool   | false   | Fetch only one page, print cursor for next                          |
| `--cursor`      |       | string |         | Resume pagination from this `query_id`                              |
| `--bot`         | `-b`  | string |         | Filter by bot name (substring match, case-insensitive)              |
| `--type`        | `-t`  | string |         | Filter by usage type: `chat`, `api`, `canvas`                       |
| `--since`       |       | string |         | Only show records after this date (`YYYY-MM-DD` or Unix timestamp)  |
| `--until`       |       | string |         | Only show records before this date (`YYYY-MM-DD` or Unix timestamp) |
| `--output`      | `-o`  | string |         | Write output to file (use `-` for stdout)                           |
| `--format`      | `-f`  | string | `table` | Output format: `table`, `csv`, `json`                               |

**Notes:**
- `--json` global flag is equivalent to `--format json`
- `--plain` global flag is equivalent to `--format csv`
- `--since`/`--until` filter client-side (the API has no date filter)
- When `--no-paginate` is used, prints `next-cursor: <query_id>` to stderr if more pages exist

**TTY table output (default):**

```sh
TIME                 BOT                    TYPE   POINTS    COST (USD)
2024-01-09 14:00     Claude-3.5-Sonnet      API       339    $0.00075
2024-01-09 13:45     GPT-4o                 Chat      210    $0.00050
...
2 records shown. Total: 549 pts / $0.00125
```

**`--format csv` output:**

```sh
time,bot_name,usage_type,cost_points,cost_usd,query_id,chat_name,input_pts,output_pts,cache_write_pts,cache_discount_pts
2024-01-09T14:00:00Z,Claude-3.5-Sonnet,API,339,0.00075,2Nhd9xBFbLcXEwmNj,,120,219,0,0
```

**`--format json` output:**

```json
[
  {
    "bot_name": "Claude-3.5-Sonnet",
    "time": "2024-01-09T14:00:00Z",
    "query_id": "2Nhd9xBFbLcXEwmNj",
    "cost_usd": "0.00075",
    "cost_points": 339,
    "usage_type": "API",
    "chat_name": null,
    "cost_breakdown": { "input": 120, "output": 219, "cache_write": 0, "cache_discount": 0, "total": 339 }
  }
]
```

---

### `poeusage summary`

Aggregate usage history and display a cost breakdown. Accepts the same filtering flags as `history`.

```sh
USAGE: poeusage summary [flags]
```

| Flag         | Short | Type   | Default | Description                                         |
| ------------ | ----- | ------ | ------- | --------------------------------------------------- |
| `--bot`      | `-b`  | string |         | Filter by bot name (substring match)                |
| `--type`     | `-t`  | string |         | Filter by usage type                                |
| `--since`    |       | string |         | Only include records after this date                |
| `--until`    |       | string |         | Only include records before this date               |
| `--group-by` | `-g`  | string | `bot`   | Group aggregation: `bot`, `type`, `day`, `bot,type` |
| `--format`   | `-f`  | string | `table` | Output format: `table`, `csv`, `json`               |

**TTY table output (default, `--group-by bot`):**

```sh
BOT                    QUERIES   POINTS    COST (USD)
Claude-3.5-Sonnet          42    14,238    $3.21
GPT-4o                     18     6,120    $1.38
...
TOTAL                      60    20,358    $4.59
```

---

### `poeusage completion <shell>`

Print shell completion script to stdout.

```sh
USAGE: poeusage completion <bash|zsh|fish>
```

Install example:

```bash
poeusage completion zsh > ~/.zsh/completions/_poeusage
```

---

## I/O Contract

- **stdout:** All primary output (data, JSON, CSV, plain text)
- **stderr:** Diagnostics, progress, warnings, errors, next-cursor hints
- TTY detection governs color and table formatting; piped output defaults to `--plain` behavior

---

## Exit Codes

| Code | Meaning                                                        |
| ---- | -------------------------------------------------------------- |
| `0`  | Success                                                        |
| `1`  | Runtime error (network failure, API error)                     |
| `2`  | Invalid usage (bad flag, unknown subcommand, validation error) |
| `3`  | Authentication error (401 from API)                            |

---

## Configuration

**Precedence (high → low):** flags > env > `~/.config/poeusage/config.toml` > defaults

### Environment Variables

| Variable             | Description                               |
| -------------------- | ----------------------------------------- |
| `POE_API_KEY`        | API key — preferred over `--api-key` flag |
| `NO_COLOR`           | Disable color output (standard)           |
| `POEUSAGE_TIMEOUT`   | Default HTTP timeout in seconds           |
| `POEUSAGE_PAGE_SIZE` | Default page size for history calls       |

### Config File

`~/.config/poeusage/config.toml` (XDG-compliant):

```toml
# api_key = "..."   # not recommended; prefer POE_API_KEY env var
timeout = 30
page_size = 100
```

The API key is intentionally **not** accepted in the config file to discourage storing plaintext secrets on disk. Use `POE_API_KEY` via a secrets manager, `.env`, or keychain.

---

## Examples

```bash
# Check current balance
poeusage balance

# Last 50 history entries as a table
poeusage history --limit 50

# Filter by bot and output CSV (for spreadsheet)
poeusage history --bot Claude-3.5-Sonnet --format csv -o usage.csv

# Usage since start of month, grouped by day
poeusage summary --since 2024-01-01 --group-by day

# Pipe JSON into jq to find expensive queries
poeusage history --format json | jq '.[] | select(.cost_points > 500)'

# Script-safe: plain balance for use in shell conditionals
BALANCE=$(poeusage balance --plain)
[[ $BALANCE -lt 500 ]] && echo "Low balance: $BALANCE pts"

# One page at a time (manual pagination)
poeusage history --no-paginate --page-size 20
# stderr: next-cursor: 2Nhd9xBFbLcXEwmNj
poeusage history --no-paginate --cursor 2Nhd9xBFbLcXEwmNj

# Usage by type over the past week
poeusage summary --since 2024-01-02 --group-by type

# Install zsh completions
poeusage completion zsh > ~/.zsh/completions/_poeusage
```

---

## Build & Release

```sh
make release TAG=v1.0.0
```

---

## Design Notes

- **Auto-paginate by default** — fetching all history is the most common need; manual paging is opt-in via `--no-paginate` + `--cursor`
- **Client-side date filtering** — the API has no date params; `--since`/`--until` filter after fetch, so fetching large history sets with a tight date window may be slow
- **`--format` over separate flags** — `table/csv/json` are mutually exclusive; `--json` global is an alias for `--format json` for ergonomics
- **No `--api-key` in config file** — reduces plaintext secret risk; `POE_API_KEY` env var is the right channel

---

## API Reference

- Base URL: `https://api.poe.com/usage`
- Authentication: `Authorization: Bearer <POE_API_KEY>`
- Get your API key: https://poe.com/api/keys
- Docs: https://creator.poe.com/docs/resources/usage-api
