package main

import (
    "bufio"
    "fmt"
    "io"
    "net/http"
    "net/url"
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
    // Get the file path from user
    var filePath string
    fmt.Println("Enter the location of the URLs text file: ")
    fmt.Scanln(&filePath)

    urls, err := readLines(filePath)
    if err != nil {
        fmt.Printf("Error reading file: %s\n", err)
        os.Exit(1)
    }

    // Instantiate default collector
    c := colly.NewCollector(
        colly.Async(true),
    )

    c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 2})

    c.OnRequest(func(r *colly.Request) {
        fmt.Println("Visiting", r.URL.String())
    })

    // Find and visit all links
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
    for _, u := range urls {
        c.Visit(u)
    }
    c.Wait()
}

// readLines reads a whole file into memory
// and returns a slice of its lines.
func readLines(path string) ([]string, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var lines []string
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }
    return lines, scanner.Err()
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
