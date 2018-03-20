package mybot

import "log"
import "regexp"
import "strings"
import "gopkg.in/telegram-bot-api.v4"

var kittiesWords = []string{"^кот$", "^котэ$", "^котик*", "^котятк*"}
var weatherWords = []string{"^погода$", "^холодно*", "^жарко*"}

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

func msgMatches(text string, patterns []string) bool {
    compiledRegExp := make([]*regexp.Regexp, 0, len(patterns))
    for _, pattern := range patterns {
        re, err := regexp.Compile(pattern)
        if err != nil {
            log.Printf("Pattern %s cannot be compiled into a regexp. Error: %s", pattern, err)
            continue
        }
        compiledRegExp = append(compiledRegExp, re)
    }

    msgWords := strings.Split(text, " ")
    for _, word := range msgWords {
        for _, re := range compiledRegExp {
            if re.MatchString(word) {
                log.Printf("Word %s matched regexp %s", word, re)
                return true
            }
        }
    }
    log.Printf("None of the words in text: %s; matched patterns %s", text, patterns)
    return false
}

func executeUpdates(updates *tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI, cfg Config) {
    for update := range *updates {
        if update.Message == nil {
            log.Print("Message: empty. Skipping");
            continue
        }

        dumpMessage(update)

        if msgMatches(update.Message.Text, kittiesWords) {
            log.Printf("Message from %s with text %s contains one of kitties words", update.Message.From.UserName, update.Message.Text)
            newMsg, err := sendKitties(update)
            if err != nil {
                log.Printf("Cannot create a message with kitties due to error: %s", err)
                continue
            }
            _, err = bot.Send(newMsg)
            if err != nil {
                log.Printf("Cannot reply with a kitty pic due to error: %s", err)
                continue
            }
            log.Print("Message with kitties has been sent!")
        }
        // TODO: duplication. Think how to use some OOP here
        if msgMatches(update.Message.Text,weatherWords) {
            log.Printf("Message from %s with text %s contains one of weather words", update.Message.From.UserName, update.Message.Text)
            newMsg, err := sendWeather(update, cfg)
            if err != nil {
                log.Printf("Cannot create a message with weather due to error: %s", err)
                continue
            }
            _, err = bot.Send(newMsg)
            if err != nil {
                log.Printf("Cannot reply with a weather due to error: %s", err)
                continue
            }
            log.Print("Message with weather has been sent!")
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

    bot, updates := setupBot(cfg.TGBot.Token);
    //go askPBelovForDate(bot)
    executeUpdates(updates, bot, cfg)

    log.Print("Stopping my bot")
    return nil
}
