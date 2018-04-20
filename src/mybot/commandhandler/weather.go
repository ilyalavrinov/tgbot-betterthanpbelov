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


func determineCity(text string) int {
    var targetCity int = cityNN

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

    return targetCity
}

func determineDate(text string) *time.Time {
    now := time.Now()
    target_day := now

    reToday := regexp.MustCompile("сегодня")
    reDayAfterTomorrow := regexp.MustCompile("послезавтра")
    reTomorrow := regexp.MustCompile("завтра")
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

func getCurrentWeather(token string, cityId int) (string, error) {
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

func getForecast(token string, cityId int, date time.Time) (string, error) {
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

var weatherWords = []string{"^погода"}

type weatherHandler struct {
    token string
}

func NewWeatherHandler(token string) CommandHandler {
    handler := weatherHandler{}
    handler.token = token
    return &handler
}

func (handler *weatherHandler) HandleMsg (msg *tgbotapi.Update, ctx Context) (*Result, error) {
    text := msg.Message.Text
    if !msgMatches(text, weatherWords) {
        return nil, nil
    }

    cityId := determineCity(text)
    date := determineDate(text)

    var replyMsg string
    var err error

    if date == nil {
        replyMsg, err = getCurrentWeather(handler.token, cityId)
    } else {
        replyMsg, err = getForecast(handler.token, cityId, *date)
    }

    result := NewResult()
    result.Reply = tgbotapi.NewMessage(msg.Message.Chat.ID, replyMsg)
    return &result, err
}
