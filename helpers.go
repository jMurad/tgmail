package main

import (
	"fmt"
	"strconv"
	"strings"

	tgb "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const sizeMsgStore = 100
const sizeText = 3000
const lessText = 500

func slicer(s string, begin, end int) string {
	return string([]rune(s)[begin:end])
}

func lensafe(s string) int {
	return len([]rune(s))
}

func inlineKb(i, ind int) (kb tgb.InlineKeyboardMarkup) {
	if i == 0 {
		kb = tgb.NewInlineKeyboardMarkup(
			tgb.NewInlineKeyboardRow(
				tgb.NewInlineKeyboardButtonData("Раскрыть", strconv.Itoa(ind)+":pag:more"),
			),
		)
	} else if i == 1 {
		kb = tgb.NewInlineKeyboardMarkup(
			tgb.NewInlineKeyboardRow(
				tgb.NewInlineKeyboardButtonData("Далее >", strconv.Itoa(ind)+":pag:next"),
			),
			tgb.NewInlineKeyboardRow(
				tgb.NewInlineKeyboardButtonData("Скрыть", strconv.Itoa(ind)+":pag:less"),
			),
		)
	} else if i == 2 {
		kb = tgb.NewInlineKeyboardMarkup(
			tgb.NewInlineKeyboardRow(

				tgb.NewInlineKeyboardButtonData("< Назад", strconv.Itoa(ind)+":pag:prev"),
				tgb.NewInlineKeyboardButtonData("Далее >", strconv.Itoa(ind)+":pag:next"),
			),
			tgb.NewInlineKeyboardRow(
				tgb.NewInlineKeyboardButtonData("Скрыть", strconv.Itoa(ind)+":pag:less"),
			),
		)
	} else if i == 3 {
		kb = tgb.NewInlineKeyboardMarkup(
			tgb.NewInlineKeyboardRow(
				tgb.NewInlineKeyboardButtonData("< Назад", strconv.Itoa(ind)+":pag:prev"),
			),
			tgb.NewInlineKeyboardRow(
				tgb.NewInlineKeyboardButtonData("Скрыть", strconv.Itoa(ind)+":pag:less"),
			),
		)
	} else if i == 4 {
		kb = tgb.NewInlineKeyboardMarkup(
			tgb.NewInlineKeyboardRow(
				tgb.NewInlineKeyboardButtonData("Скрыть", strconv.Itoa(ind)+":pag:less"),
			),
		)
	}
	return
}

func fileKb(filenames []string) (kb tgb.InlineKeyboardMarkup) {
	var rows [][]tgb.InlineKeyboardButton
	for _, path := range filenames {
		splitNames := strings.Split(path, "/")
		last := len(splitNames) - 1
		name := splitNames[last]
		keyBtn := tgb.NewInlineKeyboardButtonData(name, "file:"+path)
		rows = append(rows, tgb.NewInlineKeyboardRow(keyBtn))
	}

	kb = tgb.NewInlineKeyboardMarkup(rows...)
	return
}

func paginator(cmd, text string, indx int) (res string, kb tgb.InlineKeyboardMarkup) {
	count := lensafe(text) / sizeText
	if lensafe(text)%sizeText > 0 {
		count++
	}

	if cmd == "next" || cmd == "more" {
		if count == 1 {
			res = slicer(text, indx*sizeText, lensafe(text))
			kb = inlineKb(4, indx+1)
		} else if indx == count-1 {
			res = slicer(text, indx*sizeText, lensafe(text))
			kb = inlineKb(3, indx+1)
		} else if indx == 0 {
			indx++
			res = slicer(text, 0, indx*sizeText)
			kb = inlineKb(1, indx)
		} else {
			indx++
			res = slicer(text, (indx-1)*sizeText, indx*sizeText)
			kb = inlineKb(2, indx)
		}
	} else if cmd == "prev" {
		if indx == 2 {
			indx--
			res = slicer(text, (indx-1)*sizeText, indx*sizeText)
			kb = inlineKb(1, indx)
		} else {
			indx--
			res = slicer(text, (indx-1)*sizeText, indx*sizeText)
			kb = inlineKb(2, indx)
		}
	} else if cmd == "less" {
		length := lensafe(text)
		if length > lessText {
			text = slicer(text, 0, lessText)
		}
		res = text
		indx = 0
		kb = inlineKb(0, indx)
	}
	return
}

func beautify(body, subject string, from []string) (content string) {
	for _, fr := range from {
		name, email := strings.Split(fr, " | ")[0], strings.Split(fr, " | ")[1]
		content += fmt.Sprintf("✉️ <strong>%s</strong> | <i>%s</i>\n", name, email)
	}
	content += subject + "\n\n"
	content += body
	return
}
