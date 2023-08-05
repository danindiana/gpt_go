package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url" // Import the url package
	"os"
	"strings"
	"sync"

	"github.com/gocolly/colly"
)

var (
	visitedURLs      = make(map[string]bool)
	visitedURLsMutex = &sync.Mutex{}
	excludedDomains  = []string{"facebook", "youtube", "reddit"}
)

func main() {
	// Prompt the user for a URL
	var startingURL string
	fmt.Println("Enter the starting URL to crawl: ")
	fmt.Scanln(&startingURL)

	// Instantiate default collector
	c := colly.NewCollector(
		colly.Async(true),
	)

	c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 5})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	// Find and visit all links
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")

		u, err := url.Parse(link) // Use the url package to parse the link
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
		if _, found := visitedURLs[absoluteURL]; !found {
			visitedURLs[absoluteURL] = true
			e.Request.Visit(link)
		}
		visitedURLsMutex.Unlock()
	})

	// Download PDF files
	c.OnHTML("a[href$='.pdf']", func(e *colly.HTMLElement) {
		pdfURL := e.Request.AbsoluteURL(e.Attr("href"))
		err := downloadFile(pdfURL)
		if err != nil {
			fmt.Printf("Error downloading file: %s\n", err)
		}
	})

	// Start scraping
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
