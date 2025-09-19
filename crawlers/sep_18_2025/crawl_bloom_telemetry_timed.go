package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gocolly/colly"
	"github.com/willf/bloom"
)

const (
	filterSize = 1000000
)

func main() {
	// Initialize the Bloom filter
	filter := bloom.New(filterSize, 5)
	
	// Variables for telemetry (using atomic for thread safety)
	var linksProcessed, uniqueLinks int64
	var mu sync.Mutex // Mutex to protect shared variables

	// Read the starting URL from the user
	fmt.Print("Enter the starting URL: ")
	reader := bufio.NewReader(os.Stdin)
	startURL, _ := reader.ReadString('\n')
	startURL = strings.TrimSpace(startURL)
	startURL = preprocessURL(startURL)
	
	if startURL == "" {
		fmt.Println("Invalid starting URL")
		return
	}

	// Parse the starting URL to get the domain for restriction
	parsedStart, err := url.Parse(startURL)
	if err != nil {
		fmt.Println("Error parsing starting URL:", err)
		return
	}
	baseDomain := parsedStart.Hostname()
	// Remove www. for domain matching
	if strings.HasPrefix(baseDomain, "www.") {
		baseDomain = strings.TrimPrefix(baseDomain, "www.")
	}

	fmt.Printf("Crawling domain: %s\n", baseDomain)

	// Create a new collector
	c := colly.NewCollector(
		colly.MaxDepth(3),
		colly.Async(true),
	)

	// Restrict to the same domain
	c.AllowedDomains = []string{baseDomain, "www." + baseDomain}

	// Set reasonable timeouts
	c.SetRequestTimeout(30 * time.Second)

	// Limit the maximum parallelism
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 2,
		Delay:       1 * time.Second,
	})

	// Generate a file name based on the current system date/time and the initial URL
	fileName := fmt.Sprintf("%s_%s.txt", time.Now().Format("2006-01-02T150405"), urlToFileName(startURL))
	file, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	// Set up a ticker to report telemetry at regular intervals
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Stop the ticker when crawling is done
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
				processed := atomic.LoadInt64(&linksProcessed)
				unique := atomic.LoadInt64(&uniqueLinks)
				fmt.Printf("Links processed: %d, Unique links: %d\n", processed, unique)
			case <-done:
				return
			}
		}
	}()

	// On every a element which has href attribute call callback
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		
		// Skip empty links
		if strings.TrimSpace(link) == "" {
			return
		}

		// Get absolute URL
		absLink := e.Request.AbsoluteURL(link)
		processedLink := preprocessURL(absLink)

		if processedLink == "" {
			return
		}

		// Parse URL to check domain
		parsedURL, err := url.Parse(processedLink)
		if err != nil {
			return
		}
		
		linkDomain := parsedURL.Hostname()
		if strings.HasPrefix(linkDomain, "www.") {
			linkDomain = strings.TrimPrefix(linkDomain, "www.")
		}

		// Skip if not the same domain
		if linkDomain != baseDomain {
			return
		}

		atomic.AddInt64(&linksProcessed, 1)

		// Check if the URL is already visited using Bloom filter
		mu.Lock()
		isNew := !filter.TestAndAdd([]byte(processedLink))
		mu.Unlock()

		if isNew {
			atomic.AddInt64(&uniqueLinks, 1)
			output := fmt.Sprintf("Found: %s\n", processedLink)
			fmt.Print(output)
			
			mu.Lock()
			file.WriteString(output)
			mu.Unlock()

			// Visit the link (Colly will handle the queuing in async mode)
			c.Visit(processedLink)
		}
	})

	// Handle request errors
	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Error visiting %s: %v\n", r.Request.URL, err)
	})

	// Log successful requests
	c.OnResponse(func(r *colly.Response) {
		fmt.Printf("Visited: %s\n", r.Request.URL)
	})

	// Start the crawler
	fmt.Printf("Starting crawl at: %s\n", startURL)

	// Add the initial URL to the filter
	mu.Lock()
	filter.Add([]byte(startURL))
	mu.Unlock()

	// Start crawling
	err = c.Visit(startURL)
	if err != nil {
		fmt.Printf("Error starting crawl: %v\n", err)
		return
	}

	// Wait for all async requests to complete
	c.Wait()

	// Stop the telemetry goroutine
	close(done)

	// Log the final telemetry data
	finalProcessed := atomic.LoadInt64(&linksProcessed)
	finalUnique := atomic.LoadInt64(&uniqueLinks)
	telemetryOutput := fmt.Sprintf("Crawl finished.\nTotal links processed: %d\nUnique links found: %d\n",
		finalProcessed, finalUnique)
	fmt.Print(telemetryOutput)
	
	mu.Lock()
	file.WriteString(telemetryOutput)
	mu.Unlock()
}

// preprocessURL preprocesses the input URL to handle variations
func preprocessURL(inputURL string) string {
	if inputURL == "" {
		return ""
	}

	// Handle fragment identifiers
	if idx := strings.Index(inputURL, "#"); idx != -1 {
		inputURL = inputURL[:idx]
	}

	parsedURL, err := url.Parse(inputURL)
	if err != nil {
		return ""
	}

	// Skip non-http(s) URLs
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return ""
	}

	if parsedURL.Host == "" {
		return ""
	}

	// Clean the path (remove trailing slash)
	if parsedURL.Path != "/" {
		parsedURL.Path = strings.TrimSuffix(parsedURL.Path, "/")
	}

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
