# Integration Guide

## Overview

This driver enables Honeydipper to receive events from Slack using [Socket Mode](https://api.slack.com/apis/connections/socket) — an outbound WebSocket connection from your Honeydipper infrastructure to Slack. This eliminates the need to open public HTTP ports for webhooks.

## How It Works

1. Honeydipper starts the \`hd-driver-slacksocket\` driver as a subprocess.
2. The driver authenticates with Slack using two tokens:
   - **App-Level Token** (\`xapp-\`): Used to establish the Socket Mode WebSocket connection.
   - **Bot Token** (\`xoxb-\`): Used for API calls and event subscription verification.
3. A secure WebSocket connection is established to Slack's Socket Mode endpoint.
4. Slack sends events (messages, app mentions, channel joins, etc.) through this WebSocket.
5. The driver acknowledges each event envelope and processes the payload.
6. Events are matched against \`collapsedEvents\` conditions defined in Honeydipper configuration.
7. Matched events are emitted into Honeydipper's event pipeline as \`slack.<event_type>\`.

## Slack App Setup

### 1. Create a Slack App

1. Go to https://api.slack.com/apps
2. Click "Create New App" → "From Scratch"
3. Name your app and select your workspace

### 2. Enable Socket Mode

1. In your app settings, navigate to **Socket Mode** (under Settings)
2. Toggle "Enable Socket Mode" to **On**
3. An app-level token will be automatically created

### 3. Create App-Level Token

If needed, create a dedicated app-level token:
1. Navigate to **Basic Information** → **App-Level Tokens**
2. Click "Generate Token and Scopes..."
3. Add the \`connections:write\` scope
4. Copy the token (starts with \`xapp-\`)

### 4. Configure Bot Token Scopes

1. Navigate to **OAuth & Permissions** → **Scopes**
2. Add Bot Token Scopes based on the events you need:
   - \`chat:write\` — Send messages
   - \`channels:history\` — Read channel messages
   - \`channels:read\` — View channel info
   - \`groups:history\` — Read private channel messages
   - \`im:history\` — Read direct messages
   - \`mpim:history\` — Read group DMs
   - \`reactions:read\` — View reactions
   - \`users:read\` — View user info

### 5. Subscribe to Events

1. Navigate to **Event Subscriptions** → **Subscribe to bot events**
2. Add the events you want to receive, for example:
   - \`message.channels\` — Messages in public channels
   - \`message.groups\` — Messages in private channels
   - \`message.im\` — Messages in direct messages
   - \`app_mention\` — When your app is mentioned
   - \`member_joined_channel\` — When a user joins a channel

### 6. Install the App

1. Navigate to **OAuth & Permissions** → **OAuth Tokens for Your Workspace**
2. Click "Install to Workspace"
3. Copy the Bot User OAuth Token (starts with \`xoxb-\`)

## Honeydipper Configuration

### Adding the Driver

Add the driver to your Honeydipper configuration:

```yaml
drivers:
  daemon:
    drivers:
      slacksocket:
        name: slacksocket
        type: remote
        handlerData:
          registry: charles-gh-pages
          channel: stable
    services:
      receiver:
        description: Honeydipper event receiver for Slack events via Socket Mode
        pipelines-in:
          - slack.*
  slacksocket:
    data:
      app_token: xapp-xxxxxxxxxxxxx
      bot_token: xoxb-xxxxxxxxxxxxx
      loglevel: info
```

### Defining Event Rules with collapsedEvents

Use \`collapsedEvents\` to define which Slack events trigger which Honeydipper workflows and under what conditions:

```yaml
drivers:
  daemon:
    engines:
      engine1:
        dynamicData:
          collapsedEvents:
            slack.message:
              - match:
                  event_type: message
                  event:
                    channel: C12345
              - match:
                  event_type: app_mention
            slack.member_joined_channel:
              - match:
                  event_type: member_joined_channel
```

### Event Payload Structure

When a matching event is emitted, it has the following structure:

```json
{
  "events": ["slack.<event_type>"],
  "data": {
    "event_type": "message",
    "event": {
      "type": "message",
      "text": "Hello, world!",
      "channel": "C12345",
      "user": "U12345",
      "ts": "1234567890.123456",
      ...
    }
  }
}
```

### Workflow Example

```yaml
workflows:
  - name: handle-slack-message
    when:
      source:
        system: slack
        trigger: message
    do:
      - name: log-message
        log:
          text: "Slack message from {{ .ctx.event.user }}: {{ .ctx.event.text }}"
```

## Testing the Integration

1. Ensure Honeydipper is running with the slacksocket driver configured.
2. Send a message in a channel your app has access to.
3. Check Honeydipper logs for the received event.
4. Verify that workflows triggered by Slack events execute correctly.

## Troubleshooting

### Connection Issues

- Verify that the app-level token starts with \`xapp-\` and has the \`connections:write\` scope.
- Verify that the bot token starts with \`xoxb-\`.
- Ensure Socket Mode is enabled in your Slack app settings.
- Check that the app is installed to the workspace.

### Event Not Received

- Verify that the bot is a member of the channel (invite the bot: \`/invite @your-bot\`).
- Check that the event is subscribed in Slack app configuration.
- Verify \`collapsedEvents\` configuration matches the expected event type.

### Reconnection Failures

- Check network connectivity to Slack's endpoints.
- Verify token validity (tokens may expire or be revoked).
- Examine driver logs for reconnection attempt details.
