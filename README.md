# http-pull
Simple service that pulls files via http(s) in configurable intervals, stores them on your system and executes hooks after each successful and/or failed download. Configurable via YAML.

## Features
- **Easily Configurable** - Configure which files to pull in which interval and what to do with them in a single configuration file
- **Sub-Minute Intervals** - Pull files as often (or as infrequent) as you need to
- **Atomic Writes** - Files are downloaded into a temporary file first, then atomically moved into place (`inotify`-compatible)
- **File Permissions** - Optionally set file mode, owner, and group — applied before the atomic rename, so watchers never see incorrect permissions
- **Hooks** - Configure hooks to be executed after a pull was completed successfully or after it failed
- **Extensible** - Easily develop own hooks by satisfying a simple Go interface with just one method

## Installation
You can download a [pre-built binary](https://github.com/rescaled/http-pull/releases) for macOS, Linux or Windows from GitHub. 

Ubuntu or Debian users can also install `http-pull` as package.

## Build
Building this project requires Go 1.24+ to be installed on your system.

```sh
git clone git@github.com:rescaled/http-pull.git
cd http-pull
go build -o http-pull ./cmd/http-pull/
```

## Usage
Running the service is as easy as pointing to a configuration file. By default, `http-pull` assumes that the configuration file is stored at `/etc/http-pull/config.yaml`. It's recommended to run the software in the background or as service with e.g. systemd. You can find a [systemd service template](https://github.com/rescaled/http-pull/blob/main/packaging/systemd/http-pull.service) in the `packaging/` directory that is being used to build the .deb packages for Ubuntu and Debian and adapt it to your needs, if necessary.

```sh
http-pull --config /path/to/config.yaml
```

## Configuration

```yaml
log:
  level: INFO # DEBUG, INFO, WARN, or ERROR
  target: stdout # stdout or file
  file: /var/log/http-pull.log # only used when target is "file"

targets:
  - name: example # unique target name
    url: https://example.com/data.txt # URL to download from
    interval: 30s # interval as Go-compatible expression
    destination: /tmp/data.txt # final destination for the downloaded file
    http_request: # optional
      method: GET # default: GET
      headers: # optional
        - name: Authorization
          value: Bearer my-token
      basic_auth: # optional
        username: username
        password: password
      follow_redirects: true # default: true
    file: # optional
      mode: "0644" # octal file mode
      owner: www-data # user name or numeric UID
      group: www-data # group name or numeric GID
    hooks: # optional
      - type: shell # `shell` or `move`
        command: echo "Downloaded successfully"
        on: # default: [success]
          - success
          - failure
      - type: move 
        destination: /opt/data/data.txt
```

## File Permissions
Each target can include an optional `file` block to set mode, owner, and group on the downloaded file. Permissions are applied to the temporary file **before** the atomic rename, so services watching the destination (e.g. via `inotify`) will never observe a file with incorrect permissions.

All three fields are optional and can be used independently:

```yaml
file:
  mode: "0755"  # octal string
  owner: deploy # user name or numeric UID
  group: staff  # group name or numeric GID
```

Setting `owner` or `group` typically requires running `http-pull` as root.

## Hooks

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

## Custom Hooks
The hook interface is in `pkg/hook` so it can be imported by external projects. Writing your own hook is as easy as satisfying the interface's single method.

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

We happily accept pull requests for new hooks to extend the possibilities `http-pull` offers its users.

## License
MIT License
