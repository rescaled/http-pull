# http-pull

A lightweight Go service that periodically downloads files via HTTP(S), stores them locally, and runs configurable hooks afterward.

## Features

- **Scheduled HTTP pulls** — download files at configurable intervals per target
- **Atomic writes** — files are written to a temp file first, then renamed into place
- **Built-in hooks** — run shell commands or move files after each download
- **Extensible hook interface** — third-party hooks via a public Go interface in `pkg/hook`
- **Live reload** — send `SIGHUP` to re-read configuration without restarting
- **Graceful shutdown** — in-flight downloads and hooks complete before exit
- **Structured logging** — JSON logs via `slog`, configurable level and output target

## Installation

Requires Go 1.24+.

```sh
go build -o http-pull ./cmd/http-pull/
```

## Usage

```sh
http-pull --config /path/to/config.yaml
```

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `/etc/http-pull/config.yaml` | Path to the configuration file |

### Signals

| Signal | Behaviour |
|--------|-----------|
| `SIGHUP` | Reload configuration and apply changes live |
| `SIGINT` / `SIGTERM` | Graceful shutdown (30s timeout for in-flight work) |

## Configuration

```yaml
log:
  level: INFO           # DEBUG, INFO, WARN, or ERROR
  target: stdout        # stdout or file
  file: /var/log/http-pull.log  # only used when target is "file"

targets:
  - name: example
    url: https://example.com/data.txt
    interval: 30s
    destination: /tmp/data.txt
    http_request:           # optional
      method: GET           # default: GET
      headers:              # optional
        - name: Authorization
          value: Bearer my-token
      basic_auth:           # optional
        username: user
        password: pass
      follow_redirects: true  # default: true
    hooks:                  # optional
      - type: shell
        command: echo "Downloaded successfully"
        on:                 # default: [success]
          - success
          - failure
      - type: move
        destination: /opt/data/data.txt
```

### Defaults

| Setting | Default |
|---------|---------|
| `log.level` | `INFO` |
| `log.target` | `stdout` |
| `http_request.method` | `GET` |
| `http_request.follow_redirects` | `true` |
| `hooks[].on` | `["success"]` |
| `User-Agent` header | `http-pull/1.0` (overridable via `headers`) |

## Hooks

### Built-in hooks

**`shell`** — Executes a command via `/bin/sh -c` after the pull completes.

```yaml
- type: shell
  command: systemctl reload nginx
  on: [success]
```

**`move`** — Moves the downloaded file to another path. Uses atomic rename when possible, falls back to copy+delete for cross-device moves.

```yaml
- type: move
  destination: /etc/app/config.json
  on: [success]
```

### Hook triggers

| Value | Description |
|-------|-------------|
| `success` | Run when the download succeeds (HTTP status < 400) |
| `failure` | Run when the download fails (HTTP status >= 400 or network error) |

### Writing custom hooks

The hook interface is in `pkg/hook` so it can be imported by external projects:

```go
import "http-pull/pkg/hook"

type MyHook struct{}

func (h *MyHook) Execute(ctx context.Context, result hook.Result) error {
    if result.Success {
        // handle successful download at result.FilePath
    }
    return nil
}
```

## Project structure

```
cmd/http-pull/          Entry point and signal handling
internal/
  config/               Configuration loading and validation (Viper)
  logging/              Structured logging setup (slog)
  hook/                 Built-in hook implementations and registry
  puller/               HTTP download with atomic file writes
  runner/               Worker orchestration, reload, and graceful shutdown
pkg/hook/               Public hook interface for third-party extensions
```

## License

See [LICENSE](LICENSE) for details.
