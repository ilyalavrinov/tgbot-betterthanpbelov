package mybot

import "fmt"
import "log"
import "encoding/json"
import "net/http"
import "io/ioutil"
import "gopkg.in/telegram-bot-api.v4"


// city IDs: bulk.openweathermap.org/sample/city.list.json.gz
var cityID = map[string]int {"NN": 520555, "SPb": 98817}

type weatherData struct {
    Main struct {
        Temp float64
    }
    Name string
    Weather []struct {
        Description string
    }
    Wind struct {
        Speed float32
    }
}

func sendWeather(update tgbotapi.Update, cfg Config) (tgbotapi.MessageConfig, error) {
    weather_url := fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?id=%d&APPID=%s&lang=ru&units=metric", cityID["NN"],
                                                                                                                     cfg.Weather.Token)
    log.Printf("Sending weather request using url: %s", weather_url)

    resp, err := http.Get(weather_url)
    if err != nil {
        msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Я не смог запросить погоду :(")
        return msg, err
    }
    defer resp.Body.Close()
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    log.Printf("Weather response: %s", string(bodyBytes))

    weather_data := weatherData{}
    err = json.Unmarshal(bodyBytes, &weather_data)
    //err = json.NewDecoder(resp.Body).Decode(&weather_data)
    if err != nil {
        msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Я не смог распарсить погоду :(")
        return msg, err
    }

    weather_msg := fmt.Sprintf("Сейчас в %s: %s, %.1f градусов, дует ветер %.0f м/с", weather_data.Name,
                                                                                      weather_data.Weather[0].Description,
                                                                                      weather_data.Main.Temp,
                                                                                      weather_data.Wind.Speed)
    msg := tgbotapi.NewMessage(update.Message.Chat.ID, weather_msg)
    return msg, nil
}
