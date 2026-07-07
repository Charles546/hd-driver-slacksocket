# Development Guide

## Prerequisites

- Go 1.25+
- Access to a Slack workspace with an app that has Socket Mode enabled

## Local Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/Charles546/hd-driver-slacksocket.git
   cd hd-driver-slacksocket
   ```

2. Build the driver:
   ```bash
   make build
   ```

3. Run tests:
   ```bash
   make test
   ```

4. Run linting:
   ```bash
   golangci-lint run ./...
   ```

## Project Structure

```
.
├── .github/workflows/         # CI/CD workflows
├── cmd/
│   └── hd-driver-slacksocket/ # Main driver binary
│       ├── main.go            # Driver implementation
│       └── main_test.go       # Unit tests
├── config/
│   └── init.yaml              # Sample Honeydipper configuration
├── docs/
│   ├── DEVELOPMENT.md         # This file
│   └── INTEGRATION.md         # Integration guide
├── .gitignore
├── .golangci.yml              # Linter configuration
├── .hdci.yml                  # Honeydipper CI configuration
├── .releaserc                 # Semantic release configuration
├── CHANGELOG.md
├── LICENSE                    # AGPL v3
├── LICENSE-COMMERCIAL.md      # Commercial licensing info
├── Makefile                   # Build and test targets
├── README.md                  # Main documentation
└── go.mod / go.sum            # Go module files
```

## Architecture

The driver follows the standard Honeydipper driver pattern:

1. **Lifecycle**: The daemon launches the driver as a subprocess and communicates via stdin/stdout using the dipper wire protocol.
2. **Receiver Service**: When the driver is started as a \`receiver\` service, it establishes a Socket Mode WebSocket connection to Slack.
3. **Event Flow**: Slack events arrive via the WebSocket connection, are checked against collapsedEvents rules, and matched events are emitted back to the daemon.
4. **Reconnection**: The driver implements exponential backoff reconnection with a maximum of 10 retries, respecting the driver state (does not reconnect during drain/stop).

## Code Conventions

- Follow standard Go formatting (`gofmt -s`)
- Use descriptive error variables prefixed with \`Err\`
- Log with the driver logger: \`d.GetLogger().Infof(...)\`
- Use \`dipper.CompareAll()\` for event matching against conditions
- Add unit tests for all new functionality
