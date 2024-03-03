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
    // Initialize the HyperLogLog Sketch with 2^14 registers (precision 14)
    hll := hyperloglog.New14()

    // Variables for telemetry
    var linksProcessed int64
    var memStats runtime.MemStats

    // Create a custom HTTP transport
    customTransport := &http.Transport{
        MaxIdleConns:        100,
        IdleConnTimeout:     90 * time.Second,
        TLSHandshakeTimeout: 10 * time.Second,
        TLSClientConfig:     &tls.Config{InsecureSkipVerify: true}, // For demonstration; be cautious in production
    }

    // Create a new collector and apply the custom transport
    c := colly.NewCollector(
        colly.MaxDepth(5), // Adjusted depth
        colly.Async(true),
    )
    c.WithTransport(customTransport) // Set the custom HTTP transport

    // Limit the maximum parallelism to 5 to reduce server load and potential blocking
    err := c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 5})
    if err != nil {
        log.Fatalf("Error setting limit: %v", err)
    }

    // Read the starting URL from the user
    fmt.Print("Enter the starting URL: ")
    reader := bufio.NewReader(os.Stdin)
    startURL, err := reader.ReadString('\n')
    if err != nil {
        log.Fatalf("Error reading URL: %v", err)
    }
    startURL = strings.TrimSpace(startURL)
    startURL = preprocessURL(startURL)

    // Generate a file name based on the current system date/time and the initial URL
    fileName := fmt.Sprintf("%s_%s_HyperLogLog.txt", time.Now().Format("2006-01-02T150405"), urlToFileName(startURL))
    file, err := os.Create(fileName)
    if err != nil {
        log.Fatalf("Error creating file: %v", err)
    }
    defer file.Close()

    // Create a wait group to wait for all requests to finish
    var wg sync.WaitGroup

    // Start the crawler
    fmt.Printf("Starting crawl at: %s\n", startURL)
    c.Visit(startURL)

    // Set up a ticker to report telemetry at regular intervals
    ticker := time.NewTicker(5 * time.Second)
    go func() {
        for range ticker.C {
            runtime.ReadMemStats(&memStats)
            fmt.Printf("Links processed: %d, Unique links (estimate): %d, Cache misses: %d, Goroutines: %d\n",
                linksProcessed, hll.Estimate(), memStats.Mallocs-memStats.Frees, runtime.NumGoroutine())
        }
    }()

    // On every <a> element which has href attribute call callback
    c.OnHTML("a[href]", func(e *colly.HTMLElement) {
        wg.Add(1)
        defer wg.Done()

        link := e.Attr("href")
        link = preprocessURL(link)
        if link == "" {
            return
        }

        linksProcessed++
        hll.Insert([]byte(link))

        output := fmt.Sprintf("Visiting: %s\n", link)
        fmt.Print(output)
        file.WriteString(output + "\n") // Ensure new line for each link

        // Visit the link
        err := e.Request.Visit(link)
        if err != nil {
            log.Printf("Error visiting link: %s, error: %v\n", link, err)
        }
    })

    // Wait for all requests to finish
    c.Wait()
    wg.Wait()

    // Stop the ticker
    ticker.Stop()

    // Log the telemetry data
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
