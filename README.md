# zenfl-forwarder

Forward all incoming messages from a specific Telegram bot (`ZENFL_USERNAME`) to one or more target users, using your personal Telegram account.

## Behavior

- Receives updates in real time.
- Forwards matching messages immediately.
- Does **not** mark messages as read (this service never calls any read-history API).

## Requirements

- Go 1.22+
- Telegram API credentials from https://my.telegram.org:
  - `api_id`
  - `api_hash`

## Configure

Copy `.env.example` and fill values:

- `TG_API_ID`: your Telegram API ID
- `TG_API_HASH`: your Telegram API hash
- `TG_PHONE`: your phone number with country code
- `ZENFL_USERNAME`: bot username (without `@` is fine)
- `TARGET_USERNAMES`: comma-separated target usernames
- `TG_SESSION_FILE`: local session file path (default `session.json`)

## Run

```bash
go mod tidy
go run ./cmd/server
```

On first run, Telegram asks for login code and maybe 2FA password in terminal.

## Notes

- Targets must be resolvable usernames.
- If Telegram rate limits (`FLOOD_WAIT`), forwarding retries are not automatic in this minimal version.

## Project Structure

```text
cmd/server/main.go                  # app entrypoint
internal/app/run.go                 # app bootstrap (logger, signals)
internal/config/config.go           # environment config loading
internal/telegram/forwarder         # telegram forwarder domain/service
```
