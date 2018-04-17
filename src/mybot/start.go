package mybot

import "log"
import "regexp"
import "gopkg.in/telegram-bot-api.v4"
import "golang.org/x/net/proxy"
import "net/http"

import "./commandhandler"
import "./common"

// panics internally if something goes wrong
func setupBot(cfg Config) (*tgbotapi.BotAPI, *tgbotapi.UpdatesChannel) {
    botToken := cfg.TGBot.Token
    log.Printf("Setting up a bot with token: %s", botToken)

    var bot *tgbotapi.BotAPI = nil
    server := cfg.Proxy_SOCKS5.Server
    user := cfg.Proxy_SOCKS5.User
    pass := cfg.Proxy_SOCKS5.Pass
    if server != "" {
        log.Printf("Proxy is set, connecting to '%s' with credentials '%s':'%s'", server, user, pass)
        auth := proxy.Auth { User: user,
                             Password: pass}
        dialer, err := proxy.SOCKS5("tcp", server, &auth, proxy.Direct)
        if err != nil {
            log.Panicf("Could get proxy dialer, error: %s", err)
        }
        httpTransport := &http.Transport{}
        httpTransport.Dial = dialer.Dial
        httpClient := &http.Client{Transport: httpTransport}
        bot, err = tgbotapi.NewBotAPIWithClient(botToken, httpClient)
        if err != nil {
            log.Panicf("Could not connect via proxy, error: %s", err)
        }
    } else {
        log.Printf("No proxy is set, going without any proxy")
        var err error
        bot, err = tgbotapi.NewBotAPI(botToken)
        if err != nil {
            log.Panicf("Could not connect directly, error: %s", err)
        }
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

func modifyContext(context cmd.Context, update tgbotapi.Update) cmd.Context {
    ctx := context

    matched, err := regexp.MatchString("бот", update.Message.Text)
    if err != nil {
        log.Printf("Error '%s' happened during parsing message '%s' for bot-explicit keywords", err, update.Message.Text)
        return ctx
    }

    if matched {
        log.Printf("Bot explicit command is discovered in message '%s'", update.Message.Text)
        ctx.BotMessage = true
    }

    return ctx
}

func executeUpdates(updates *tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI, cfg Config) {
    notificationChannel := make(chan common.Notification)

    // register all handlers
    handlers := make([]cmd.CommandHandler, 0, 10)
    handlers = append(handlers, cmd.NewKittiesHandler(),
                                cmd.NewWeatherHandler(cfg.Weather.Token),
                                cmd.NewForecastHandler(cfg.Weather.Token),
                                cmd.NewDeathHandler(),
                                cmd.NewRemindHandler(notificationChannel))

    context := cmd.NewContext([]string{cfg.Owners.ID[0]})

    isRunning := true

    for isRunning {
        select {
            case update := <-*updates:
                log.Printf("Received an update from tgbotapi")
                // TODO: move to a function
                if update.Message == nil {
                    log.Print("Message: empty. Skipping");
                    continue
                }
                dumpMessage(update)
                ctx := modifyContext(context, update)
                for _, handler := range(handlers) {
                    result, err := handler.HandleMsg(&update, ctx)
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
                            log.Printf("Cannot reply with a message due to error: %s", err)
                            continue
                        }
                        log.Print("Reply has been sent!")
                    }
                    if result.BotToStop == true {
                        log.Print("Bot stop has been requested")
                        isRunning = false
                    }
                }

            case notification := <-notificationChannel:
                log.Printf("A new notification from internals has been received")
                message := tgbotapi.NewMessage(notification.ChatId, notification.Msg)
                message.BaseChat.ReplyToMessageID = notification.ReplyTo_MsgId
                _, err := bot.Send(message)
                if err != nil {
                    log.Printf("Cannot reply with a notification due to error: %s", err)
                    continue
                }
        }
    }

    log.Print("Main cycle has been aborted")
}

func Start(cfg_filename string) error {
    log.Print("Starting my bot")

    cfg, err := NewConfig(cfg_filename)
    if err != nil {
        log.Printf("My bot cannot be sarted due to error: %s", err)
        return err
    }

    log.Printf("Starting bot with config: %+v", cfg)

    bot, updates := setupBot(cfg);
    //go askPBelovForDate(bot)
    executeUpdates(updates, bot, cfg)

    log.Print("Stopping my bot")
    return nil
}
