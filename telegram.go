package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	tgb "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type MessageStore struct {
	Messages map[int]string
	Queue    [sizeMsgStore]int
	index    int
}

func (m *MessageStore) add(id int, text string) {
	m.Messages[id] = text
	if m.index >= sizeMsgStore {
		delete(m.Messages, m.Queue[0])
		for i := 1; i < 50; i++ {
			m.Queue[i-1] = m.Queue[i]
		}
		m.index--
	}
	m.Queue[m.index] = id
	m.index++
}

func (m *MessageStore) get(id int) string {
	return m.Messages[id]
}

type TeleBot struct {
	Bot      *tgb.BotAPI
	Updates  tgb.UpdatesChannel
	ChatID   int64
	MsgStore MessageStore
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
	}

	log.Printf("Authorized on account %s", tb.Bot.Self.UserName)

	u := tgb.NewUpdate(0)
	u.Timeout = 60

	//Получаем обновления от бота
	tb.Updates = tb.Bot.GetUpdatesChan(u)

	tb.MsgStore.Messages = make(map[int]string)
}

func (tb *TeleBot) editMsg(mid, index int, cmd string) {
	text, kb := paginator(cmd, tb.MsgStore.get(mid), index)

	msg := tgb.NewEditMessageText(tb.ChatID, mid, text)
	msg.ReplyMarkup = &kb
	msg.ParseMode = tgb.ModeHTML

	if _, err := tb.Bot.Send(msg); err != nil {
		log.Panic(err)
	}
}

func (tb *TeleBot) sendMsg(env Envelope) {
	text := beautify(env.message, env.subject, env.from)

	msg := tgb.NewMessage(tb.ChatID, text)

	if lensafe(text) > lessText {
		msg.Text = slicer(text, 0, lessText)
		kb := inlineKb(0, 0)
		kb.InlineKeyboard = append(kb.InlineKeyboard, fileKb(env.filenames).InlineKeyboard...)
		msg.ReplyMarkup = kb
	} else if len(env.filenames) > 0 {
		msg.ReplyMarkup = fileKb(env.filenames)
	}

	msg.ParseMode = tgb.ModeHTML

	respMsg, err := tb.Bot.Send(msg)
	if err != nil {
		log.Panic(err)
	}
	tb.MsgStore.add(respMsg.MessageID, text)
}

func (tb *TeleBot) sendFile(id int, filename string) {
	log.Println("Name:", filename)
	doc := tgb.NewDocument(tb.ChatID, tgb.FilePath(filename))
	doc.Caption = strings.Split(filename, "/")[len(strings.Split(filename, "/"))-1]
	doc.ReplyToMessageID = id
	msg, err := tb.Bot.Send(doc)
	log.Println("MesID:", msg.MessageID)

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

			} else if update.CallbackQuery != nil {
				callback := tgb.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
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
