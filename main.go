package main

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func init() {
	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func main() {
	dat, _ := os.ReadFile(".mtext")
	tt = string(dat)
	// log.Println(utf8.ValidString(tt))

	var telebot TeleBot

	telebot.TBInit()

	envch := make(chan Envelope)

	go func(ec chan Envelope) {
		ticker := time.NewTicker(time.Minute)
		for ; true; <-ticker.C {
			envelopes := mailReceiver()
			for _, e := range envelopes {
				ec <- e
			}
		}
	}(envch)

	telebot.RunBot(envch)
}
