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
    "github.com/gocolly/colly/debug"
)

func main() {
    // Initialize HyperLogLog with a precision of 14
    hll := hyperloglog.New14()

    var linksProcessed int64
    var memStats runtime.MemStats

    // Create a custom HTTP transport
    customTransport := &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 100,
        IdleConnTimeout:     90 * time.Second,
        TLSHandshakeTimeout: 10 * time.Second,
        TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
    }

    // Create a new collector with debugging enabled
    c := colly.NewCollector(
        colly.Async(true),
        colly.Debugger(&debug.LogDebugger{}),
    )

    c.WithTransport(customTransport)

    // Limit the maximum parallelism
    if err := c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 5}); err != nil {
        log.Fatalf("Error setting limit: %v", err)
    }

    // Map to track visited URLs
    visited := make(map[string]bool)
    var mu sync.Mutex

    // Read the starting URL
    fmt.Print("Enter the starting URL: ")
    reader := bufio.NewReader(os.Stdin)
    startURL, err := reader.ReadString('\n')
    if err != nil {
        log.Fatalf("Error reading input: %v", err)
    }
    startURL = strings.TrimSpace(startURL)
    startURL = preprocessURL(startURL)

    // Create a file to log visited URLs
    fileName := fmt.Sprintf("%s_%s_HyperLogLog.txt", time.Now().Format("2006-01-02T150405"), urlToFileName(startURL))
    file, err := os.Create(fileName)
    if err != nil {
        log.Fatalf("Error creating file: %v", err)
    }
    defer file.Close()

    // Setup a ticker for regular logging
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    go func() {
        for range ticker.C {
            runtime.ReadMemStats(&memStats)
            log.Printf("Links processed: %d, Unique links (estimate): %d, Goroutines: %d\n", linksProcessed, hll.Estimate(), runtime.NumGoroutine())
        }
    }()

    // Log request details
    c.OnRequest(func(r *colly.Request) {
        log.Println("Visiting", r.URL.String())
    })

    // Handle found links
    c.OnHTML("a[href]", func(e *colly.HTMLElement) {
        link := e.Request.AbsoluteURL(e.Attr("href"))
        link = preprocessURL(link)

        mu.Lock()
        if visited[link] {
            mu.Unlock()
            return
        }
        visited[link] = true
        mu.Unlock()

        linksProcessed++
        hll.Insert([]byte(link))

        log.Printf("Visiting: %s\n", link)
        file.WriteString(link + "\n")

        // Attempt to visit the link and log errors
        if err := e.Request.Visit(link); err != nil {
            log.Printf("Error visiting %s: %v", link, err)
        }
    })

    // Start scraping
    if err := c.Visit(startURL); err != nil {
        log.Printf("Error visiting %s: %v", startURL, err)
    }
    c.Wait()
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
