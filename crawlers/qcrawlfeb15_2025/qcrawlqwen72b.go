package main

import (
    "crypto/tls"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "time"

    "github.com/gocolly/colly"
    "github.com/gocolly/colly/queue"
)

const (
    visitedFilePath = "visitedURLs.txt"
    downloadTimeout = 90 * time.Second // Timeout for downloading PDFs
    maxRetries      = 3                // Maximum number of retries for failed downloads
)

var (
    visitedURLsMap = &sync.Map{}
    delayedQueue   = make(chan string, 3400) // Channel to store delayed requests
    httpClient     *http.Client              // Shared HTTP client
)

func main() {
    // Set up shared HTTP client
    httpClient = &http.Client{
        Timeout: downloadTimeout,
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Skip certificate verification
        },
    }

    // Prompt for starting URL
    var startingURL string
    fmt.Println("Enter the starting URL to crawl: ")
    fmt.Scanln(&startingURL)

    // Normalize, sanitize, and validate the starting URL
    startingURL = processURL(startingURL)
    if err := validateURL(startingURL); err != nil {
        log.Fatalf("Error validating starting URL: %s", err)
    }

    // Load visited URLs
    loadVisitedURLs()

    // Define the directory to store downloads
    selectedDir := "downloads"
    if err := os.MkdirAll(selectedDir, 0755); err != nil {
        log.Fatalf("Error creating directory: %s", err)
    }

    // Create the collector and queue
    c := colly.NewCollector()
    q, err := queue.New(8, &queue.InMemoryQueueStorage{MaxSize: 10000})
    if err != nil {
        log.Fatalf("Error creating queue: %s", err)
    }
    c.WithTransport(httpClient.Transport) // Set the transport directly on the collector

    // Define request and HTML handling
    c.OnRequest(func(r *colly.Request) {
        log.Println("Visiting", r.URL.String())
    })

    c.OnHTML("a[href]", func(e *colly.HTMLElement) {
        link := e.Attr("href")
        absoluteURL := e.Request.AbsoluteURL(link)
        if !hasVisited(absoluteURL) {
            saveVisitedURL(absoluteURL)
            q.AddURL(absoluteURL)
        }
    })

    c.OnHTML("a[href$='.pdf']", func(e *colly.HTMLElement) {
        pdfURL := e.Request.AbsoluteURL(e.Attr("href"))
        log.Printf("Found PDF URL: %s", pdfURL)
        err := downloadFile(pdfURL, selectedDir)
        if err != nil {
            log.Printf("Error downloading file: %s", err)
        }
    })

    log.Printf("Adding starting URL to queue: %s", startingURL)
    q.AddURL(startingURL)

    log.Println("Starting the crawler...")
    q.Run(c)
    log.Println("Crawler finished.")
}

func processURL(urlStr string) string {
    if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
        urlStr = "https://" + urlStr
    }
    return urlStr
}

func validateURL(urlStr string) error {
    if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
        return fmt.Errorf("invalid URL scheme: %s", urlStr)
    }
    return nil
}

func loadVisitedURLs() {
    file, err := os.Open(visitedFilePath)
    if err != nil {
        return
    }
    defer file.Close()
    var line string
    for {
        _, err := fmt.Fscanln(file, &line)
        if err != nil {
            break
        }
        visitedURLsMap.Store(line, true)
    }
}

func hasVisited(url string) bool {
    _, found := visitedURLsMap.Load(url)
    return found
}

func saveVisitedURL(url string) {
    visitedURLsMap.Store(url, true)
    file, err := os.OpenFile(visitedFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        log.Println("Error opening file to save URL:", err)
        return
    }
    defer file.Close()
    fmt.Fprintln(file, url)
}

func downloadFile(URL, dir string) error {
    resp, err := httpClient.Get(URL)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
    }
    
    segments := strings.Split(URL, "/")
    fileName := segments[len(segments)-1]
    filePath := filepath.Join(dir, fileName)
    
    out, err := os.Create(filePath)
    if err != nil {
        return err
    }
    defer out.Close()
    
    _, err = io.Copy(out, resp.Body)
    return err
}
