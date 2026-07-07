# Slack Socket Mode Driver for Honeydipper

This driver enables Honeydipper to receive Slack events via [Socket Mode](https://api.slack.com/apis/connections/socket) — an outbound WebSocket connection — instead of HTTP webhooks. This solves the corporate requirement of not opening ports to the public internet.

## Features (Phase 1)

- **Socket Mode WebSocket Connection**: Establishes a secure outbound WebSocket connection to Slack
- **Event Ingestion**: Receives Slack events (`message`, `app_mention`, `member_joined_channel`, etc.) through Socket Mode
- **Event Matching**: Matches incoming events against Honeydipper `collapsedEvents` rules using `dipper.CompareAll()`
- **Graceful Reconnection**: Exponential backoff with configurable max retries
- **Envelope Acknowledgment**: Properly acknowledges each Slack event envelope

## Prerequisites

- Go 1.25+ (for building)
- A Slack App with Socket Mode enabled
- Slack App-Level Token (\`xapp-\`) with \`connections:write\` scope
- Slack Bot Token (\`xoxb-\`) with appropriate event subscriptions

## Quick Start

### 1. Build the Driver

```bash
git clone https://github.com/Charles546/hd-driver-slacksocket.git
cd hd-driver-slacksocket
make build
```

### 2. Configure Slack App

1. Enable Socket Mode in your Slack App settings
2. Create an App-Level Token with \`connections:write\` scope
3. Subscribe to the bot events you need
4. Install the app to your workspace and get the Bot Token

### 3. Configure Honeydipper

Add the driver to your Honeydipper configuration:

```yaml
drivers:
  daemon:
    services:
      receiver:
        pipelines-in:
          - slack.*
  slacksocket:
    data:
      app_token: xapp-your-app-token
      bot_token: xoxb-your-bot-token
```

### 4. Configure Event Rules

Define which Slack events to process using \`collapsedEvents\`:

```yaml
drivers:
  daemon:
    engines:
      main:
        dynamicData:
          collapsedEvents:
            slack.message:
              - match:
                  event_type: message
            slack.app_mention:
              - match:
                  event_type: app_mention
```

### 5. Run

Start Honeydipper with the slacksocket driver configured. The driver will automatically connect to Slack via Socket Mode and start receiving events.

## Driver Lifecycle

| Method     | Description                                              |
|------------|----------------------------------------------------------|
| `start()`  | Initializes Slack client, establishes WebSocket connection |
| `drain()`  | Gracefully closes WebSocket, sets driver to drained state |
| `reload()` | Reloads options from configuration                      |

## Event Flow

1. Slack sends an event through the Socket Mode WebSocket
2. Driver receives the envelope and acknowledges it back to Slack
3. Driver extracts the event payload
4. Driver checks the event against \`collapsedEvents\` conditions
5. If matched, the event is emitted to Honeydipper's event bus as \`slack.<event_type>\`

## Reconnection Logic

- On disconnect requested by Slack, the driver gets a new WebSocket URL and reconnects
- On connection drop, the driver uses exponential backoff (1s → 2s → 4s → ... → 60s max)
- Maximum of 10 reconnection attempts before panicking
- Reconnection respects driver state — no reconnection if draining or completed

## Documentation

- [Integration Guide](docs/INTEGRATION.md) — Detailed setup and configuration
- [Development Guide](docs/DEVELOPMENT.md) — Building, testing, and contributing

## License

This project is prepared for dual licensing:

- `LICENSE` — GNU Affero General Public License v3.0
- `LICENSE-COMMERCIAL.md` — commercial licensing path for organizations that want to use the software outside the AGPL terms

The AGPL license applies by default unless you have a separate written commercial agreement with the copyright holder.

## Commercial licensing

If your intended use does not fit AGPL obligations, see `LICENSE-COMMERCIAL.md` and contact the copyright holder for commercial terms.

## Contributing

Contributions are welcome. Please ensure tests pass and add new tests for new features.

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run \`go test ./...\` and \`golangci-lint run\`
5. Submit a pull request

## References

- [Slack Socket Mode Documentation](https://api.slack.com/apis/connections/socket)
- [Slack Events API Documentation](https://api.slack.com/apis/connections/events-api)
- [slack-go/slack Library](https://github.com/slack-go/slack)
- [Honeydipper Documentation](https://github.com/honeydipper/honeydipper)
