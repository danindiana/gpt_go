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
	"github.com/gocolly/colly/queue"
)

const visitedFilePath = "visitedURLs.txt"

var excludedDomains = []string{"facebook", "youtube", "reddit", "linkedin", "wikipedia", "twitter", "http://pubchem.ncbi.nlm.nih.gov", "ncbi.nlm.nih.gov"}
var visitedURLsMap = make(map[string]bool)
var mapMutex = &sync.Mutex{}

func main() { // main() function is called
	loadVisitedURLs() // loadVisitedURLs() function is called

	var startingURL string
	fmt.Println("Enter the starting URL to crawl: ")
	fmt.Scanln(&startingURL)

	c := colly.NewCollector()

	// create a request queue with 15 consumer threads
	q, _ := queue.New(26, &queue.InMemoryQueueStorage{
		MaxSize: 30000,
	}) // queue.New() function is called
	c.OnRequest(func(r *colly.Request) { // c.OnRequest() function is called
		fmt.Println("Visiting", r.URL.String())
	})

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
		if !hasVisited(absoluteURL) {
			saveVisitedURL(absoluteURL)
			q.AddURL(absoluteURL)
		}
	})

	c.OnHTML("a[href$='.pdf']", func(e *colly.HTMLElement) {
		pdfURL := e.Request.AbsoluteURL(e.Attr("href"))
		err := downloadFile(pdfURL)
		if err != nil {
			fmt.Printf("Error downloading file: %s\n", err)
		}
	})

	q.AddURL(startingURL)
	q.Run(c)
}

func loadVisitedURLs() {
	data, err := os.ReadFile(visitedFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("Error reading visited URLs file: %s\n", err)
		}
		return
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		visitedURLsMap[line] = true
	}
}

func hasVisited(url string) bool {
	mapMutex.Lock()
	_, found := visitedURLsMap[url]
	mapMutex.Unlock()
	return found
}

func saveVisitedURL(url string) {
	mapMutex.Lock()
	visitedURLsMap[url] = true
	mapMutex.Unlock()

	f, err := os.OpenFile(visitedFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
