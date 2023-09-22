package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/aarzilli/sandblast"
	tgb "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/net/html"
)

type TeleBot struct {
	Bot      *tgb.BotAPI
	Updates  tgb.UpdatesChannel
	ChatID   int64
	MsgStore MessageStore
	Debug    bool
}

func (tb *TeleBot) Init() {
	var err error
	tb.ChatID, err = strconv.ParseInt(os.Getenv("TELEG_CHATID"), 10, 64)
	if err != nil {
		panic(err)
	}

	token := os.Getenv("TELEG_TOKEN")
	tb.Bot, err = tgb.NewBotAPI(token)
	if err != nil {
		panic(err)
	}

	if os.Getenv("TELEG_DEBAG") == "true" {
		tb.Bot.Debug = true
		tb.Debug = true
	}

	log.Println("Authorized on account", tb.Bot.Self.UserName)

	u := tgb.NewUpdate(0)
	u.Timeout = 60

	//Получаем обновления от бота
	tb.Updates = tb.Bot.GetUpdatesChan(u)

	tb.MsgStore.Messages = make(map[int]string)
}

func (tb *TeleBot) editMsg(mid, index int, cmd string) {
	msgText, _ := tb.MsgStore.get(mid)
	text, kb := paginator(cmd, msgText, index)

	msg := tgb.NewEditMessageText(tb.ChatID, mid, text)
	msg.ReplyMarkup = &kb
	msg.ParseMode = tgb.ModeHTML

	if _, err := tb.Bot.Send(msg); err != nil {
		log.Panic(err)
	}
}

func (tb *TeleBot) sendMsg(env Envelope) {
	id := rand.Intn(10000)
	tb.MsgStore.add(id, env.message)
	var text string
	if env.htmlType {
		node, err := html.Parse(strings.NewReader(env.message))
		if err != nil {
			log.Fatal("Parsing error: ", err)
		}
		_, text, _ = sandblast.Extract(node, sandblast.KeepLinks)
		text = beautify(text, env.subject, env.from)
	} else {
		text = beautify(env.message, env.subject, env.from)
	}
	msg := tgb.NewMessage(tb.ChatID, text)

	if lensafe(text) > lessText {
		msg.Text = slicer(text, 0, lessText)
		// kb := inlineKb(0, 0)
		kb := webAppKb(fmt.Sprintf("https://88.210.9.244.sslip.io/%d", id))
		kb.InlineKeyboard = append(kb.InlineKeyboard, fileKb(env.filenames).InlineKeyboard...)
		// kb.InlineKeyboard = append(kb.InlineKeyboard, kb2.InlineKeyboard...)
		msg.ReplyMarkup = kb
	} else if len(env.filenames) > 0 {
		msg.ReplyMarkup = fileKb(env.filenames)
	}

	msg.ParseMode = tgb.ModeHTML

	_, err := tb.Bot.Send(msg)
	if err != nil {
		log.Panic(err)
	}

}

func (tb *TeleBot) sendFile(id int, filename string) {
	doc := tgb.NewDocument(tb.ChatID, tgb.FilePath(filename))
	doc.Caption = strings.Split(filename, "/")[len(strings.Split(filename, "/"))-1]
	doc.ReplyToMessageID = id
	_, err := tb.Bot.Send(doc)

	if err != nil {
		log.Panic(err)
	}
}

func (tb *TeleBot) RunBot(envelopes chan Envelope) {
	for {
		select {
		case update := <-tb.Updates:
			if update.Message != nil {
				fmt.Println(">", update.Message.Text)
				tb.ChatID = update.Message.Chat.ID

			} else if update.CallbackQuery != nil {
				callback := tgb.NewCallback(update.CallbackQuery.ID, "")
				if _, err := tb.Bot.Request(callback); err != nil {
					panic(err)
				}
				if strings.Contains(update.CallbackQuery.Data, ":pag:") {
					data := strings.Split(update.CallbackQuery.Data, ":pag:")
					index, _ := strconv.Atoi(data[0])
					tb.editMsg(update.CallbackQuery.Message.MessageID, index, data[1])
				} else if strings.Contains(update.CallbackQuery.Data, "file:") {
					path := strings.Split(update.CallbackQuery.Data, "file:")[1]
					tb.sendFile(update.CallbackQuery.Message.MessageID, path)
				}
			}
		case envelope := <-envelopes:
			tb.sendMsg(envelope)
		}
	}
}
