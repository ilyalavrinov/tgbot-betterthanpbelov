package cmd

import "fmt"
import "log"
import "time"
import "regexp"
import "strings"
import "encoding/json"
import "net/http"
import "io/ioutil"
import "gopkg.in/telegram-bot-api.v4"


// city IDs: bulk.openweathermap.org/sample/city.list.json.gz
const (
    cityNN = 520555
    citySPb = 498817
)

const (
    EmojiSunny = '\u2600'
    EmojiCloudy = '\u2601'
    EmojiRainy = '\u2602' // TODO: find correct code
    EmojiWindy = '\u2603' // TODO: find correct code
)

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

var weatherWords = []string{"^погода*"}

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

    bytes := requestData("weather", cityNN, handler.token)

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
const timeFormat_API = "2006-01-02 15:04:05"
const timeFormat_Out_Date = "Mon, 02 Jan"
const timeFormat_Out_Time = "15:04"

type forecastData struct {
    City struct {
        Name string
    }
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
    text := msg.Message.Text
    if !msgMatches(text, forecastWords) {
        return nil, nil
    }

    targetCity := cityNN

    // TODO: move to a common function for all weather handlers
    reInCity := regexp.MustCompile("в ([\\wА-Яа-я]+)")
    if reInCity.MatchString(text) {
        log.Printf("Message '%s' matches 'in city' regexp %s", text, reInCity)
        matches := reInCity.FindStringSubmatch(text)
        city := matches[1] // ([\wА-Яа-я]+)
        city = strings.ToLower(city)

        reCitySPb := regexp.MustCompile("питер|спб")
        if reCitySPb.MatchString(city) {
            log.Printf("Switching target city to SPb")
            targetCity = citySPb
        }
    }

    bytes := requestData("forecast", targetCity, handler.token)

    result := NewResult()

    forecast_data := forecastData{}
    err := json.Unmarshal(bytes, &forecast_data)
    //err = json.NewDecoder(resp.Body).Decode(&weather_data)
    if err != nil {
        msg := tgbotapi.NewMessage(msg.Message.Chat.ID, "Я не смог распарсить прогноз :(")
        result.Reply = msg
        return &result, err
    }

    now := time.Now()
    target_day := now
    reTomorrow := regexp.MustCompile("завтра")
    reDayAfterTomorrow := regexp.MustCompile("послезавтра")
    if reDayAfterTomorrow.MatchString(text) {  // DayAfterTomorrow should go first as simple Tomorrow is a substring
        log.Printf("Forecast is requested for the day after tomorrow")
        target_day = time.Date(target_day.Year(), target_day.Month(), target_day.Day() + 2,
                               0, 0, 0, 0, time.Local)
    } else if reTomorrow.MatchString(text) {
        log.Printf("Forecast is requested for tomorrow")
        target_day = time.Date(target_day.Year(), target_day.Month(), target_day.Day() + 1,
                               0, 0, 0, 0, time.Local)
    }

    if (target_day == now) && (target_day.Hour() > 18) {
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
        t, err := time.Parse(timeFormat_API, val.DT_txt)
        if err != nil {
          log.Printf("Error while parsing date: %s; error: %s", val.DT_txt, err)
          continue
        }
        t = t.Local()
        if t.Before(forecast_start) || t.After(forecast_end) {
          log.Printf("Skipping date: %s", t)
          continue
        }
        log.Printf("Forecast: %s,t = %.1f, %s", t, val.Main.Temp, val.Weather[0].Description)
        forecasts = append(forecasts, fmt.Sprintf("%s: %.1f\u2103, %s", t.Format(timeFormat_Out_Time), val.Main.Temp, val.Weather[0].Description))
    }

    if len(forecasts) == 0 {
        log.Printf("Something went wrong - no forecast")
        msg := tgbotapi.NewMessage(msg.Message.Chat.ID, "Я не смог сделать прогноз :(")
        result.Reply = msg
        return &result, err
    }

    forecast_msg := fmt.Sprintf("Прогнозирую на %s в %s:\n", target_day.Format(timeFormat_Out_Date), forecast_data.City.Name)
    for _, forecast := range forecasts {
        forecast_msg += forecast
        forecast_msg += "\n"
    }
    reply := tgbotapi.NewMessage(msg.Message.Chat.ID, forecast_msg)
    result.Reply = reply
    return &result, nil
}
