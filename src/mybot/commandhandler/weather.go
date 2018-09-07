package cmd

import "fmt"
import "log"
import "time"
import "regexp"
import "strings"
import "encoding/json"
import "net/http"
import "io/ioutil"
import "github.com/admirallarimda/tgbot-base"
import "gopkg.in/telegram-bot-api.v4"
import "github.com/go-redis/redis"


// city IDs: bulk.openweathermap.org/sample/city.list.json.gz
const (
    cityNN = 520555
    citySPb = 498817
)

var reToday *regexp.Regexp = regexp.MustCompile("сегодня")
var reDayAfterTomorrow *regexp.Regexp = regexp.MustCompile("послезавтра")
var reTomorrow *regexp.Regexp = regexp.MustCompile("завтра")


func requestData(reqType string, cityId int64, apiKey string) []byte {
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


func (h *weatherHandler) determineCity(text string) (int64, error) {
    var targetCity int64 = cityNN

    reInCity := regexp.MustCompile("(в|in) ([\\wA-Za-zА-Яа-я]+)")
    if reInCity.MatchString(text) {
        log.Printf("Message '%s' matches 'in city' regexp %s", text, reInCity)
        matches := reInCity.FindStringSubmatch(text)
        city := matches[2]
        city = strings.ToLower(city)

        key := fmt.Sprintf("openweathermap:city:%s", city)
        result := h.redisconn.HGet(key, "id")
        if result.Err() != nil {
            log.Printf("Could HGet for key '%s', error: %s", key, result.Err())
            return 0, result.Err()
        }

        cityId, err := result.Int64()
        if err != nil {
            log.Printf("Could not convert ID for key '%s' into int, error: %s", key, err)
            return 0, err
        }
        targetCity = cityId
        log.Printf("City ID for %s is %d", city, targetCity)
    }

    log.Printf("City ID is %d", targetCity)
    return targetCity, nil
}

func determineDate(text string) *time.Time {
    now := time.Now()
    target_day := now

    if reDayAfterTomorrow.MatchString(text) {  // DayAfterTomorrow should go first as simple Tomorrow is a substring
        log.Printf("Forecast is requested for the day after tomorrow")
        target_day = time.Date(now.Year(), now.Month(), now.Day() + 2,
                               0, 0, 0, 0, time.Local)
    } else if reTomorrow.MatchString(text) {
        log.Printf("Forecast is requested for tomorrow")
        target_day = time.Date(now.Year(), now.Month(), now.Day() + 1,
                               0, 0, 0, 0, time.Local)
    } else if reToday.MatchString(text) {
        log.Printf("Forecast is requested for today")
        target_day = time.Date(now.Year(), now.Month(), now.Day(),
                               0, 0, 0, 0, time.Local)
    }

    var result *time.Time = nil
    if target_day != now { // probably 'is_regex_matched' flag is better
        result = &target_day
    }
    return result
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

func getCurrentWeather(token string, cityId int64) (string, error) {
    bytes := requestData("weather", cityId, token)

    weather_data := weatherData{}
    err := json.Unmarshal(bytes, &weather_data)
    //err = json.NewDecoder(resp.Body).Decode(&weather_data)
    if err != nil {
        return "Я не смог распарсить погоду :(", err
    }

    weather_msg := fmt.Sprintf("Сейчас в %s: %s, %.1f градусов, дует ветер %.0f м/с", weather_data.Name,
                                                                                      weather_data.Weather[0].Description,
                                                                                      weather_data.Main.Temp,
                                                                                      weather_data.Wind.Speed)
    return weather_msg, nil
}

func getForecast(token string, cityId int64, date time.Time) (string, error) {
    log.Printf("Checking for upcoming weather in city %d", cityId)
    bytes := requestData("forecast", cityId, token)

    forecast_data := forecastData{}
    err := json.Unmarshal(bytes, &forecast_data)
    //err = json.NewDecoder(resp.Body).Decode(&weather_data)
    if err != nil {
        return "Я не смог распарсить прогноз :(", err
    }

    now := time.Now()
    if (date == now) && (date.Hour() > 18) {
        return "Иди спи, нечего гулять по ночам", nil
    }

    forecast_start := date
    if forecast_start.Hour() < 6 {
        forecast_start = time.Date(forecast_start.Year(), forecast_start.Month(), forecast_start.Day(),
                                   5, 59, 0, 0, time.Local)
    }
    forecast_end := time.Date(forecast_start.Year(), forecast_start.Month(), forecast_start.Day(),
                              18, 01, 00, 0, time.Local)

    forecasts := make([]string, 0, 5)
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
        return "Я не смог сделать прогноз :(", err
    }

    forecast_msg := fmt.Sprintf("Прогнозирую на %s в %s:\n", date.Format(timeFormat_Out_Date), forecast_data.City.Name)
    for _, forecast := range forecasts {
        forecast_msg += forecast
        forecast_msg += "\n"
    }

    return forecast_msg, nil
}

const timeFormat_API = "2006-01-02 15:04:05"
const timeFormat_Out_Date = "Mon, 02 Jan"
const timeFormat_Out_Time = "15:04"

var weatherWords = []string{"^погода", "^weather"}

type weatherHandler struct {
    botbase.BaseHandler
    token string
    redisconn *redis.Client
}

func NewWeatherHandler(token string, opts redis.Options) botbase.IncomingMessageHandler {
    handler := weatherHandler{}
    handler.token = token
    //handler.redisconn = redis.NewClient(&opts)
    return &handler
}

func (h *weatherHandler) Init(outMsgCh chan<- tgbotapi.MessageConfig, srvCh chan<- botbase.ServiceMsg) botbase.HandlerTrigger {
    h.OutMsgCh = outMsgCh
    return botbase.HandlerTrigger{Re: regexp.MustCompile("^погода")}
}

func (h *weatherHandler) Name() string {
    return "weather"
}

func (h *weatherHandler) HandleOne(msg tgbotapi.Message) {
    text := msg.Text

    date := determineDate(text)
    cityID, err := h.determineCity(text)
    if err != nil {
        log.Printf("Could not determine city from message '%s' due to error: '%s'", text, err)

        reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Не смог распарсить город :("))
        reply.BaseChat.ReplyToMessageID = msg.MessageID
        h.OutMsgCh<- reply
    }

    var replyMsg string

    if date == nil {
        replyMsg, err = getCurrentWeather(h.token, cityID)
    } else {
        replyMsg, err = getForecast(h.token, cityID, *date)
    }

    reply := tgbotapi.NewMessage(msg.Chat.ID, replyMsg)
    reply.BaseChat.ReplyToMessageID = msg.MessageID
    h.OutMsgCh<- reply
}
