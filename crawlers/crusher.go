package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/gocolly/colly"
	"github.com/willf/bloom"
)

const (
	filterSize = 100000 // The size of the Bloom filter
)

func main() {
	// Initialize the Bloom filter
	filter := bloom.New(filterSize, 5)

	// Create a new collector
	c := colly.NewCollector()

	// On every a element which has href attribute call callback
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		// Preprocess the URL to handle variations
		link = preprocessURL(link)
		if link == "" {
			return
		}

		// Check if the URL is already visited
		if !filter.TestAndAdd([]byte(link)) {
			fmt.Printf("Visiting: %s\n", link)
			// Visit the link
			e.Request.Visit(link)
		}
	})

	// Read the starting URL from the user
	fmt.Print("Enter the starting URL: ")
	reader := bufio.NewReader(os.Stdin)
	startURL, _ := reader.ReadString('\n')
	startURL = strings.TrimSpace(startURL)
	startURL = preprocessURL(startURL)

	// Start the crawler
	fmt.Printf("Starting crawl at: %s\n", startURL)
	c.Visit(startURL)

	// Wait until the crawling is finished
	c.Wait()
}

// preprocessURL preprocesses the input URL to handle variations
func preprocessURL(inputURL string) string {
	parsedURL, err := url.Parse(inputURL)
	if err != nil {
		return ""
	}

	// If the scheme is missing, assume http
	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "http"
	}

	// If the host is missing, the URL is invalid
	if parsedURL.Host == "" {
		return ""
	}

	// Normalize the URL by converting it to lowercase and removing www. if present
	hostname := strings.ToLower(parsedURL.Hostname())
	if strings.HasPrefix(hostname, "www.") {
		hostname = strings.TrimPrefix(hostname, "www.")
	}
	parsedURL.Host = hostname

	return parsedURL.String()
}
