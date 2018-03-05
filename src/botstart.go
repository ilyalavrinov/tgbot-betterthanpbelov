package main

import "./mybot"
import "log"

func main() {
    log.Print("Starting my bot")

    err := mybot.Start("mybot.cfg")
    if err != nil {
        log.Printf("My bot could not be started due to error: %s", err)
    }

    log.Print("My bot has stopped working")
}
