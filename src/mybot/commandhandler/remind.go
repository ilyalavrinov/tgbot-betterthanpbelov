package cmd

import "../reminder"
import "../common"
import "log"
import "regexp"
import "time"
import "strconv"
import "errors"
import "gopkg.in/telegram-bot-api.v4"

type remindHandler struct {
    storage *reminder.Storage
}

func NewRemindHandler(notifChan chan<- common.Notification) *remindHandler {
    handler := &remindHandler{}
    handler.storage = reminder.NewStorage()

    go SendNotifications(handler.storage, notifChan)

    return handler
}

var reminderWords = []string {"напомни"}

func determineReminderTime(msg string) (time.Time, error) {
    reAfter := regexp.MustCompile("через (\\d+) ([\\wа-я]+)")
    // TODO: uncomment during implementation
    //reAt := regexp.MustCompile("в (\\d{1,2}):(\\d{1,2})")
    //reTomorrow := regexp.MustCompile("завтра")
    //reDayAfterTomorrow := regexp.MustCompile("послезавтра")
    // TODO: add days of week parsing

    now := time.Now()
    if reAfter.MatchString(msg) {
        log.Printf("Message '%s' matches 'after' regexp %s", msg, reAfter)
        matches := reAfter.FindStringSubmatch(msg)
        timeQuantity := matches[1] // (\d+)
        timePeriod := matches[2] // ([\wа-я]+)

        q, _ := strconv.Atoi(timeQuantity)
        period := time.Minute
        matchedMinute, _ := regexp.MatchString("минут", timePeriod)
        matchedHour, _ := regexp.MatchString("час", timePeriod)
        matchedDay, _ := regexp.MatchString("дней", timePeriod)
        if matchedMinute {
            period = time.Minute
        } else if matchedHour {
            period = time.Hour
        } else if matchedDay {
            period = 24 * time.Hour
        } else {
            log.Printf("Time period %s doesn't match any known format", timePeriod)
            err := errors.New("Time period doesn't match any known")
            return now, err
        }

        return now.Add(period * time.Duration(q)), nil
    }

    return now, nil
}

func (handler *remindHandler) HandleMsg(msg *tgbotapi.Update, ctx Context) (*Result, error) {
    if !ctx.BotMessage {
        log.Printf("Message '%s' is not designated for bot manipulation, will not check for reminder", msg.Message.Text)
        return nil, nil
    }

    if !msgMatches(msg.Message.Text, reminderWords) {
        return nil, nil
    }

    t, err := determineReminderTime(msg.Message.Text)
    if err != nil {
        log.Printf("Could not determine time from message '%s' with error: %s", msg.Message.Text, err)
        return nil, err
    }

    handler.storage.AddReminder(msg.Message.From.ID, msg.Message.MessageID, msg.Message.Chat.ID, t)

    return nil, nil
}


func SendNotifications(storage *reminder.Storage, notifChan chan<- common.Notification) {
    for true {
        time.Sleep(30 * time.Second)
        storage.MoveToPending(time.Now())
        for _, record := range storage.Pending {
            notification := common.NewNotification(record.UserId,
                                                   "Напоминаю",
                                                   record.MsgId,
                                                   record.ChatId)
            notifChan<- *notification
        }
        storage.ResetPending()
    }
}
