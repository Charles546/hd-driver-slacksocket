## [1.0.1](https://github.com/Charles546/hd-driver-slacksocket/compare/v1.0.0...v1.0.1) (2026-07-12)


### Bug Fixes

* add rules for workflow resuming ([#7](https://github.com/Charles546/hd-driver-slacksocket/issues/7)) ([7310581](https://github.com/Charles546/hd-driver-slacksocket/commit/7310581fe69b538d73938750d7a390027041d1f8))

# 1.0.0 (2026-07-11)


### Bug Fixes

* correct config yaml ([#5](https://github.com/Charles546/hd-driver-slacksocket/issues/5)) ([9bfca6e](https://github.com/Charles546/hd-driver-slacksocket/commit/9bfca6e9fad75b550760b18113831e122f7b2a5d))


### Features

* add Dockerfile for containerized deployment ([#2](https://github.com/Charles546/hd-driver-slacksocket/issues/2)) ([0de7090](https://github.com/Charles546/hd-driver-slacksocket/commit/0de7090d0fdb6daa834d99689796d26d2fd97ea1))
* add Slack interactive event and slash command support via Socket Mode ([#3](https://github.com/Charles546/hd-driver-slacksocket/issues/3)) ([81109d2](https://github.com/Charles546/hd-driver-slacksocket/commit/81109d2aca26e029861c1f0b5d62d447db393c1c)), closes [#2](https://github.com/Charles546/hd-driver-slacksocket/issues/2)
* initial Slack Socket Mode driver for Honeydipper ([#1](https://github.com/Charles546/hd-driver-slacksocket/issues/1)) ([538dfad](https://github.com/Charles546/hd-driver-slacksocket/commit/538dfad93217bcad80cd1712d06f398dfe180732))

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
