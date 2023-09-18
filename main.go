package main

import (
	"log"
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
	var telebot TeleBot
	var rec Receiver

	telebot.TBInit()
	rec.Init()

	envch := make(chan Envelope)

	go func(ec chan Envelope, rec *Receiver) {
		ticker := time.NewTicker(time.Minute)
		for ; true; <-ticker.C {
			envelopes := rec.MailReceiver()
			for _, e := range envelopes {
				ec <- e
			}
		}
	}(envch, &rec)

	telebot.RunBot(envch)
}
