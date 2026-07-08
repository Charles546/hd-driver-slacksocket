// Copyright 2026 Chun Huang (Charles).
//
// This Source Code Form is dual-licensed.
// By default, this file is licensed under the GNU Affero General Public License v3.0.
// If you have a separate written commercial agreement, you may use this file under those terms instead.

package main

import (
	"errors"
	"testing"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"github.com/stretchr/testify/assert"
)

func TestBuildEventPayload(t *testing.T) {
	t.Parallel()

	d := &slacksocketDriver{}

	innerEvent := slackevents.EventsAPIInnerEvent{
		Type: "message",
		Data: map[string]interface{}{
			"text":    "Hello, world!",
			"channel": "C12345",
			"user":    "U12345",
		},
	}

	payload := d.buildEventPayload(innerEvent)
	assert.NotNil(t, payload)
	assert.Equal(t, "message", payload["event_type"])

	event, ok := payload["event"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "Hello, world!", event["text"])
	assert.Equal(t, "C12345", event["channel"])
	assert.Equal(t, "U12345", event["user"])
}

func TestBuildEventPayloadNilData(t *testing.T) {
	t.Parallel()

	d := &slacksocketDriver{}

	innerEvent := slackevents.EventsAPIInnerEvent{
		Type: "app_mention",
		Data: nil,
	}

	payload := d.buildEventPayload(innerEvent)
	assert.NotNil(t, payload)
	assert.Equal(t, "app_mention", payload["event_type"])
	// When Data is nil, "event" key should not be set
	_, ok := payload["event"]
	assert.False(t, ok)
}

func TestMatchCollapsedEventsNoConditions(t *testing.T) {
	t.Parallel()

	d := &slacksocketDriver{}
	d.collapsedEvents = nil

	payload := map[string]interface{}{
		"event_type": "message",
		"event": map[string]interface{}{
			"text": "hello",
		},
	}

	assert.True(t, d.matchCollapsedEvents(payload), "should match when no collapsed events are defined")
}

func TestMatchCollapsedEventsMatchesCondition(t *testing.T) {
	t.Parallel()

	d := &slacksocketDriver{}
	d.collapsedEvents = map[string]interface{}{
		"slack.message": []interface{}{
			map[string]interface{}{
				"match": map[string]interface{}{
					"event_type": "message",
				},
			},
		},
	}

	payload := map[string]interface{}{
		"event_type": "message",
		"event": map[string]interface{}{
			"text": "hello",
		},
	}

	assert.True(t, d.matchCollapsedEvents(payload), "should match when event_type matches")
}

func TestMatchCollapsedEventsNoMatch(t *testing.T) {
	t.Parallel()

	d := &slacksocketDriver{}
	d.collapsedEvents = map[string]interface{}{
		"slack.message": []interface{}{
			map[string]interface{}{
				"match": map[string]interface{}{
					"event_type": "message",
				},
			},
		},
	}

	payload := map[string]interface{}{
		"event_type": "app_mention",
		"event": map[string]interface{}{
			"text": "hello",
		},
	}

	assert.False(t, d.matchCollapsedEvents(payload), "should not match when event_type differs")
}

func TestMatchCollapsedEventsMultipleBranches(t *testing.T) {
	t.Parallel()

	d := &slacksocketDriver{}
	d.collapsedEvents = map[string]interface{}{
		"slack.message": []interface{}{
			map[string]interface{}{
				"match": map[string]interface{}{
					"event_type": "app_mention",
				},
			},
			map[string]interface{}{
				"match": map[string]interface{}{
					"event_type": "message",
					"event": map[string]interface{}{
						"channel": "C12345",
					},
				},
			},
		},
	}

	payload := map[string]interface{}{
		"event_type": "message",
		"event": map[string]interface{}{
			"text":    "hello",
			"channel": "C12345",
		},
	}

	assert.True(t, d.matchCollapsedEvents(payload), "should match second branch with channel condition")
}

func TestMatchCollapsedEventsMalformedEntry(t *testing.T) {
	t.Parallel()

	d := &slacksocketDriver{}
	d.collapsedEvents = map[string]interface{}{
		"slack.event": []interface{}{
			"not a map",
		},
	}

	payload := map[string]interface{}{
		"event_type": "message",
	}

	assert.False(t, d.matchCollapsedEvents(payload), "should not match when condition entries are malformed")
}

func TestErrors(t *testing.T) {
	t.Parallel()

	assert.True(t, errors.Is(ErrMissingAppToken, ErrMissingAppToken))
	assert.True(t, errors.Is(ErrMissingBotToken, ErrMissingBotToken))
	assert.True(t, errors.Is(ErrInvalidAppToken, ErrInvalidAppToken))
	assert.True(t, errors.Is(ErrInvalidBotToken, ErrInvalidBotToken))
	assert.NotEqual(t, ErrMissingAppToken.Error(), ErrMissingBotToken.Error())
}

// --- Interactive Event Tests ---

func TestBuildInteractivePayload(t *testing.T) {
	t.Parallel()

	d := &slacksocketDriver{}

	event := socketmode.Event{
		Type: socketmode.EventTypeInteractive,
		Data: slack.InteractionCallback{
			Type:       slack.InteractionTypeBlockActions,
			CallbackID: "test_callback",
			ActionTs:   "1234567890.123",
			Channel: slack.Channel{
				GroupConversation: slack.GroupConversation{
					Conversation: slack.Conversation{
						ID: "C12345",
					},
				},
			},
			User: slack.User{
				ID: "U12345",
			},
			Team:        slack.Team{ID: "T12345"},
			ResponseURL: "https://hooks.slack.com/actions/T12345/123",
			TriggerID:   "trigger_123",
		},
		Request: &socketmode.Request{
			EnvelopeID: "test-envelope-id",
			Type:       "interactive",
		},
	}

	payload := d.buildInteractivePayload(event)
	assert.NotNil(t, payload)
	assert.Equal(t, "interactive", payload["event_type"])
	assert.Equal(t, string(slack.InteractionTypeBlockActions), payload["callback_type"])
	assert.Equal(t, "test_callback", payload["callback_id"])
	assert.Equal(t, "1234567890.123", payload["action_ts"])
	assert.Equal(t, "C12345", payload["channel"])
	assert.Equal(t, "U12345", payload["user"])
	assert.Equal(t, "T12345", payload["team"])
	assert.Equal(t, "https://hooks.slack.com/actions/T12345/123", payload["response_url"])
	assert.Equal(t, "trigger_123", payload["trigger_id"])
}

func TestBuildInteractivePayloadBadData(t *testing.T) {
	t.Parallel()

	d := &slacksocketDriver{}

	event := socketmode.Event{
		Type:    socketmode.EventTypeInteractive,
		Data:    "not an InteractionCallback",
		Request: &socketmode.Request{EnvelopeID: "test-env-id", Type: "interactive"},
	}

	payload := d.buildInteractivePayload(event)
	assert.Nil(t, payload, "should return nil when data is not InteractionCallback")
}

func TestBuildInteractivePayloadNilRequest(t *testing.T) {
	t.Parallel()

	d := &slacksocketDriver{}

	event := socketmode.Event{
		Type: socketmode.EventTypeInteractive,
		Data: slack.InteractionCallback{
			Type:       slack.InteractionTypeViewSubmission,
			CallbackID: "view_callback",
			User:       slack.User{ID: "U12345"},
			Team:       slack.Team{ID: "T12345"},
		},
		// Request is nil
	}

	payload := d.buildInteractivePayload(event)
	assert.Nil(t, payload, "should return nil when request is missing")
}

func TestBuildInteractivePayloadWithViewAndActions(t *testing.T) {
	t.Parallel()

	d := &slacksocketDriver{}

	event := socketmode.Event{
		Type: socketmode.EventTypeInteractive,
		Data: slack.InteractionCallback{
			Type:       slack.InteractionTypeViewSubmission,
			CallbackID: "modal_submit",
			ActionTs:   "1234567890.456",
			Channel: slack.Channel{
				GroupConversation: slack.GroupConversation{
					Conversation: slack.Conversation{
						ID: "C54321",
					},
				},
			},
			User: slack.User{
				ID: "U67890",
			},
			Team:        slack.Team{ID: "T12345"},
			ResponseURL: "https://hooks.slack.com/actions/T12345/456",
			TriggerID:   "trigger_456",
			View: slack.View{
				ID:   "V12345",
				Type: "modal",
			},
			ActionCallback: slack.ActionCallbacks{
				BlockActions: []*slack.BlockAction{
					{
						ActionID: "submit_button",
						Value:    "clicked",
					},
				},
			},
		},
		Request: &socketmode.Request{
			EnvelopeID: "test-envelope-id-2",
			Type:       "interactive",
		},
	}

	payload := d.buildInteractivePayload(event)
	assert.NotNil(t, payload)
	assert.Equal(t, "interactive", payload["event_type"])
	assert.Equal(t, string(slack.InteractionTypeViewSubmission), payload["callback_type"])
	assert.Equal(t, "C54321", payload["channel"])
	assert.Equal(t, "U67890", payload["user"])

	// Verify view data is present
	view, ok := payload["view"].(map[string]interface{})
	assert.True(t, ok, "view should be present")
	assert.Equal(t, "V12345", view["id"])
	assert.Equal(t, "modal", view["type"])

	// Verify actions data is present (ActionCallbacks marshals as JSON array)
	actions, ok := payload["actions"].([]interface{})
	assert.True(t, ok, "actions should be present")
	assert.GreaterOrEqual(t, len(actions), 1, "should have at least one action")
}

func TestMatchCollapsedEventsWithInteractivePayload(t *testing.T) {
	t.Parallel()

	d := &slacksocketDriver{}
	d.collapsedEvents = map[string]interface{}{
		"slack.interactive": []interface{}{
			map[string]interface{}{
				"match": map[string]interface{}{
					"event_type":    "interactive",
					"callback_type": string(slack.InteractionTypeBlockActions),
				},
			},
		},
	}

	payload := map[string]interface{}{
		"event_type":    "interactive",
		"callback_type": string(slack.InteractionTypeBlockActions),
		"callback_id":   "test_callback",
		"channel":       "C12345",
		"user":          "U12345",
		"team":          "T12345",
	}

	assert.True(t, d.matchCollapsedEvents(payload), "interactive event should match block_actions condition")
}

func TestMatchCollapsedEventsWithInteractivePayloadNoMatch(t *testing.T) {
	t.Parallel()

	d := &slacksocketDriver{}
	d.collapsedEvents = map[string]interface{}{
		"slack.interactive": []interface{}{
			map[string]interface{}{
				"match": map[string]interface{}{
					"event_type":    "interactive",
					"callback_type": string(slack.InteractionTypeBlockActions),
				},
			},
		},
	}

	payload := map[string]interface{}{
		"event_type":    "interactive",
		"callback_type": string(slack.InteractionTypeViewSubmission),
		"callback_id":   "view_submit",
		"channel":       "C54321",
		"user":          "U67890",
		"team":          "T12345",
	}

	assert.False(t, d.matchCollapsedEvents(payload),
		"view_submission should not match block_actions condition")
}

func TestMatchCollapsedEventsWithInteractivePayloadAllEvents(t *testing.T) {
	t.Parallel()

	d := &slacksocketDriver{}
	d.collapsedEvents = map[string]interface{}{
		"slack.interactive": []interface{}{
			map[string]interface{}{
				"match": map[string]interface{}{
					"event_type": "interactive",
				},
			},
		},
	}

	payload := map[string]interface{}{
		"event_type":    "interactive",
		"callback_type": string(slack.InteractionTypeViewSubmission),
		"callback_id":   "view_submit",
	}

	assert.True(t, d.matchCollapsedEvents(payload),
		"interactive event should match when condition only requires event_type")
}

// --- Slash Command Tests ---

func TestBuildSlashCommandPayload(t *testing.T) {
	t.Parallel()

	d := &slacksocketDriver{}

	event := socketmode.Event{
		Type: socketmode.EventTypeSlashCommand,
		Data: slack.SlashCommand{
			Command:     "/test",
			Text:        "arg1 arg2",
			UserID:      "U12345",
			ChannelID:   "C12345",
			TeamID:      "T12345",
			ResponseURL: "https://hooks.slack.com/commands/T12345/123",
			TriggerID:   "trigger_123",
			UserName:    "testuser",
			ChannelName: "general",
		},
		Request: &socketmode.Request{
			EnvelopeID: "test-envelope-id",
			Type:       "slash_commands",
		},
	}

	payload := d.buildSlashCommandPayload(event)
	assert.NotNil(t, payload)
	assert.Equal(t, "slash_command", payload["event_type"])
	assert.Equal(t, "/test", payload["command"])
	assert.Equal(t, "arg1 arg2", payload["text"])
	assert.Equal(t, "U12345", payload["user_id"])
	assert.Equal(t, "C12345", payload["channel_id"])
	assert.Equal(t, "T12345", payload["team_id"])
	assert.Equal(t, "https://hooks.slack.com/commands/T12345/123", payload["response_url"])
	assert.Equal(t, "trigger_123", payload["trigger_id"])
	assert.Equal(t, "testuser", payload["user_name"])
	assert.Equal(t, "general", payload["channel_name"])
}

func TestBuildSlashCommandPayloadBadData(t *testing.T) {
	t.Parallel()

	d := &slacksocketDriver{}

	event := socketmode.Event{
		Type:    socketmode.EventTypeSlashCommand,
		Data:    "not a SlashCommand",
		Request: &socketmode.Request{EnvelopeID: "test-env-id", Type: "slash_commands"},
	}

	payload := d.buildSlashCommandPayload(event)
	assert.Nil(t, payload, "should return nil when data is not SlashCommand")
}

func TestBuildSlashCommandPayloadNilRequest(t *testing.T) {
	t.Parallel()

	d := &slacksocketDriver{}

	event := socketmode.Event{
		Type: socketmode.EventTypeSlashCommand,
		Data: slack.SlashCommand{
			Command: "/test",
			Text:    "some text",
		},
		// Request is nil
	}

	payload := d.buildSlashCommandPayload(event)
	assert.Nil(t, payload, "should return nil when request is missing")
}

func TestMatchCollapsedEventsWithSlashCommandPayload(t *testing.T) {
	t.Parallel()

	d := &slacksocketDriver{}
	d.collapsedEvents = map[string]interface{}{
		"slack.slash_command": []interface{}{
			map[string]interface{}{
				"match": map[string]interface{}{
					"event_type": "slash_command",
					"command":    "/test",
				},
			},
		},
	}

	payload := map[string]interface{}{
		"event_type":   "slash_command",
		"command":      "/test",
		"text":         "arg1 arg2",
		"user_id":      "U12345",
		"channel_id":   "C12345",
		"team_id":      "T12345",
		"user_name":    "testuser",
		"channel_name": "general",
	}

	assert.True(t, d.matchCollapsedEvents(payload), "slash command should match condition")
}

func TestMatchCollapsedEventsWithSlashCommandPayloadNoMatch(t *testing.T) {
	t.Parallel()

	d := &slacksocketDriver{}
	d.collapsedEvents = map[string]interface{}{
		"slack.slash_command": []interface{}{
			map[string]interface{}{
				"match": map[string]interface{}{
					"event_type": "slash_command",
					"command":    "/test",
				},
			},
		},
	}

	payload := map[string]interface{}{
		"event_type":   "slash_command",
		"command":      "/other",
		"text":         "some text",
		"user_id":      "U12345",
		"channel_id":   "C12345",
		"user_name":    "testuser",
		"channel_name": "general",
	}

	assert.False(t, d.matchCollapsedEvents(payload), "/other should not match /test condition")
}

func TestMatchCollapsedEventsWithSlashCommandPayloadAllEvents(t *testing.T) {
	t.Parallel()

	d := &slacksocketDriver{}
	d.collapsedEvents = map[string]interface{}{
		"slack.slash_command": []interface{}{
			map[string]interface{}{
				"match": map[string]interface{}{
					"event_type": "slash_command",
				},
			},
		},
	}

	payload := map[string]interface{}{
		"event_type": "slash_command",
		"command":    "/any_command",
	}

	assert.True(t, d.matchCollapsedEvents(payload),
		"slash command should match when condition only requires event_type")
}
