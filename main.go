package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

func main() {

	
	godotenv.Load(".env")

	token := os.Getenv("SLACK_AUTH_TOKEN")
	appToken := os.Getenv("SLACK_APP_TOKEN")
	
	client := slack.New(token, slack.OptionDebug(true), slack.OptionAppLevelToken(appToken))

	socket := socketmode.New(
		client,
		socketmode.OptionDebug(true),
		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func(ctx context.Context, client *slack.Client, socket *socketmode.Client) {
		
		for {
			select {
			case <-ctx.Done():
				log.Println("Shutting down socketmode listener")
				return
			case event := <-socket.Events:
				switch event.Type {
				case socketmode.EventTypeEventsAPI:
					eventsAPI, ok := event.Data.(slackevents.EventsAPIEvent)
					if !ok {
						log.Printf("Could not type cast the event to the EventsAPIEvent: %v\n", event)
						continue
					}
					socket.Ack(*event.Request)
					err := HandleEventMessage(eventsAPI, client)
					if err != nil {
						log.Fatal(err)
					}
				}
			}
		}
	}(ctx, client, socket)

	socket.Run()
}

func HandleEventMessage(event slackevents.EventsAPIEvent, client *slack.Client) error {
	switch event.Type {
	case slackevents.CallbackEvent:

		innerEvent := event.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			err := HandleAppMentionEventToBot(ev, client)
			if err != nil {
				return err
			}
		}
	default:
		return errors.New("unsupported event type")
	}
	return nil
}

func HandleAppMentionEventToBot(event *slackevents.AppMentionEvent, client *slack.Client) error {
	user, err := client.GetUserInfo(event.User)
	if err != nil {
		return err
	}
	text := strings.ToLower(event.Text)

	attachment := slack.Attachment{}
	attachment.Fields = []slack.AttachmentField{
		{
			Title: "Date",
			Value: time.Now().String(),
		}, {
			Title: "Initializer",
			Value: user.Name,
		},
	}
	if strings.Contains(text, "hello") || strings.Contains(text, "hi") {
		attachment.Text = fmt.Sprintf("Hello %s", user.Name)
		attachment.Pretext = "Greetings"
		attachment.Color = "#4af030"
	} else if strings.Contains(text, "weather") {
		attachment.Text = fmt.Sprintf("Weather is sunny today. %s", user.Name)
		attachment.Pretext = "How can I be of service"
		attachment.Color = "#4af030"
	} else {
		
		attachment.Text = fmt.Sprintf("I am good. How are you %s?", user.Name)
		attachment.Pretext = "How can I be of service"
		attachment.Color = "#4af030"
	}
	_, _, err = client.PostMessage(event.Channel, slack.MsgOptionAttachments(attachment))
	if err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}
	return nil
}