package cmd

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

var weatherWords = []string{"^погода$", "^холодно*", "^жарко*"}

type weatherHandler struct {
    token string
}

func NewWeatherHandler(token string) CommandHandler {
    handler := weatherHandler{}
    handler.token = token
    return &handler
}

func (handler *weatherHandler) HandleMsg (msg *tgbotapi.Update, ctx Context) (*Result, error) {
    if !msgMatches(msg.Message.Text, weatherWords) {
        return nil, nil
    }

    weather_url := fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?id=%d&APPID=%s&lang=ru&units=metric", cityID["NN"],
                                                                                                                     handler.token)
    log.Printf("Sending weather request using url: %s", weather_url)

    result := NewResult()

    resp, err := http.Get(weather_url)
    if err != nil {
        msg := tgbotapi.NewMessage(msg.Message.Chat.ID, "Я не смог запросить погоду :(")
        result.Reply = msg
        return &result, err
    }
    defer resp.Body.Close()
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    log.Printf("Weather response: %s", string(bodyBytes))

    weather_data := weatherData{}
    err = json.Unmarshal(bodyBytes, &weather_data)
    //err = json.NewDecoder(resp.Body).Decode(&weather_data)
    if err != nil {
        msg := tgbotapi.NewMessage(msg.Message.Chat.ID, "Я не смог распарсить погоду :(")
        result.Reply = msg
        return &result, err
    }

    weather_msg := fmt.Sprintf("Сейчас в %s: %s, %.1f градусов, дует ветер %.0f м/с", weather_data.Name,
                                                                                      weather_data.Weather[0].Description,
                                                                                      weather_data.Main.Temp,
                                                                                      weather_data.Wind.Speed)
    reply := tgbotapi.NewMessage(msg.Message.Chat.ID, weather_msg)
    result.Reply = reply
    return &result, nil
}
