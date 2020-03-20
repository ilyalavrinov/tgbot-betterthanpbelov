package cmd

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/admirallarimda/tgbotbase"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

type covid19Handler struct {
	tgbotbase.BaseHandler
	props tgbotbase.PropertyStorage
	cron  tgbotbase.Cron

	updates chan covidData
	toSend  chan tgbotbase.ChatID
}

var _ tgbotbase.BackgroundMessageHandler = &covid19Handler{}

func NewCovid19Handler(cron tgbotbase.Cron,
	props tgbotbase.PropertyStorage) tgbotbase.BackgroundMessageHandler {
	h := &covid19Handler{
		props: props,
		cron:  cron,

		updates: make(chan covidData, 0),
		toSend:  make(chan tgbotbase.ChatID, 0),
	}
	return h
}

func (h *covid19Handler) Init(outMsgCh chan<- tgbotapi.Chattable, srvCh chan<- tgbotbase.ServiceMsg) {
	h.OutMsgCh = outMsgCh
}

func (h *covid19Handler) Run() {
	// TODO: same as for kitties. Write common func
	now := time.Now()

	countriesOfInterest := []string{"Italy", "Russia", "China"}

	go func() {
		data := covidData{}
		for {
			select {
			case data = <-h.updates:
				// ok, do nothing
			case chatID := <-h.toSend:
				text := fmt.Sprintf("Новости коронавируса!")
				worldLatest := data.countryLatest["World"]
				text = fmt.Sprintf("%s\nВ мире:\nВсего заболевших: %d (новых: +%d)\nВсего умерших: %d (новых: +%d)",
					text, worldLatest.totalCases, worldLatest.newCases, worldLatest.totalDeaths, worldLatest.newDeaths)
				for _, country := range countriesOfInterest {
					if cases, found := data.countryLatest[country]; found {
						text = fmt.Sprintf("%s\n\nВ %s (данные на %s):\nЗаболевших: %d (новых за день: +%d)\nУмерших: %d (новых за день: +%d)",
							text, country, cases.date.Format("2006-01-02"), cases.totalCases, cases.newCases, cases.totalDeaths, cases.newDeaths)
					}
				}
				russiaData := data.countryLatest["Russia"]
				italyRaw := data.countryRaw["Italy"]
				for i := len(italyRaw) - 1; i >= 0; i-- {
					raw := italyRaw[i]
					if raw.totalCases < russiaData.totalCases {
						text = fmt.Sprintf("%s\n\nВ Италии похожее количестве заболевших (%d; +%d по сравнению с предыдущим днём) было %s (%.0f дней назад)",
							text, raw.totalCases, raw.newCases, raw.date.Format("2006-01-02"), russiaData.date.Sub(raw.date).Hours()/24)
						break
					}
				}
				msg := tgbotapi.NewMessage(int64(chatID), text)
				h.OutMsgCh <- msg
			}
		}
	}()

	h.cron.AddJob(time.Now(), &covidUpdateJob{updates: h.updates})

	props, _ := h.props.GetEveryHavingProperty("covid19Time")
	for _, prop := range props {
		if (prop.User != 0) && (tgbotbase.ChatID(prop.User) != prop.Chat) {
			log.Printf("COVID-19: Skipping special setting for user %d in chat %d", prop.User, prop.Chat)
			continue
		}
		dur, err := time.ParseDuration(prop.Value)
		if err != nil {
			log.Printf("Could not parse duration %s for chat %d due to error: %s", prop.Value, prop.Chat, err)
			continue
		}
		when := tgbotbase.CalcNextTimeFromMidnight(now, dur)
		job := covidJob{
			chatID: prop.Chat,
			ch:     h.toSend,
		}
		h.cron.AddJob(when, &job)
	}
}

func (h *covid19Handler) Name() string {
	return "coronavirus stats at morning"
}

type covidJob struct {
	chatID tgbotbase.ChatID
	ch     chan<- tgbotbase.ChatID
}

var _ tgbotbase.CronJob = &weatherJob{}

const (
	colDate        = 0
	colCountry     = 1
	colNewCases    = 2
	colNewDeaths   = 3
	colTotalCases  = 4
	colTotalDeaths = 5
)

type casesData struct {
	date        time.Time
	newCases    int
	totalCases  int
	newDeaths   int
	totalDeaths int
}

func (job *covidJob) Do(scheduledWhen time.Time, cron tgbotbase.Cron) {
	defer cron.AddJob(scheduledWhen.Add(24*time.Hour), job)

	job.ch <- job.chatID
}

func downloadFile(filepath string, url string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

type covidData struct {
	countryRaw    map[string][]casesData
	countryLatest map[string]casesData
	worldLatest   casesData
}

type covidUpdateJob struct {
	updates chan<- covidData
}

func (j *covidUpdateJob) Do(scheduledWhen time.Time, cron tgbotbase.Cron) {
	defer cron.AddJob(scheduledWhen.Add(1*time.Hour), j)

	url := "https://covid.ourworldindata.org/data/ecdc/full_data.csv"
	fpath := path.Join("/tmp", "ilya-tgbot", "covid")
	if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
		log.Printf("Could not create covid directories at %q, err: %s", fpath, err)
		return
	}
	fname := path.Join(fpath, fmt.Sprintf("cases-%s.csv", time.Now().Format("20060102150405")))
	if err := downloadFile(fname, url); err != nil {
		log.Printf("Could not download covid info from %q to %q, err: %s", url, fname, err)
		return
	}

	f, err := os.Open(fname)
	if err != nil {
		log.Printf("Could not open covid info at %q, err: %s", fname, err)
		return
	}

	r := csv.NewReader(f)
	data, err := r.ReadAll()
	if err != nil {
		log.Printf("Could not read covid info at %q, err: %s", fname, err)
		return
	}

	raw := make(map[string][]casesData, 200)
	latest := make(map[string]casesData, 200)
	dates := make(map[string]time.Time, 200)
	for _, line := range data {
		d, _ := time.Parse("2006-01-02", line[colDate])

		newCases, _ := strconv.Atoi(line[colNewCases])
		totalCases, _ := strconv.Atoi(line[colTotalCases])
		newDeaths, _ := strconv.Atoi(line[colNewDeaths])
		totalDeaths, _ := strconv.Atoi(line[colTotalDeaths])
		cinfo := casesData{
			date:        d,
			newCases:    newCases,
			totalCases:  totalCases,
			newDeaths:   newDeaths,
			totalDeaths: totalDeaths,
		}

		// assuming that dates are ordered
		country := line[colCountry]
		raw[country] = append(raw[country], cinfo)

		date, found := dates[country]
		if found && d.Before(date) {
			continue
		}

		dates[country] = d
		latest[country] = cinfo
	}

	j.updates <- covidData{
		countryRaw:    raw,
		countryLatest: latest,
	}
}
