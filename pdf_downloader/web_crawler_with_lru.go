package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	lru "github.com/hashicorp/golang-lru"
)

var (
	visitedURLsCache  *lru.Cache
	cacheCapacity     = 1000 // Setting the capacity of LRU cache to 1000
	visitedURLsMutex  = &sync.Mutex{}
	excludedDomains   = []string{"facebook", "youtube", "reddit", "linkedin"}
)

func main() {
	var startingURL string
	fmt.Println("Enter the starting URL to crawl: ")
	fmt.Scanln(&startingURL)

	c := colly.NewCollector(
		colly.Async(true),
	)

	c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 15})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	// Initialize the LRU cache
	var err error
	visitedURLsCache, err = lru.New(cacheCapacity)
	if err != nil {
		fmt.Println("Error initializing LRU cache:", err)
		return
	}

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")

		u, err := url.Parse(link)
		if err != nil {
			return
		}

		for _, domain := range excludedDomains {
			if strings.Contains(u.Host, domain) {
				return
			}
		}

		absoluteURL := e.Request.AbsoluteURL(link)
		visitedURLsMutex.Lock()
		if !visitedURLsCache.Contains(absoluteURL) {
			visitedURLsCache.Add(absoluteURL, nil)
			e.Request.Visit(link)
		}
		visitedURLsMutex.Unlock()
	})

	c.OnHTML("a[href$='.pdf']", func(e *colly.HTMLElement) {
		pdfURL := e.Request.AbsoluteURL(e.Attr("href"))
		err := downloadFile(pdfURL)
		if err != nil {
			fmt.Printf("Error downloading file: %s\n", err)
		}
	})

	c.Visit(startingURL)
	c.Wait()
}

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
