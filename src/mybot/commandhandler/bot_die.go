package cmd

import "gopkg.in/telegram-bot-api.v4"

var dieWords = []string{"^умри$", "^die$"}

type botDeathHandler struct {}

func NewDeathHandler() *botDeathHandler {
    return &botDeathHandler{}
}

func (handler *botDeathHandler) HandleMsg(msg *tgbotapi.Update, ctx Context) (*Result, error) {
    if !msgMatches(msg.Message.Text, dieWords) {
        return nil, nil
    }

    result := NewResult()
    result.BotToStop = true
    return &result, nil
}
