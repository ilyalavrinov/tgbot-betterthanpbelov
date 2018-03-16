package mybot

import "log"
import "os"
import "io"
import "strings"
import "net/http"
import "gopkg.in/telegram-bot-api.v4"


type PicCache struct {
    name string
    location string     // folder which contains cached files
    filenames []string  // just short names
    curPos uint         // current position in a list of filenames
    loadBatchSize uint
    loadLowLimit uint
}

func NewPicCache(cacheName string) *PicCache {
    cache := &PicCache{
                name: cacheName,
                location: "/home/ilyalavrinov/" + cacheName + "/",
                filenames: make([]string, 0, 100),
                curPos: 0,
                loadBatchSize: 20,
                loadLowLimit: 5}

    // TODO: load exiting from disk

    return cache
}

func (cache *PicCache) GetNext() string {
    if uint(len(cache.filenames)) - cache.curPos < cache.loadLowLimit {
        log.Printf("Need to grow pic cache %s: curPos: %d, curSize: %d", cache.location, cache.curPos, len(cache.filenames))
        // TODO: goroutine
        // TODO: make it independent from kitties
        cache.filenames = append(cache.filenames, loadMoreKitties(cache.location, cache.loadBatchSize)...)
    }

    filename := cache.location + cache.filenames[cache.curPos]
    cache.curPos += 1

    return filename
}

func loadMoreKitties(location string, loadBatchSize uint) []string {
    const url = "http://thecatapi.com/api/images/get?format=src&type=jpg"

    log.Printf("Preparing to load %d catpics to %s using %s", loadBatchSize, location, url)
    err := os.MkdirAll(location, os.ModePerm)
    if err != nil {
        log.Fatalf("Could not create cat storage directory at %s due to error: %s", location, err)
    }
    filenames := make([]string, 0, loadBatchSize)
    for i := 0; i < int(loadBatchSize); i++ {
        resp, err := http.Get(url)
        if err != nil {
            log.Printf("Error has been occured during loading cat No. %d: %s. Aborting loading", i, err)
            break
        }
        defer resp.Body.Close()

        actualUrl := resp.Request.URL.String()
        log.Printf("Cat %d received from %s", i, actualUrl)
        actualUrlParts := strings.Split(actualUrl, "/")
        filename := actualUrlParts[len(actualUrlParts) - 1] // getting last piece as actual filename
        file, err := os.Create(location + filename)
        if err != nil {
            log.Printf("Could not create new file for a cat %s due to error: %s. Skipping this one", filename, err)
            continue
        }
        // Use io.Copy to just dump the response body to the file. This supports huge files
        _, err = io.Copy(file, resp.Body)
        if err != nil {
            // TODO: remove created file
            log.Printf("Could not store a catpic from the Internet to %s due to error: %s", filename, err)
            continue
        }
        file.Close()
        log.Printf("Saved cat %s to the filesystem", filename)
        filenames = append(filenames, filename)
    }

    log.Printf("Total %d of cats (out of %d requested) have been loaded", len(filenames), loadBatchSize)
    return filenames
}

var kittiesCache *PicCache = nil

func sendKitties(update tgbotapi.Update) (tgbotapi.PhotoConfig, error) {
    log.Print("Sending kitties")

    if kittiesCache == nil {
        kittiesCache = NewPicCache("kitties")
    }
    // TODO: send ID if this cat has been already sent
    picMsg := tgbotapi.NewPhotoUpload(update.Message.Chat.ID, kittiesCache.GetNext())
    return picMsg, nil
}
