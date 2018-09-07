package mybot

import "github.com/admirallarimda/tgbot-base"
import "gopkg.in/gcfg.v1"
import "log"

type Config struct {
    botbase.Config
    Weather struct {
        Token string
    }

    Owners struct {
        ID []string
    }
}

func NewConfig(filename string) (Config, error) {
    log.Printf("Reading configuration from: %s", filename)

    var cfg Config

    err := gcfg.ReadFileInto(&cfg, filename)
    if err != nil {
        log.Printf("Could not correctly parse configuration file: %s; error: %s", filename, err)
        return cfg, err
    }

    log.Printf("Configuration has been successfully read from %s: %+v", filename, cfg)
    return cfg, nil
}
