package mybot

import "log"
import "github.com/admirallarimda/tgbot-base"
import "github.com/go-redis/redis"

import "./commandhandler"

func Start(cfg_filename string) error {
    log.Print("Starting my bot")

    fullcfg, err := NewConfig(cfg_filename)
    if err != nil {
        log.Printf("My bot cannot be sarted due to error: %s", err)
        return err
    }

    log.Printf("Starting bot with full config: %+v", fullcfg)

    tgcfg := botbase.Config{TGBot: fullcfg.TGBot,
                            Proxy_SOCKS5: fullcfg.Proxy_SOCKS5}
    bot := botbase.NewBot(tgcfg)
    bot.AddHandler(botbase.NewIncomingMessageDealer(cmd.NewWeatherHandler(fullcfg.Weather.Token, redis.Options{})))
    /*     handlers = append(handlers, cmd.NewKittiesHandler(),
                                    cmd.NewWeatherHandler(cfg.Weather.Token, opts),
                                    cmd.NewDeathHandler(),
                                    cmd.NewRemindHandler(notificationChannel))
    */
    bot.Start()

    log.Print("Stopping my bot")
    return nil
}
