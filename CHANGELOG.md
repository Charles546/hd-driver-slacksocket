# Changelog

## [0.1.0] - 2026-07-07

### Initial Release

- Slack Socket Mode driver for Honeydipper
- Establishes Socket Mode WebSocket connection to receive Slack events
- Event ingestion via Socket Mode (outbound connection, no open ports required)
- Event matching against Honeydipper collapsedEvents rules using `dipper.CompareAll()`
- Graceful connection lifecycle management with exponential backoff reconnection
- Support for `events_api`, `disconnect`, and `hello` envelope types
- Slack envelope acknowledgment support
