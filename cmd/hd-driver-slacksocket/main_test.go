// Copyright 2026 Chun Huang (Charles).
//
// This Source Code Form is dual-licensed.
// By default, this file is licensed under the GNU Affero General Public License v3.0.
// If you have a separate written commercial agreement, you may use this file under those terms instead.

package main

import (
	"errors"
	"testing"

	"github.com/slack-go/slack/slackevents"
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
