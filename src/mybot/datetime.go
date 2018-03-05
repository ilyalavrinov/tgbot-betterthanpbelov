package mybot

import "time"
import "log"
import "gopkg.in/telegram-bot-api.v4"

func askPBelovForDate(bot *tgbotapi.BotAPI) {
    // every day at 6:00 send a request to pbelov bot

    now := time.Now()
    t1 := now + time.ParseDuration("24h")
    next_date_req := time.Date(t1.Year(), t1.Month(), t1.Day(), 06, 00, 00, 00, time.Local)
    diff := next_date_req.Sub(now)

    log.Print("Next duration will be asked in: ", diff.String())
    time.Sleep(diff)
    log.Print("Wake up! Time to ask pbelov for the time!")
}
