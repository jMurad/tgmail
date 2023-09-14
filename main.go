package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aarzilli/sandblast"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	tgb "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"golang.org/x/net/html"
)

func init() {
	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func main() {
	tgbot()
}

type TeleBot struct {
	Bot     *tgb.BotAPI
	Updates tgb.UpdatesChannel
	ChatID  int64
}

func (tb *TeleBot) TBInit() {
	//Создаем бота
	token := os.Getenv("TELEG_TOKEN")

	var err error
	tb.ChatID, err = strconv.ParseInt(os.Getenv("TELEG_CHAID"), 10, 64)
	if err != nil {
		panic(err)
	}

	tb.Bot, err = tgb.NewBotAPI(token)
	if err != nil {
		panic(err)
	}

	if os.Getenv("TELEG_DEBAG") == "true" {
		tb.Bot.Debug = true
	}

	log.Printf("Authorized on account %s", tb.Bot.Self.UserName)

	u := tgb.NewUpdate(0)
	u.Timeout = 60

	//Получаем обновления от бота
	tb.Updates = tb.Bot.GetUpdatesChan(u)
}

func (tb *TeleBot) sendMsg(md bool, id int64, text string, kb interface{}) {
	msg := tgb.NewMessage(id, text)
	msg.ReplyMarkup = kbMore()

	if md {
		msg.ParseMode = tgb.ModeHTML
	}

	if len(text) < 2000 {
		if _, err := tb.Bot.Send(msg); err != nil {
			log.Panic(err)
		}
	} else {
		msg.Text = text[:2000]
		if _, err := tb.Bot.Send(msg); err != nil {
			log.Panic(err)
		}

		size := 2000
		allowedSize := 4095
		for {
			if size > len(text) {
				break
			}
			if size+allowedSize >= len(text) {
				msg.Text = text[size:]
				msg.ReplyMarkup = kbLess()
				if _, err := tb.Bot.Send(msg); err != nil {
					log.Panic(err)
				}
			} else {
				msg.Text = text[size : size+allowedSize]
				if _, err := tb.Bot.Send(msg); err != nil {
					log.Panic(err)
				}
			}
			size += allowedSize
		}
	}
}

func kbMore() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Раскрыть"),
		),
	)
}

func kbLess() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Скрыть"),
		),
	)
}

func (tb *TeleBot) RunBot(envelopes chan Envelope) {
	for {
		select {
		case update := <-tb.Updates:
			if update.Message != nil {

			}
		case envelope := <-envelopes:
			tb.sendMsg(true, tb.ChatID, envelope.message, kbMore())
		}

	}

	// for update := range tb.Updates {
	// 	if update.Message != nil { // If we got a message
	// 		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

	// 		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
	// 		msg.ReplyToMessageID = update.Message.MessageID

	// 		bot.Send(msg)
	// 	}
	// }
}

func tgbot() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEG_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	chatID, err := strconv.ParseInt(os.Getenv("TELEG_CHAID"), 10, 64)
	if err != nil {
		panic(err)
	}

	msg := tgbotapi.NewMessage(chatID, "")
	bot.Send(msg)

	ticker := time.NewTicker(time.Minute)
	for ; true; <-ticker.C {
		envs := mailReceiver()
		if len(envs) != 0 {
			for _, e := range envs {
				report := e.message
				i := 0
				lenReport := 4095
				msg := tgbotapi.NewMessage(chatID, "")
				msg.ParseMode = tgbotapi.ModeHTML
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("Больше..."),
					),
				)
				for {
					if i > len(report) {
						break
					}
					if i+lenReport >= len(report) {
						msg.Text = report[i:]
						bot.Send(msg)
						// tb.sendMsg(true, id, report[i:], kb.MainMenu)
					} else {
						msg.Text = report[i : i+lenReport]
						bot.Send(msg)
						// tb.sendMsg(true, id, report[i:i+lenReport], kb.MainMenu)
					}
					i += lenReport
				}
			}

		}
	}
}

type Envelope struct {
	date        time.Time
	to          []string
	from        []string
	subject     string
	message     string
	attachments []io.Reader
}

func mailReceiver() []Envelope {
	// log.SetOutput(io.Discard)
	log.Println("Connecting to server...")

	// Connect to server
	c, err := client.DialTLS("imap.yandex.com:993", nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected")

	// Don't forget to logout
	defer c.Logout()

	// Login
	if err := c.Login("sh0ma04@yandex.ru", "htglhbaigtvybtcn"); err != nil {
		log.Fatal(err)
	}
	log.Println("Logged in")

	// Select a mailbox
	if _, err := c.Select("INBOX", false); err != nil {
		log.Fatal(err)
	}

	// Set search criteria
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	ids, err := c.Search(criteria)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("IDs found:", ids)
	if len(ids) == 0 {
		log.Println("No Ids")
		return []envelope{}
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(ids...)

	var section imap.BodySectionName
	items := []imap.FetchItem{section.FetchItem()}

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)

	go func() {
		done <- c.Fetch(seqSet, items, messages)
	}()

	var envlps []envelope
	for msg := range messages {
		var envlp envelope

		if msg == nil {
			log.Fatal("Server didn't returned message")
		}

		r := msg.GetBody(&section)
		if r == nil {
			log.Fatal("Server didn't returned message body")
		}

		mr, err := mail.CreateReader(r)
		if err != nil {
			log.Fatal(err)
		}

		// Print some info about the message
		header := mr.Header
		if date, err := header.Date(); err == nil {
			// log.Println("Date:", date)
			envlp.date = date
		}
		if from, err := header.AddressList("From"); err == nil {
			// log.Println("From:", from)
			for _, fr := range from {
				envlp.from = append(envlp.from, fr.Name+" | "+fr.Address)
			}
		}
		if to, err := header.AddressList("To"); err == nil {
			// log.Println("To:", to)
			for _, t := range to {
				envlp.to = append(envlp.to, t.Name+" | "+t.Address)
			}
		}
		if subject, err := header.Subject(); err == nil {
			// log.Println("Subject:", subject)
			envlp.subject = subject
		}

		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Fatal(err)
			}

			switch h := p.Header.(type) {
			case *mail.InlineHeader:
				// This is the message's text (can be plain-text or HTML)
				b, _ := io.ReadAll(p.Body)
				// log.Printf("Got text: %v\n", string(b))
				envlp.message += string(b)

			case *mail.AttachmentHeader:
				// This is an attachment
				filename, _ := h.Filename()
				log.Printf("Got attachment: %v\n", filename)
				// Create file with attachment name
				file, err := os.Create(filename)
				if err != nil {
					log.Fatal(err)
				}
				// using io.Copy instead of io.ReadAll to avoid insufficient memory issues
				_, err = io.Copy(file, p.Body)
				if err != nil {
					log.Fatal(err)
				}
				file.Close()
				envlp.attachments = append(envlp.attachments, p.Body)
			}
		}

		node, err := html.Parse(strings.NewReader(envlp.message))
		if err != nil {
			log.Fatal("Parsing error: ", err)
		}
		title, text, _ := sandblast.Extract(node, sandblast.KeepLinks)
		fmt.Printf("Title: %s\n%s", title, text)

		envlp.message = text

		envlps = append(envlps, envlp)
	}

	if err := <-done; err != nil {
		log.Fatal(err)
	}

	log.Println("Done!")

	return envlps
}
