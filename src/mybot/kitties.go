package mybot

import "log"
import "gopkg.in/telegram-bot-api.v4"

func sendKitties(update tgbotapi.Update) (tgbotapi.PhotoConfig, error) {
    log.Print("Sending kitties")
    picMsg := tgbotapi.NewPhotoUpload(update.Message.Chat.ID, "/home/ilyalavrinov/Pictures/cat.jpeg")
    return picMsg, nil
}
