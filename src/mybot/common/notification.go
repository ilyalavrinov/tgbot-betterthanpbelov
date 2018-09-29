package common

type Notification struct {
	UserId        int
	ChatId        int64
	ReplyTo_MsgId int
	Msg           string
}

func NewNotification(userId int, msg string, replyMsgId int, chatId int64) *Notification {
	notification := &Notification{
		UserId:        userId,
		ChatId:        chatId,
		ReplyTo_MsgId: replyMsgId,
		Msg:           msg}
	return notification
}
