package reminder

import "log"
import "time"

type Storage struct {
    Future []Record
    Pending []Record  // need to send them
}

func NewStorage() *Storage {
    storage := &Storage{}

    storage.Future = make([]Record, 0)
    storage.Pending = make([]Record, 0)

    return storage
}

func (storage *Storage) MoveToPending(t time.Time) {
    log.Printf("Starting moving everything to pending")
    newFuture := make([]Record, 0, len(storage.Pending))
    for _, record := range storage.Future {
        if record.Time.Before(t) {
            log.Printf("Reminder for user %d needs to be sent as its time %s is before checked time %s",
                            record.UserId, record.Time, t)
            storage.Pending = append(storage.Pending, record)
        } else {
            newFuture = append(newFuture, record)
        }
    }
    storage.Future = newFuture
    log.Printf("After moving there are %d items in Future, %d items need to be sent", len(storage.Future), len(storage.Pending))
}

func (storage *Storage) AddReminder(userId, msgId int, chatId int64, t time.Time) {
    record := NewRecord(userId, msgId, chatId, t)
    storage.Future = append(storage.Future, *record)
    log.Printf("A new record has been added to reminder storage for user %d to be executed at %s", userId, t)
}

func (storage *Storage) ResetPending() {
    storage.Pending = make([]Record, 0)
}
