package cmd

import "github.com/admirallarimda/tgbot-base"
import "log"
import "regexp"
import "time"
import "strconv"
import "errors"
import "fmt"
import "gopkg.in/telegram-bot-api.v4"

const timeFormat_Out_Confirm = "2006-01-02 15:04:05"

type remindCronJob struct {
	outMsgCh chan<- tgbotapi.MessageConfig

	chatID     int64
	text       string
	replyMsgID int
}

func (j *remindCronJob) Do(scheduled time.Time, cron botbase.Cron) {
	msg := tgbotapi.NewMessage(j.chatID, j.text)
	msg.BaseChat.ReplyToMessageID = j.replyMsgID

	j.outMsgCh <- msg
}

type remindHandler struct {
	botbase.BaseHandler
	cron botbase.Cron
}

func NewRemindHandler(cron botbase.Cron) *remindHandler {
	handler := &remindHandler{cron: cron}

	return handler
}

func determineReminderTime(msg string) (time.Time, error) {
	reAfter := regexp.MustCompile("через (\\d*) *([\\wа-я]+)")
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
		timePeriod := matches[2]   // ([\wа-я]+)

		log.Printf("Reminder command matched: quantity '%s' period '%s'", timeQuantity, timePeriod)

		var q int = 1
		if len(timeQuantity) > 0 {
			q, _ = strconv.Atoi(timeQuantity)
		}
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

func (h *remindHandler) HandleOne(msg tgbotapi.Message) {
	t, err := determineReminderTime(msg.Text)
	if err != nil {
		log.Printf("Could not determine time from message '%s' with error: %s", msg.Text, err)
	}

	job := remindCronJob{
		outMsgCh:   h.OutMsgCh,
		chatID:     msg.Chat.ID,
		text:       "Напоминаю",
		replyMsgID: msg.MessageID}
	h.cron.AddJob(t, &job)
	replyText := fmt.Sprintf("Принято, напомню около %s", t.Format(timeFormat_Out_Confirm))
	replyMsg := tgbotapi.NewMessage(msg.Chat.ID, replyText)
	replyMsg.BaseChat.ReplyToMessageID = msg.MessageID
	h.OutMsgCh <- replyMsg

}

func (h *remindHandler) Init(outMsgCh chan<- tgbotapi.MessageConfig, srvCh chan<- botbase.ServiceMsg) botbase.HandlerTrigger {
	h.OutMsgCh = outMsgCh
	return botbase.NewHandlerTrigger(nil, []string{"remind", "todo"})
}

func (h *remindHandler) Name() string {
	return "reminder"
}
