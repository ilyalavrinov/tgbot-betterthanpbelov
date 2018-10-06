package cmd

import "time"
import "github.com/admirallarimda/tgbot-base"

type Reminder struct {
	t       time.Time
	chat    botbase.ChatID
	replyTo int // message ID
}

type ReminderStorage interface {
	AddReminder(Reminder)
	RemoveReminder(Reminder)
	LoadAll() []Reminder
}
