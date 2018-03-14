package mybot

import "log"
import "gopkg.in/telegram-bot-api.v4"

// panics internally if something goes wrong
func setupBot(botToken string) (*tgbotapi.BotAPI, *tgbotapi.UpdatesChannel) {
    log.Printf("Setting up a bot with token: %s", botToken)
    bot, err := tgbotapi.NewBotAPI(botToken)
    if err != nil {
        log.Panic(err)
    }

    log.Printf("Authorized on account %s", bot.Self.UserName)

    // TODO: save on disk on shutdown?
    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60

    updates, err := bot.GetUpdatesChan(u)
    if err != nil {
        log.Panic(err)
    }

    return bot, &updates
}

func executeUpdates(updates *tgbotapi.UpdatesChannel) {
    for update := range *updates {
        if update.Message == nil {
            log.Print("Message: empty. Skipping");
            continue
        }

        log.Printf("Message from: %s; Text: %s", update.Message.From.UserName, update.Message.Text)
        log.Printf("Update: %+v", update)
        log.Printf("Message: %+v", update.Message)
        log.Printf("Message.Chat: %+v", update.Message.Chat)
        log.Printf("Message.NewChatMembers: %+v", update.Message.NewChatMembers)
        // this bot does nothing now. Maybe it will be updated later
    }
}

func Start(cfg_filename string) error {
    log.Print("Starting my bot")

    cfg, err := NewConfig(cfg_filename)
    if err != nil {
        log.Printf("My bot cannot be sarted due to error: %s", err)
        return err
    }

    bot, updates := setupBot(cfg.TGBot.Token);
    go askPBelovForDate(bot)
    executeUpdates(updates)

    log.Print("Stopping my bot")
    return nil
}
