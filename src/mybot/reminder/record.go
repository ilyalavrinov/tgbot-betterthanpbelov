package reminder

import "time"

type Record struct {
    UserId int
    MsgId int  // msg id which initiated this reminder
    Time time.Time
}

func NewRecord(userId, msgId int, reminderTime time.Time) *Record {
    result := &Record {
        UserId: userId,
        MsgId: msgId,
        Time: reminderTime }

    return result
}
