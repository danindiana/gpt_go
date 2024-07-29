package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/queue"
	"golang.org/x/time/rate"
)

// Configuration holds the settings for the crawler
type Configuration struct {
	StartingURL     string
	FileType        string
	RateLimit       float64
	ExcludedDomains []string
	MaxDepth        int
	VisitedFile     string
}

var (
	visitedURLsMap = make(map[string]bool)
	mapMutex       = &sync.Mutex{}
	config         Configuration
)

func main() {
	// Initialize configuration
	config = Configuration{
		ExcludedDomains: []string{"facebook", "youtube", "reddit", "linkedin", "wikipedia", "twitter", "pubchem.ncbi.nlm.nih.gov", "ncbi.nlm.nih.gov"},
		MaxDepth:        3,
		VisitedFile:     "visitedURLs.txt",
	}

	// Load visited URLs from file
	loadVisitedURLs(config.VisitedFile)

	// Get user input
	getUserInput()

	// Initialize rate limiter
	limiter := rate.NewLimiter(rate.Limit(config.RateLimit), 1)

	// Initialize the collector
	c := colly.NewCollector(
		colly.Async(true),
	)

	// Initialize the queue
	q, _ := queue.New(15, &queue.InMemoryQueueStorage{MaxSize: 10000})

	// Set up the request handler
	c.OnRequest(func(r *colly.Request) {
		limiter.Wait(context.Background())
		fmt.Println("Visiting", r.URL.String())
	})

	// Set up the HTML link handler
	c.OnHTML("a[href]", handleLink(q, c))

	// Set up the file download handler
	c.OnHTML(fmt.Sprintf("a[href$='.%s']", config.FileType), handleFileDownload)

	// Start crawling from the starting URL with initial depth 1
	ctx := colly.NewContext()
	ctx.Put("depth", "1")
	c.Request("GET", config.StartingURL, nil, ctx, nil)

	if err := q.Run(c); err != nil {
		fmt.Println("Error running queue:", err)
	}

	// Wait for the collector to finish
	c.Wait()
	fmt.Println("Crawling finished.")
}

// getUserInput prompts the user for input and sets the configuration values
func getUserInput() {
	fmt.Println("Enter the starting URL to crawl: ")
	fmt.Scanln(&config.StartingURL)

	fmt.Println("Enter the type of file to download (options: txt, html, pdf): ")
	fmt.Scanln(&config.FileType)

	fmt.Println("Enter the rate limit (requests per second): ")
	fmt.Scanln(&config.RateLimit)
}

// handleLink processes discovered links and adds them to the queue if they have not been visited
func handleLink(q *queue.Queue, c *colly.Collector) func(*colly.HTMLElement) {
	return func(e *colly.HTMLElement) {
		link := e.Attr("href")
		u, err := url.Parse(link)
		if err != nil {
			return
		}

		for _, domain := range config.ExcludedDomains {
			if strings.Contains(u.Host, domain) {
				return
			}
		}

		absoluteURL := e.Request.AbsoluteURL(link)
		if !hasVisited(absoluteURL) {
			// Get the depth from the request context
			depthStr := e.Request.Ctx.Get("depth")
			depth, err := strconv.Atoi(depthStr)
			if err != nil {
				return
			}

			// Check if the depth is within the allowed limit
			if depth < config.MaxDepth {
				newDepth := depth + 1
				ctx := colly.NewContext()
				ctx.Put("depth", strconv.Itoa(newDepth))

				fmt.Println("Found new URL:", absoluteURL, "at depth:", newDepth)
				saveVisitedURL(absoluteURL, config.VisitedFile)
				c.Request("GET", absoluteURL, nil, ctx, nil)
			}
		}
	}
}

// handleFileDownload processes and downloads the file
func handleFileDownload(e *colly.HTMLElement) {
	fileURL := e.Request.AbsoluteURL(e.Attr("href"))
	fmt.Println("Found file to download:", fileURL)
	err := downloadFile(fileURL)
	if err != nil {
		fmt.Printf("Error downloading file: %s\n", err)
	}
}

// loadVisitedURLs loads visited URLs from the specified file into a map
func loadVisitedURLs(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("Error reading visited URLs file: %s\n", err)
		}
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		visitedURLsMap[line] = true
	}
}

// hasVisited checks if a URL has been visited
func hasVisited(url string) bool {
	mapMutex.Lock()
	defer mapMutex.Unlock()
	_, found := visitedURLsMap[url]
	return found
}

// saveVisitedURL saves a visited URL to the map and appends it to the file
func saveVisitedURL(url, filePath string) {
	mapMutex.Lock()
	defer mapMutex.Unlock()
	visitedURLsMap[url] = true

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error saving URL to file: %s\n", err)
		return
	}
	defer f.Close()

	_, err = f.WriteString(url + "\n")
	if err != nil {
		fmt.Printf("Error writing to file: %s\n", err)
	}
}

// downloadFile downloads a file from the specified URL
func downloadFile(URL string) error {
	resp, err := http.Get(URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	segments := strings.Split(URL, "/")
	filePath := segments[len(segments)-1]

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
