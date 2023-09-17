package main

import (
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

	tb.MsgStore.Messages = make(map[int]string)
}

func (tb *TeleBot) editMsg(id int64, mid, index int, cmd string) {
	text, kb := paginator(cmd, tb.MsgStore.get(mid), index)

	msg := tgb.NewEditMessageText(tb.ChatID, mid, text)
	msg.ReplyMarkup = &kb
	msg.ParseMode = tgb.ModeHTML

	if _, err := tb.Bot.Send(msg); err != nil {
		log.Panic(err)
	}
}

func (tb *TeleBot) sendMsg(id int64, text string) {
	fullText := text
	length := lensafe(text)
	if length > lessText {
		text = slicer(text, 0, lessText)
	}

	msg := tgb.NewMessage(id, text)
	msg.ReplyMarkup = inlineKb(0, 0)
	msg.ParseMode = tgb.ModeHTML

	respMsg, err := tb.Bot.Send(msg)
	if err != nil {
		log.Panic(err)
	}
	tb.MsgStore.add(respMsg.MessageID, fullText)
}

func (tb *TeleBot) RunBot(envelopes chan Envelope) {
	for {
		select {
		case update := <-tb.Updates:
			if update.Message != nil {
				if update.Message.Text == "help" {
					tb.sendMsg(tb.ChatID, tt)
				}
			} else if update.CallbackQuery != nil {
				callback := tgb.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
				if _, err := tb.Bot.Request(callback); err != nil {
					panic(err)
				}
				data := strings.Split(update.CallbackQuery.Data, ".")
				index, _ := strconv.Atoi(data[0])
				tb.editMsg(tb.ChatID, update.CallbackQuery.Message.MessageID, index, data[1])
			}
		case envelope := <-envelopes:
			tb.sendMsg(tb.ChatID, envelope.message)
		}
	}
}
