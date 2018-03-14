package main

import "./mybot"
import "log"

const cfg_filename = "mybot.cfg"

func main() {
    log.Print("Starting my bot")

    err := mybot.Start(cfg_filename)
    if err != nil {
        log.Printf("My bot could not be started due to error: %s", err)
    }

    log.Print("My bot has stopped working")
}
