package cmd

import "fmt"
import "log"
import "time"
import "encoding/json"
import "net/http"
import "io/ioutil"
import "gopkg.in/telegram-bot-api.v4"


// city IDs: bulk.openweathermap.org/sample/city.list.json.gz
var cityID = map[string]int {"NN": 520555, "SPb": 98817}

func requestData(reqType string, cityId int, apiKey string) []byte {
    weather_url := fmt.Sprintf("http://api.openweathermap.org/data/2.5/%s?id=%d&APPID=%s&lang=ru&units=metric", reqType,
                                                                                                                cityId,
                                                                                                                apiKey)
    log.Printf("Sending weather request using url: %s", weather_url)

    resp, err := http.Get(weather_url)
    if err != nil {
        log.Printf("Could not get data from '%s' due to error: %s", weather_url, err)
        return []byte{}
    }
    defer resp.Body.Close()
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Printf("Could not read response body from '%s' due to error: %s", weather_url, err)
        return []byte{}
    }

    log.Printf("Weather response: %s", string(bodyBytes))

    return bodyBytes
}


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

var weatherWords = []string{"^погода*", "^холодно*", "^жарко*"}

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

    bytes := requestData("weather", cityID["NN"], handler.token)

    result := NewResult()

    weather_data := weatherData{}
    err := json.Unmarshal(bytes, &weather_data)
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


var forecastWords = []string{"^прогноз*"}
const timeFormat = "2006-01-02 15:04:05"

type forecastData struct {
    List []struct {
        DT_txt string
        Weather []struct {
            Description string
        }
        Main struct {
            Temp float32
        }
    }
}

type forecastHandler struct {
    token string
}

func NewForecastHandler(token string) *forecastHandler {
    handler := &forecastHandler{}
    handler.token = token
    return handler
}

func (handler *forecastHandler) HandleMsg(msg *tgbotapi.Update, ctx Context) (*Result, error) {
    if !msgMatches(msg.Message.Text, forecastWords) {
        return nil, nil
    }

    bytes := requestData("forecast", cityID["NN"], handler.token)

    result := NewResult()

    forecast_data := forecastData{}
    err := json.Unmarshal(bytes, &forecast_data)
    //err = json.NewDecoder(resp.Body).Decode(&weather_data)
    if err != nil {
        msg := tgbotapi.NewMessage(msg.Message.Chat.ID, "Я не смог распарсить прогноз :(")
        result.Reply = msg
        return &result, err
    }

    target_day := time.Now()
    if target_day.Hour() > 18 {
        log.Printf("Today (%s) is too late for the forecast, switching to tomorrow", target_day)
        target_day = time.Date(target_day.Year(), target_day.Month(), target_day.Day() + 1,
                               0, 0, 0, 0, time.Local)
    }
    forecast_start := target_day
    if forecast_start.Hour() < 9 {
        forecast_start = time.Date(forecast_start.Year(), forecast_start.Month(), forecast_start.Day(),
                                   8, 59, 0, 0, time.Local)
    }
    forecast_end := time.Date(forecast_start.Year(), forecast_start.Month(), forecast_start.Day(),
                              18, 01, 00, 0, time.Local)

    forecasts := make([]string, 0, 4)  // 4 since it is usually no more than 4 forecasts
    for _, val := range forecast_data.List {
        t, err := time.Parse(timeFormat, val.DT_txt)
        if err != nil {
          log.Printf("Error while parsing date: %s; error: %s", val.DT_txt, err)
          continue
        }
        t = t.Local()
        if t.Before(forecast_start) || t.After(forecast_end) {
          log.Printf("Skipping date: %s", t)
          continue
        }
        log.Printf("Forecast: %s, %.1f, %s", t, val.Main.Temp, val.Weather[0].Description)
        forecasts = append(forecasts, fmt.Sprintf("%s: температура %.1f, %s", t.Format(timeFormat), val.Main.Temp, val.Weather[0].Description))
    }
    forecast_msg := "Прогнозирую:\n"
    for _, forecast := range forecasts {
        forecast_msg += forecast
        forecast_msg += "\n"
    }
    reply := tgbotapi.NewMessage(msg.Message.Chat.ID, forecast_msg)
    result.Reply = reply
    return &result, nil
}
