package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/queue"
)

const (
	visitedFilePath   = "visitedURLs.txt"
	failedFilePath    = "failedURLs.txt"
	pdfURLsFilePath   = "pdf_urls.txt"
	downloadTimeout   = 46 * time.Second
	maxRetryAttempts  = 3
	retryWorkerCount  = 5
)

var (
	excludedDomains = []string{
		"facebook.com", "youtube.com", "reddit.com", "linkedin.com",
		"wikipedia.org", "twitter.com", "pubchem.ncbi.nlm.nih.gov", "ncbi.nlm.nih.gov",
	}
	visitedURLsMap = &sync.Map{}
	failedURLsMap  = &sync.Map{}
	pdfURLsMap     = &sync.Map{}
	retryQueue     = make(chan RetryItem, 100)
	inRetry        = make(map[string]bool)
	retryMutex     sync.Mutex
)

type RetryItem struct {
	URL      string
	Attempts int
}

func main() {
	fmt.Println("Enter the starting URL to crawl: ")
	var startingURL string
	fmt.Scanln(&startingURL)
	startingURL = normalizeURL(startingURL)

	// Prompt for download directory
	fmt.Println("Enter the directory to save downloaded PDFs (or leave blank to use default): ")
	var selectedDir string
	fmt.Scanln(&selectedDir)
	if selectedDir == "" {
		timestamp := time.Now().Format("20060102150405")
		selectedDir = fmt.Sprintf("pdf-scrape-%s", timestamp)
		if err := os.Mkdir(selectedDir, 0755); err != nil {
			log.Fatalf("Error creating directory: %s", err)
		}
	}

	// Load visited and failed URLs
	loadVisitedURLs()
	loadFailedURLs()

	// Initialize Colly collector
	c := colly.NewCollector()

	// Queue setup with 16 threads
	q, err := queue.New(16, &queue.InMemoryQueueStorage{MaxSize: 10000})
	if err != nil {
		log.Fatalf("Error creating queue: %s", err)
	}

	c.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL.String())
	})

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		absoluteURL := e.Request.AbsoluteURL(link)

		if isExcludedDomain(absoluteURL) || hasVisited(absoluteURL) {
			return
		}

		saveVisitedURL(absoluteURL)
		q.AddURL(absoluteURL)
	})

	c.OnHTML("a[href$='.pdf']", func(e *colly.HTMLElement) {
		pdfURL := e.Request.AbsoluteURL(e.Attr("href"))
		log.Printf("Found PDF URL: %s", pdfURL)

		pdfURLsMap.Store(pdfURL, struct{}{})

		if _, failed := failedURLsMap.Load(pdfURL); failed {
			log.Printf("Skipping previously failed URL: %s", pdfURL)
			return
		}

		err := downloadFileWithTimeout(pdfURL, selectedDir)
		if err != nil {
			log.Printf("Error downloading file: %s", err)
			retryMutex.Lock()
			if !inRetry[pdfURL] {
				retryQueue <- RetryItem{URL: pdfURL, Attempts: 1}
				inRetry[pdfURL] = true
			}
			retryMutex.Unlock()
		}
	})

	log.Printf("Adding starting URL to queue: %s", startingURL)
	q.AddURL(startingURL)

	// Start retry workers
	for i := 0; i < retryWorkerCount; i++ {
		go retryWorker(retryQueue, selectedDir)
	}

	log.Println("Starting the crawler...")
	q.Run(c)
	log.Println("Crawler finished.")

	// Save PDF URLs
	savePDFURLs()
}

func loadVisitedURLs() {
	data, err := os.ReadFile(visitedFilePath)
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if line != "" {
				visitedURLsMap.Store(line, true)
			}
		}
	}
}

func loadFailedURLs() {
	data, err := os.ReadFile(failedFilePath)
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if line != "" {
				failedURLsMap.Store(line, true)
			}
		}
	}
}

func hasVisited(url string) bool {
	_, found := visitedURLsMap.Load(url)
	return found
}

func saveVisitedURL(url string) {
	visitedURLsMap.Store(url, true)
	f, err := os.OpenFile(visitedFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		f.WriteString(url + "\n")
	}
}

func saveFailedURL(url string) {
	failedURLsMap.Store(url, true)
	f, err := os.OpenFile(failedFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		f.WriteString(url + "\n")
	}
}

func savePDFURLs() {
	f, err := os.OpenFile(pdfURLsFilePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		pdfURLsMap.Range(func(key, value interface{}) bool {
			fmt.Fprintln(f, key.(string))
			return true
		})
	}
}

func isExcludedDomain(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	for _, domain := range excludedDomains {
		if strings.Contains(u.Host, domain) {
			return true
		}
	}
	return false
}

func normalizeURL(urlStr string) string {
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}
	return urlStr
}

func downloadFileWithTimeout(URL, dir string) error {
	client := &http.Client{Timeout: downloadTimeout}
	resp, err := client.Get(URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	fileName := filepath.Base(URL)
	filePath := filepath.Join(dir, fileName)

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func retryWorker(queue chan RetryItem, dir string) {
	for item := range queue {
		if item.Attempts >= maxRetryAttempts {
			log.Printf("Giving up on URL: %s after %d attempts", item.URL, item.Attempts)
			retryMutex.Lock()
			delete(inRetry, item.URL)
			retryMutex.Unlock()
			saveFailedURL(item.URL)
			continue
		}

		backoff := time.Duration(math.Pow(2, float64(item.Attempts))) * time.Second
		time.Sleep(backoff)

		err := downloadFileWithTimeout(item.URL, dir)
		if err != nil {
			log.Printf("Retry failed for URL %s: %s", item.URL, err)
			queue <- RetryItem{URL: item.URL, Attempts: item.Attempts + 1}
		} else {
			log.Printf("Successfully downloaded URL: %s after %d attempts", item.URL, item.Attempts+1)
			retryMutex.Lock()
			delete(inRetry, item.URL)
			retryMutex.Unlock()
		}
	}
}
