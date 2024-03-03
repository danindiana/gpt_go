package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly"
	"github.com/clarkduvall/hyperloglog"
)

func main() {
	// Initialize the HyperLogLog
	hll := hyperloglog.New()

	// Variables for telemetry
	var linksProcessed int64
	var memStats runtime.MemStats

	// Create a new collector
	c := colly.NewCollector(
		colly.MaxDepth(12), // Set the maximum depth to 12
		colly.Async(true),  // Enable asynchronous network requests
	)

	// Limit the maximum parallelism to 12
	c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 12})

	// Create a wait group to wait for all requests to finish
	var wg sync.WaitGroup

	// Read the starting URL from the user
	fmt.Print("Enter the starting URL: ")
	reader := bufio.NewReader(os.Stdin)
	startURL, _ := reader.ReadString('\n')
	startURL = strings.TrimSpace(startURL)
	startURL = preprocessURL(startURL)

	// Generate a file name based on the current system date/time and the initial URL
	fileName := fmt.Sprintf("%s_%s_HyperLogLog.txt", time.Now().Format("2006-01-02T150405"), urlToFileName(startURL))
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatalf("Error creating file: %v", err)
	}
	defer file.Close()

	// Start the crawler
	fmt.Printf("Starting crawl at: %s\n", startURL)
	c.Visit(startURL)

	// Set up a ticker to report telemetry at regular intervals
	ticker := time.NewTicker(5 * time.Second) // Adjust the interval as needed
	go func() {
		for range ticker.C {
			runtime.ReadMemStats(&memStats)
			fmt.Printf("Links processed: %d, Unique links (estimate): %d, Cache misses: %d\n", linksProcessed, hll.Estimate(), memStats.Mallocs-memStats.Frees)
		}
	}()

	// On every a element which has href attribute call callback
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		wg.Add(1) // Increment the wait group counter
		defer wg.Done() // Decrement the wait group counter when the goroutine is done

		link := e.Attr("href")
		// Preprocess the URL to handle variations
		link = preprocessURL(link)
		if link == "" {
			return
		}

		linksProcessed++ // Increment total links processed

		// Add the link to the HyperLogLog
		hll.Insert([]byte(link))

		output := fmt.Sprintf("Visiting: %s\n", link)
		fmt.Print(output)        // Output to console
		file.WriteString(output) // Write the same output to the file

		// Add the link to the collector to be visited
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

// preprocessURL preprocesses the input URL to handle variations
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

// urlToFileName sanitizes the URL to be used in a file name
func urlToFileName(url string) string {
	sanitized := strings.ReplaceAll(url, "http://", "")
	sanitized = strings.ReplaceAll(sanitized, "https://", "")
	sanitized = strings.ReplaceAll(sanitized, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, ":", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	return sanitized
}
