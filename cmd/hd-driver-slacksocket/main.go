// Copyright 2026 Chun Huang (Charles).
//
// This Source Code Form is dual-licensed.
// By default, this file is licensed under the GNU Affero General Public License v3.0.
// If you have a separate written commercial agreement, you may use this file under those terms instead.

// Package hd-driver-slacksocket enables Honeydipper to receive Slack events
// via Socket Mode (outbound WebSocket) instead of HTTP webhooks.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/honeydipper/honeydipper/v4/pkg/dipper"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

var (
	// ErrMissingAppToken is raised when the Slack app-level token is not configured.
	ErrMissingAppToken = errors.New("missing Slack app token (xapp-)")
	// ErrMissingBotToken is raised when the Slack bot token is not configured.
	ErrMissingBotToken = errors.New("missing Slack bot token (xoxb-)")
	// ErrInvalidAppToken is raised when the app token format is invalid.
	ErrInvalidAppToken = errors.New("invalid Slack app token format, must start with xapp-")
	// ErrInvalidBotToken is raised when the bot token format is invalid.
	ErrInvalidBotToken = errors.New("invalid Slack bot token format, must start with xoxb-")
)

const (
	// maxReconnectRetries is the maximum number of reconnection attempts.
	maxReconnectRetries = 10
	// reconnectBaseDelay is the initial delay for exponential backoff (in seconds).
	reconnectBaseDelay = 1
	// reconnectMaxDelay is the maximum delay between reconnection attempts (in seconds).
	reconnectMaxDelay = 60
)

// slacksocketDriver holds all driver state.
type slacksocketDriver struct {
	*dipper.Driver

	mu sync.Mutex

	appToken string
	botToken string

	collapsedEvents map[string]interface{}

	socketClient *socketmode.Client
	slackClient  *slack.Client
	cancel       context.CancelFunc
	stopCh       chan struct{}
}

var driver = &slacksocketDriver{}

func initFlags() {
	flag.Usage = func() {
		fmt.Printf("%s [ -h ] <service name>\n", os.Args[0])
		fmt.Printf("    This driver supports receiver service.\n")
		fmt.Printf("  This program provides honeydipper with capability of receiving Slack events via Socket Mode.\n")
	}
}

func main() {
	initFlags()
	flag.Parse()

	driver.Driver = dipper.NewDriver(os.Args[1], "slacksocket")
	if driver.Service == "receiver" {
		driver.Start = driver.start
		driver.Drain = driver.drain
		driver.Reload = driver.loadOptions
	}
	driver.Run()
}

func (d *slacksocketDriver) loadOptions(m *dipper.Message) {
	log := d.GetLogger()

	appToken, ok := d.GetOptionStr("data.app_token")
	if !ok || appToken == "" {
		log.Panicf("[%s] %v", d.Service, ErrMissingAppToken)
	}
	if len(appToken) < 5 || appToken[:5] != "xapp-" {
		log.Panicf("[%s] %v", d.Service, ErrInvalidAppToken)
	}

	botToken, ok := d.GetOptionStr("data.bot_token")
	if !ok || botToken == "" {
		log.Panicf("[%s] %v", d.Service, ErrMissingBotToken)
	}
	if len(botToken) < 5 || botToken[:5] != "xoxb-" {
		log.Panicf("[%s] %v", d.Service, ErrInvalidBotToken)
	}

	d.mu.Lock()
	d.appToken = appToken
	d.botToken = botToken

	events, ok := d.GetOption("dynamicData.collapsedEvents")
	if ok {
		if evtMap, ok := events.(map[string]interface{}); ok {
			d.collapsedEvents = evtMap
		} else {
			d.collapsedEvents = nil
		}
	} else {
		d.collapsedEvents = nil
	}
	d.mu.Unlock()

	log.Debugf("[%s] Slack Socket Mode driver initialized", d.Service)
	log.Debugf("[%s] collapsed events: %+v", d.Service, d.collapsedEvents)
}

func (d *slacksocketDriver) start(m *dipper.Message) {
	d.loadOptions(m)

	d.stopCh = make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel

	go d.eventLoop(ctx)

	d.GetLogger().Infof("[%s] Slack Socket Mode receiver started", d.Service)
}

func (d *slacksocketDriver) drain(m *dipper.Message) {
	log := d.GetLogger()

	if d.cancel != nil {
		d.cancel()
		d.cancel = nil
	}

	if d.stopCh != nil {
		<-d.stopCh
		d.stopCh = nil
	}

	log.Infof("[%s] Slack Socket Mode receiver drained", d.Service)
}

func (d *slacksocketDriver) eventLoop(ctx context.Context) {
	defer func() {
		if d.stopCh != nil {
			close(d.stopCh)
		}
	}()

	backoff := reconnectBaseDelay

	for attempt := 0; ; attempt++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if attempt > 0 {
			if d.State != dipper.DriverStateAlive {
				return
			}

			d.GetLogger().Debugf("[%s] reconnecting (attempt %d/%d)...", d.Service, attempt, maxReconnectRetries)

			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(backoff) * time.Second):
			}

			backoff *= 2
			if backoff > reconnectMaxDelay {
				backoff = reconnectMaxDelay
			}
		}

		if attempt >= maxReconnectRetries {
			d.GetLogger().Panicf("[%s] max reconnection attempts reached", d.Service)

			return
		}

		d.connectAndRun(ctx)
	}
}

func (d *slacksocketDriver) connectAndRun(ctx context.Context) {
	log := d.GetLogger()

	d.mu.Lock()
	d.slackClient = slack.New(
		d.botToken,
		slack.OptionAppLevelToken(d.appToken),
	)
	d.socketClient = socketmode.New(
		d.slackClient,
		socketmode.OptionDebug(false),
	)
	d.mu.Unlock()

	log.Infof("[%s] connected to Slack Socket Mode", d.Service)

	done := make(chan struct{})

	go func() {
		defer close(done)

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-d.socketClient.Events:
				if !ok {
					return
				}
				d.handleSocketEvent(event)
			}
		}
	}()

	if err := d.socketClient.RunContext(ctx); err != nil {
		log.Debugf("[%s] socket mode client exited: %v", d.Service, err)
	}

	<-done
}

func (d *slacksocketDriver) handleSocketEvent(event socketmode.Event) {
	log := d.GetLogger()

	switch event.Type {
	case socketmode.EventTypeEventsAPI:
		d.handleEventsAPIBackground(event)
	case socketmode.EventTypeInteractive:
		d.handleInteractiveEvent(event)
	case socketmode.EventTypeSlashCommand:
		d.handleSlashCommandEvent(event)
	case socketmode.EventTypeDisconnect:
		log.Infof("[%s] received disconnect message from Slack", d.Service)
	case socketmode.EventTypeHello:
		log.Debugf("[%s] received hello from Slack", d.Service)
	case socketmode.EventTypeConnected:
		log.Infof("[%s] connected to Slack Socket Mode", d.Service)
	default:
		log.Debugf("[%s] unhandled socket event type: %s", d.Service, event.Type)
	}
}

func (d *slacksocketDriver) handleEventsAPIBackground(event socketmode.Event) {
	log := d.GetLogger()

	// Ack the envelope immediately
	d.socketClient.Ack(*event.Request)

	evt := event.Data
	eventsAPIEvent, ok := evt.(slackevents.EventsAPIEvent)
	if !ok {
		log.Debugf("[%s] unexpected events API event type: %T", d.Service, evt)

		return
	}

	innerEvent := eventsAPIEvent.InnerEvent
	log.Debugf("[%s] received event: type=%s, event_type=%s", d.Service, eventsAPIEvent.Type, innerEvent.Type)

	payload := d.buildEventPayload(innerEvent)
	if payload == nil {
		return
	}

	if !d.matchCollapsedEvents(payload) {
		log.Debugf("[%s] event %s did not match any collapsed event rule", d.Service, innerEvent.Type)

		return
	}

	_ = d.EmitEvent(map[string]interface{}{
		"events": []interface{}{d.Name + "." + innerEvent.Type},
		"data":   payload,
	})
}

func (d *slacksocketDriver) handleInteractiveEvent(event socketmode.Event) {
	log := d.GetLogger()

	// Ack the envelope immediately (required by Slack for interactive callbacks)
	d.socketClient.Ack(*event.Request)

	payload := d.buildInteractivePayload(event)
	if payload == nil {
		return
	}

	callback, _ := event.Data.(slack.InteractionCallback)
	log.Debugf("[%s] received interactive event: type=%s, callback_id=%s", d.Service, callback.Type, callback.CallbackID)

	if !d.matchCollapsedEvents(payload) {
		log.Debugf("[%s] interactive event %s did not match any collapsed event rule", d.Service, callback.Type)

		return
	}

	_ = d.EmitEvent(map[string]interface{}{
		"events": []interface{}{d.Name + ".interactive"},
		"data":   payload,
	})
}

func (d *slacksocketDriver) handleSlashCommandEvent(event socketmode.Event) {
	log := d.GetLogger()

	// Ack the envelope immediately (required by Slack for slash commands)
	d.socketClient.Ack(*event.Request)

	payload := d.buildSlashCommandPayload(event)
	if payload == nil {
		return
	}

	cmd, _ := event.Data.(slack.SlashCommand)
	log.Debugf("[%s] received slash command: command=%s, text=%s, user=%s", d.Service, cmd.Command, cmd.Text, cmd.UserID)

	if !d.matchCollapsedEvents(payload) {
		log.Debugf("[%s] slash command %s did not match any collapsed event rule", d.Service, cmd.Command)

		return
	}

	_ = d.EmitEvent(map[string]interface{}{
		"events": []interface{}{d.Name + ".slash_command"},
		"data":   payload,
	})
}

// buildInteractivePayload extracts data from a socketmode interactive event and
// returns a payload map, or nil if the event data is invalid or missing a request.
func (d *slacksocketDriver) buildInteractivePayload(event socketmode.Event) map[string]interface{} {
	if event.Request == nil {
		return nil
	}

	callback, ok := event.Data.(slack.InteractionCallback)
	if !ok {
		return nil
	}

	payload := map[string]interface{}{
		"event_type":    "interactive",
		"callback_type": string(callback.Type),
		"callback_id":   callback.CallbackID,
		"action_ts":     callback.ActionTs,
		"channel":       callback.Channel.ID,
		"user":          callback.User.ID,
		"team":          callback.Team.ID,
		"response_url":  callback.ResponseURL,
		"trigger_id":    callback.TriggerID,
	}

	// Add message data if present (e.g., for button clicks on messages)
	messageData, err := json.Marshal(callback.Message)
	if err == nil {
		var msg interface{}
		if err := json.Unmarshal(messageData, &msg); err == nil && msg != nil {
			payload["message"] = msg
		}
	}

	// Add view data if present (e.g., for modal submissions)
	viewData, err := json.Marshal(callback.View)
	if err == nil {
		var view interface{}
		if err := json.Unmarshal(viewData, &view); err == nil && view != nil {
			payload["view"] = view
		}
	}

	// Add actions data if present (e.g., for block actions)
	actionsData, err := json.Marshal(callback.ActionCallback)
	if err == nil {
		var actions interface{}
		if err := json.Unmarshal(actionsData, &actions); err == nil && actions != nil {
			payload["actions"] = actions
		}
	}

	return payload
}

// buildSlashCommandPayload extracts data from a socketmode slash command event and
// returns a payload map, or nil if the event data is invalid or missing a request.
func (d *slacksocketDriver) buildSlashCommandPayload(event socketmode.Event) map[string]interface{} {
	if event.Request == nil {
		return nil
	}

	cmd, ok := event.Data.(slack.SlashCommand)
	if !ok {
		return nil
	}

	return map[string]interface{}{
		"event_type":   "slash_command",
		"command":      cmd.Command,
		"text":         cmd.Text,
		"user_id":      cmd.UserID,
		"channel_id":   cmd.ChannelID,
		"team_id":      cmd.TeamID,
		"response_url": cmd.ResponseURL,
		"trigger_id":   cmd.TriggerID,
		"user_name":    cmd.UserName,
		"channel_name": cmd.ChannelName,
	}
}

func (d *slacksocketDriver) buildEventPayload(innerEvent slackevents.EventsAPIInnerEvent) map[string]interface{} {
	payload := map[string]interface{}{
		"event_type": innerEvent.Type,
	}

	data, err := json.Marshal(innerEvent.Data)
	if err == nil {
		var eventData interface{}
		if err := json.Unmarshal(data, &eventData); err == nil && eventData != nil {
			payload["event"] = eventData
		}
	}

	return payload
}

func (d *slacksocketDriver) matchCollapsedEvents(payload map[string]interface{}) bool {
	d.mu.Lock()
	events := d.collapsedEvents
	d.mu.Unlock()

	if events == nil {
		return true
	}

	for _, evt := range events {
		branches, ok := evt.([]interface{})
		if !ok {
			continue
		}
		for _, branch := range branches {
			branchMap, ok := branch.(map[string]interface{})
			if !ok {
				continue
			}
			condition, ok := branchMap["match"]
			if !ok {
				continue
			}
			if dipper.CompareAll(payload, condition) {
				return true
			}
		}
	}

	return false
}
