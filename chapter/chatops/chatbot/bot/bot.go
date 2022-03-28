// Package bot defines a basic slack bot that can listen for app mention events for our bot
// and send the message to a handler to handle the interaction.
package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

// HandleFunc receive the user who sent a message and the message. It can then use api or client to respond to said message.
type HandleFunc func(ctx context.Context, m Message)

// Message details information about a message that was sent in an AppMention event.
type Message struct {
	// User has the user information on who mentioned the bot.
	User *slack.User
	// AppMention gives information on the event.
	AppMention *slackevents.AppMentionEvent
	// Text gives the text of the message without the @User stuff. If you want the full message, see AppMention.
	Text string
}

type register struct {
	r *regexp.Regexp
	h HandleFunc
}

// Bot provides a slack bot for listening to slack channels.
type Bot struct {
	api    *slack.Client
	client *socketmode.Client
	ctx    context.Context
	cancel context.CancelFunc

	defaultHandler HandleFunc
	reg            []register
}

// New creates a new Bot.
func New(api *slack.Client, client *socketmode.Client) (*Bot, error) {
	ctx, cancel := context.WithCancel(context.Background())
	b := &Bot{
		api:    api,
		client: client,
		ctx:    ctx,
		cancel: cancel,
	}
	return b, nil
}

// Start starts listening for events from the socket client. This blocks until the client dies
// or Stop() is called.
func (b *Bot) Start() {
	go b.loop()

	b.client.RunContext(b.ctx)
}

// Stop stops the bot. The bot cannot be reused after this.
func (b *Bot) Stop() {
	b.cancel()
}

// Register registers a function for handling a message to the bot. The regex is checked in the order
// that it is added. A nil regexp is considered the default handler. Only 1 default handler can be added and
// is always the choice of last resort.
func (b *Bot) Register(r *regexp.Regexp, h HandleFunc) {
	if h == nil {
		panic("HandleFunc cannot be nil")
	}
	if r == nil {
		if b.defaultHandler != nil {
			panic("cannot add two default handles")
		}
		b.defaultHandler = h
		return
	}
	b.reg = append(b.reg, register{r, h})
}

// loop is the event loop.
func (b *Bot) loop() {
	for {
		ctx := context.Background()
		select {
		case <-b.ctx.Done():
			return
		case evt := <-b.client.Events:
			switch evt.Type {
			case socketmode.EventTypeConnecting, socketmode.EventTypeConnected:
			case socketmode.EventTypeConnectionError:
				log.Println("connection failed. Retrying later...")
			case socketmode.EventTypeEventsAPI:
				data, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					log.Printf("bug: got %T which should be a slackevents.EventsAPIEvent", evt.Data)
					continue
				}
				b.client.Ack(*evt.Request)
				go b.appMentioned(ctx, data)
			}
		}
	}
}

// appMentioned handles an event socketmode.EventTypeEventsAPI that had a .Data that is a slackevents.EventsAPIEvent that eventually
// is a AppMentionEvent. This has a crazy amount of freaking event wrapping.
func (b *Bot) appMentioned(ctx context.Context, data slackevents.EventsAPIEvent) {
	switch data.Type {
	case slackevents.CallbackEvent:
		callback := data.Data.(*slackevents.EventsAPICallbackEvent)

		switch ev := data.InnerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			if ev.BotID != "" {
				_, _, err := b.api.PostMessage(ev.Channel, slack.MsgOptionText("I don't talk to other bots", false))
				if err != nil {
					log.Printf("failed posting message: %v", err)
				}
				return
			}

			msg, err := b.makeMsg(callback, ev)
			if err != nil {
				log.Println(err)
				return
			}
			for _, reg := range b.reg {
				if reg.r.MatchString(msg.Text) {
					reg.h(ctx, msg)
					return
				}
			}
			if b.defaultHandler != nil {
				b.defaultHandler(ctx, msg)
			}
		}
	default:
		b.client.Debugf("unsupported Events API event received")
	}
}

// makeMsg extracts the user and text from an event and callback into a Message type.
func (b *Bot) makeMsg(callback *slackevents.EventsAPICallbackEvent, event *slackevents.AppMentionEvent) (Message, error) {
	user, err := b.api.GetUserInfo(event.User)
	if err != nil {
		return Message{}, fmt.Errorf("could not get user data: %w", err)
	}
	rm := rawMessage{}
	if err := json.Unmarshal(*callback.InnerEvent, &rm); err != nil {
		return Message{}, fmt.Errorf("bot received a callback with no InnerEvent: %w", err)
	}
	return Message{User: user, AppMention: event, Text: rm.getText()}, nil
}

// rawMessage is used to covert a slackevents.EventsAPICallbackEvent.InnerEvent, which is the raw JSON, into
// a form in which I can abstract the message sent by the user without things like @user in it. This is
// not carried into the exposed Go type and I want to use Slack's pre-filtering instead of doing it myself.
type rawMessage struct {
	Blocks []interface{}
}

// getText gets the text without all the extra @user stuff.
func (r rawMessage) getText() string {
	for _, block := range r.Blocks {
		blockReal := block.(map[string]interface{})
		if blockReal["type"] != "rich_text" {
			continue
		}
		elements := blockReal["elements"].([]interface{})
		for _, el := range elements {
			elReal := el.(map[string]interface{})
			if elReal["type"].(string) != "rich_text_section" {
				continue
			}
			subElements := elReal["elements"].([]interface{})
			for _, subEl := range subElements {
				subElReal := subEl.(map[string]interface{})
				if subElReal["type"] != "text" {
					continue
				}
				return strings.TrimSpace(subElReal["text"].(string))
			}
		}
	}
	return ""
}
