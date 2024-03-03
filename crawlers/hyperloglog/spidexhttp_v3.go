package main

import (
    "bufio"
    "crypto/tls"
    "fmt"
    "log"
    "net/http"
    "net/url"
    "os"
    "runtime"
    "strings"
    "sync"
    "time"

    "github.com/axiomhq/hyperloglog"
    "github.com/gocolly/colly"
)

func main() {
    hll := hyperloglog.New14()

    var linksProcessed int64
    var memStats runtime.MemStats

    customTransport := &http.Transport{
        MaxIdleConns:        1000,
        IdleConnTimeout:     20 * time.Second,
        TLSHandshakeTimeout: 10 * time.Second,
        TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
    }

    c := colly.NewCollector(
        colly.MaxDepth(10),
        colly.Async(true),
    )
    c.WithTransport(customTransport)

    err := c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 5})
    if err != nil {
        log.Fatalf("Error setting limit: %v", err)
    }

    visitedURLs := make(map[string]bool)

    fmt.Print("Enter the starting URL: ")
    reader := bufio.NewReader(os.Stdin)
    startURL, err := reader.ReadString('\n')
    if err != nil {
        log.Fatalf("Error reading URL: %v", err)
    }
    startURL = strings.TrimSpace(startURL)
    startURL = preprocessURL(startURL)

    fileName := fmt.Sprintf("%s_%s_HyperLogLog.txt", time.Now().Format("2006-01-02T150405"), urlToFileName(startURL))
    file, err := os.Create(fileName)
    if err != nil {
        log.Fatalf("Error creating file: %v", err)
    }
    defer file.Close()

    var wg sync.WaitGroup

    fmt.Printf("Starting crawl at: %s\n", startURL)
    c.Visit(startURL)

    ticker := time.NewTicker(15 * time.Second)
    go func() {
        for range ticker.C {
            runtime.ReadMemStats(&memStats)
            fmt.Printf("Links processed: %d, Unique links (estimate): %d, Cache misses: %d, Goroutines: %d\n",
                linksProcessed, hll.Estimate(), memStats.Mallocs-memStats.Frees, runtime.NumGoroutine())
        }
    }()

    c.OnHTML("a[href]", func(e *colly.HTMLElement) {
        link := e.Attr("href")
        link = preprocessURL(link)
        if link == "" || visitedURLs[link] {
            return
        }

        visitedURLs[link] = true
        linksProcessed++
        hll.Insert([]byte(link))

        output := fmt.Sprintf("Visiting: %s\n", link)
        fmt.Print(output)
        file.WriteString(output + "\n")

        err := e.Request.Visit(link)
        if err != nil {
            log.Printf("Error visiting link: %s, error: %v\n", link, err)
        }
    })

    c.Wait()
    wg.Wait()

    ticker.Stop()

    telemetryOutput := fmt.Sprintf("Crawl finished.\nTotal links processed: %d\nUnique links found (estimate): %d\n",
        linksProcessed, hll.Estimate())
    fmt.Print(telemetryOutput)
    file.WriteString(telemetryOutput)
}

func preprocessURL(inputURL string) string {
    parsedURL, err := url.Parse(inputURL)
    if err != nil {
        return ""
    }

    if parsedURL.Scheme == "" {
        parsedURL.Scheme = "http"
    }

    if parsedURL.Host == "" {
        return ""
    }

    hostname := strings.ToLower(parsedURL.Hostname())
    if strings.HasPrefix(hostname, "www.") {
        hostname = strings.TrimPrefix(hostname, "www.")
    }
    parsedURL.Host = hostname

    return parsedURL.String()
}

func urlToFileName(url string) string {
    sanitized := strings.ReplaceAll(url, "http://", "")
    sanitized = strings.ReplaceAll(sanitized, "https://", "")
    sanitized = strings.ReplaceAll(sanitized, "/", "_")
    sanitized = strings.ReplaceAll(sanitized, ":", "_")
    sanitized = strings.ReplaceAll(sanitized, ".", "_")
    return sanitized
}
