package cmd

import "log"
import "strings"
import "gopkg.in/telegram-bot-api.v4"
import "github.com/admirallarimda/tgbot-base"

type propertyHandler struct {
	storage botbase.PropertyStorage
}

var _ botbase.IncomingMessageHandler = &propertyHandler{}

func NewPropertyHandler(storage botbase.PropertyStorage) *propertyHandler {
	return &propertyHandler{storage: storage}
}

func (h *propertyHandler) HandleOne(msg tgbotapi.Message) {
	args := msg.CommandArguments()
	user := botbase.UserID(msg.From.ID)
	chat := botbase.ChatID(msg.Chat.ID)

	splits := strings.SplitN(args, " ", 2)
	if len(splits) != 2 {
		log.Printf("Could not split property arguments '%s' into name + value", args)
		return
	}
	propname := splits[0]
	propvalue := splits[1]

	if msg.Command() == "propsetchat" {
		user = 0
	}

	err := h.storage.SetPropertyForUserInChat(propname, user, chat, propvalue)
	if err != nil {
		log.Printf("Could not correctly set property '%s' for user %d chat %d due to error: %s", propname, user, chat, err)
	}
}

func (h *propertyHandler) Init(outMsgCh chan<- tgbotapi.MessageConfig, srvCh chan<- botbase.ServiceMsg) botbase.HandlerTrigger {
	return botbase.NewHandlerTrigger(nil, []string{"propset", "propsetchat"})
}

func (h *propertyHandler) Name() string {
	return "Property"
}
