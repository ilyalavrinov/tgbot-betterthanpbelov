package reminder

import "time"

type Record struct {
	UserId int
	MsgId  int // msg id which initiated this reminder
	ChatId int64
	Time   time.Time
}

func NewRecord(userId, msgId int, chatId int64, reminderTime time.Time) *Record {
	result := &Record{
		UserId: userId,
		MsgId:  msgId,
		ChatId: chatId,
		Time:   reminderTime}

	return result
}
