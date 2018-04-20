package cmd

import "log"
import "os"
import "path"
import "math/rand"
import "io"
import "io/ioutil"
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

const cache_dir = "tgbot-cache"
const cache_pic_subdir = "pics"

func NewPicCache(cacheName string) PicCache {
    home_dir := os.Getenv("HOME")
    if home_dir == "" {
        log.Fatal("Home directory cannot be retrieved from environment")
    }
    cache := PicCache{
                name: cacheName,
                location:  path.Join(home_dir, cache_dir, cache_pic_subdir, cacheName),
                filenames: make([]string, 0, 100),
                curPos: 0,
                loadBatchSize: 5,
                loadLowLimit: 2}

    err := os.RemoveAll(cache.location)
    if err != nil {
        log.Fatalf("Could not remove contents of cache storage directory '%s' due to error: %s", cache.location, err)
    }

    err = os.MkdirAll(cache.location, os.ModePerm)
    if err != nil {
        log.Fatalf("Could not create cache storage directory at '%s' due to error: %s", cache.location, err)
    }
    log.Printf("New cache has been requested for '%s'; final location: %s", cache.name, cache.location)

    loadMoreKitties(cache.location, cache.loadBatchSize)

    fs_files, err := ioutil.ReadDir(cache.location)
    if err != nil {
        log.Fatalf("Can't read %s to get a list of files. Error: %s", cache.location, err)
    }
    log.Printf("Found %d files at %s; loading them to cache %s", len(fs_files), cache.location, cache.name)

    // shuffling (rand.Shuffle is available only in go 1.10)
    for i := range fs_files {
        j := rand.Intn(i + 1)
        fs_files[i], fs_files[j] = fs_files[j], fs_files[i]
    }

    allowed_ext := map[string]bool{".jpg": true, ".jpeg": true, ".png": true}
    for _, f := range(fs_files) {
        if allowed_ext[path.Ext(f.Name())] != true {
            log.Printf("File '%s' doesn't have expected extenstion for cache %s. Removing it", f.Name(), cache.name)
            _ = os.Remove(path.Join(cache.location, f.Name()))
            continue
        }

        cache.filenames = append(cache.filenames, f.Name())
    }

    return cache
}

func (cache *PicCache) GetNext() string {
    if uint(len(cache.filenames)) - cache.curPos < cache.loadLowLimit {
        log.Printf("Need to grow pic cache %s: curPos: %d, curSize: %d", cache.location, cache.curPos, len(cache.filenames))
        // TODO: goroutine
        // TODO: make it independent from kitties
        cache.filenames = append(cache.filenames, loadMoreKitties(cache.location, cache.loadBatchSize)...)
    }

    filename := path.Join(cache.location, cache.filenames[cache.curPos])
    cache.curPos += 1
    log.Printf("Pic cache '%s' returns file '%s' as a result (total cached: %d, current position: %d)", cache.name,
                                                                                                        filename,
                                                                                                        len(cache.filenames),
                                                                                                        cache.curPos)

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
        file, err := os.Create(path.Join(location, filename))
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
        log.Printf("Saved cat '%s' to the filesystem at '%s'", filename, location)
        filenames = append(filenames, filename)
    }

    log.Printf("Total %d of cats (out of %d requested) have been loaded", len(filenames), loadBatchSize)
    return filenames
}

var kittiesWords = []string{"^кот$", "^котэ$", "^котик*", "^котятк*"}

type kittiesHandler struct {
    cache PicCache
}

func NewKittiesHandler() CommandHandler {
    handler := kittiesHandler{}
    handler.cache = NewPicCache("kitties")
    return &handler
}

func (handler *kittiesHandler) HandleMsg(msg *tgbotapi.Update, ctx Context) (*Result, error) {
    if !msgMatches(msg.Message.Text, kittiesWords) {
        return nil, nil
    }

    picMsg := tgbotapi.NewPhotoUpload(msg.Message.Chat.ID, handler.cache.GetNext())

    result := NewResult()
    result.Reply = &picMsg
    return &result, nil
}
