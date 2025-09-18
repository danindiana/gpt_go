package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly"
	"github.com/willf/bloom"
)

const (
	filterSize = 100000 // The size of the Bloom filter
)

func main() {
	// Initialize the Bloom filter
	filter := bloom.New(filterSize, 5)

	// Variables for telemetry
	var linksProcessed, uniqueLinks int
	var mu sync.Mutex // Mutex to protect shared variables

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
	fileName := fmt.Sprintf("%s_%s.txt", time.Now().Format("2006-01-02T150405"), urlToFileName(startURL))
	file, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	// Set up a ticker to report telemetry at regular intervals
	ticker := time.NewTicker(5 * time.Second) // Adjust the interval as needed
	defer ticker.Stop()
	go func() {
		for range ticker.C {
			mu.Lock()
			fmt.Printf("Links processed: %d, Unique links: %d\n", linksProcessed, uniqueLinks)
			mu.Unlock()
		}
	}()

	// On every a element which has href attribute call callback
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		// Preprocess the URL to handle variations
		link = preprocessURL(link)
		if link == "" {
			return
		}

		mu.Lock()
		linksProcessed++ // Increment total links processed
		mu.Unlock()

		// Check if the URL is already visited
		if !filter.TestAndAdd([]byte(link)) {
			mu.Lock()
			uniqueLinks++ // Increment unique links count
			output := fmt.Sprintf("Visiting: %s\n", link)
			fmt.Print(output)        // Output to console
			file.WriteString(output) // Write the same output to the file
			mu.Unlock()
			
			// Visit the link
			wg.Add(1)
			go func() {
				defer wg.Done()
				c.Visit(link)
			}()
		}
	})

	// Handle request errors
	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Error visiting %s: %v\n", r.Request.URL, err)
	})

	// Start the crawler
	fmt.Printf("Starting crawl at: %s\n", startURL)
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.Visit(startURL)
	}()

	// Wait for all requests to finish
	wg.Wait()

	// Log the telemetry data
	telemetryOutput := fmt.Sprintf("Crawl finished.\nTotal links processed: %d\nUnique links found: %d\n",
		linksProcessed, uniqueLinks)
	fmt.Print(telemetryOutput)
	file.WriteString(telemetryOutput)
}

// preprocessURL preprocesses the input URL to handle variations
func preprocessURL(inputURL string) string {
	// Handle relative URLs by making them absolute
	if !strings.Contains(inputURL, "://") && !strings.HasPrefix(inputURL, "//") {
		inputURL = "http://" + inputURL
	}

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

	// Clean the path to remove unnecessary segments
	parsedURL.Path = strings.TrimSuffix(parsedURL.Path, "/")

	return parsedURL.String()
}

// urlToFileName sanitizes the URL to be used in a file name
func urlToFileName(url string) string {
	sanitized := strings.ReplaceAll(url, "http://", "")
	sanitized = strings.ReplaceAll(sanitized, "https://", "")
	sanitized = strings.ReplaceAll(sanitized, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, ":", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	sanitized = strings.ReplaceAll(sanitized, "?", "_")
	sanitized = strings.ReplaceAll(sanitized, "&", "_")
	sanitized = strings.ReplaceAll(sanitized, "=", "_")
	sanitized = strings.ReplaceAll(sanitized, "%", "_")
	
	// Limit length to avoid OS filename limits
	if len(sanitized) > 100 {
		sanitized = sanitized[:100]
	}
	
	return sanitized
}
