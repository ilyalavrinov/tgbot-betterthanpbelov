package mybot

import "log"
import "gopkg.in/telegram-bot-api.v4"

import "./commandhandler"

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

func dumpMessage(update tgbotapi.Update) {
    log.Printf("Message from: %s; Text: %s", update.Message.From.UserName, update.Message.Text)
    log.Printf("Update: %+v", update)
    log.Printf("Message: %+v", update.Message)
    log.Printf("Message.Chat: %+v", update.Message.Chat)
    log.Printf("Message.NewChatMembers: %+v", update.Message.NewChatMembers)
}

func executeUpdates(updates *tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI, cfg Config) {
    // register all handlers
    handlers := make([]cmd.CommandHandler, 0, 10)
    handlers = append(handlers, cmd.NewKittiesHandler(),
                                cmd.NewWeatherHandler(cfg.Weather.Token),
                                cmd.NewDeathHandler())

    context := cmd.Context{}
    context.Owners = append(context.Owners, cfg.Owners.ID[0])

    isRunning := true

    for update := range *updates {
        if update.Message == nil {
            log.Print("Message: empty. Skipping");
            continue
        }

        dumpMessage(update)
        for _, handler := range(handlers) {
            result, err := handler.HandleMsg(&update, context)
            if err != nil {
                log.Printf("Handler could not handle message with text '%s' due to erros: %s", update.Message.Text, err)
                // going further - maybe we have something to reply to a user
            }

            if result == nil {
                // do nothing - this handler didn't handle this message
                continue
            }

            log.Printf("Message with text '%s' has been handled by some handler", update.Message.Text)
            if result.Reply != nil {
                _, err = bot.Send(result.Reply)
                if err != nil {
                    log.Printf("Cannot reply with a weather due to error: %s", err)
                    continue
                }
                log.Print("Reply has been sent!")
            }
            if result.BotToStop == true {
                log.Print("Bot stop has been requested")
                isRunning = false
            }
        }

        if isRunning == false {
            log.Print("Aborting main cycle")
            break
        }
    }
}

func Start(cfg_filename string) error {
    log.Print("Starting my bot")

    cfg, err := NewConfig(cfg_filename)
    if err != nil {
        log.Printf("My bot cannot be sarted due to error: %s", err)
        return err
    }

    log.Printf("Starting bot with config: %+v", cfg)

    bot, updates := setupBot(cfg.TGBot.Token);
    //go askPBelovForDate(bot)
    executeUpdates(updates, bot, cfg)

    log.Print("Stopping my bot")
    return nil
}
