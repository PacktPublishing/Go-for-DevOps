package main

import (
	"flag"
	"log"
	"os"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/11/chatbot/bot"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/11/chatbot/internal/handlers"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/11/ops/client"

	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

var (
	opsAddr = flag.String("opsAddr", "127.0.0.1:7000", "The address the Ops service runs on.")
	debug   = flag.Bool("debug", false, "If turned on will log debug information to the screen.")
)

func main() {
	flag.Parse()

	err := godotenv.Load()
	if err != nil {
		panic("could not load .env file")
	}

	api := slack.New(
		os.Getenv("AUTH_TOKEN"),
		slack.OptionAppLevelToken(os.Getenv("APP_TOKEN")),
		slack.OptionDebug(*debug),
	)

	smClient := socketmode.New(
		api,
		socketmode.OptionDebug(*debug),
		socketmode.OptionLog(
			log.New(
				os.Stdout,
				"socketmode: ",
				log.Lshortfile|log.LstdFlags,
			),
		),
	)

	opsClient, err := client.New(*opsAddr)
	if err != nil {
		panic(err)
	}

	b, err := bot.New(api, smClient)
	if err != nil {
		panic(err)
	}
	h := handlers.Ops{OpsClient: opsClient, API: api, SMClient: smClient}
	h.Register(b)
	log.Println("Bot started")
	b.Start()

	panic("Bot stopped functioning")
}
