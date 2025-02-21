package main

import (
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net"
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
	visitedFilePath = "visitedURLs.xml"
	downloadTimeout = 90 * time.Second
)

var (
	visitedURLsMap = &sync.Map{}
	delayedQueue   = make(chan string, 3400)
)

type VisitedURLs struct {
	XMLName xml.Name `xml:"visitedURLs"`
	URLs    []string `xml:"url"`
}

func main() {
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Fatalf("Error listing network interfaces: %s", err)
	}

	fmt.Println("Available network interfaces:")
	for i, iface := range interfaces {
		fmt.Printf("%d: %s\n", i+1, iface.Name)
	}

	var selectedIndex int
	fmt.Print("Select the network interface to use (enter the number): ")
	fmt.Scanln(&selectedIndex)

	if selectedIndex < 1 || selectedIndex > len(interfaces) {
		log.Fatalf("Invalid network interface selection")
	}

	selectedInterface := interfaces[selectedIndex-1]

	dirs, err := os.ReadDir(".")
	if err != nil {
		log.Fatalf("Error reading current directory: %s", err)
	}

	dirList := []os.DirEntry{}
	for _, dir := range dirs {
		if dir.IsDir() {
			dirList = append(dirList, dir)
		}
	}

	fmt.Println("Available directories:")
	for i, dir := range dirList {
		fmt.Printf("%d: %s\n", i+1, dir.Name())
	}

	fmt.Println("0: Create a new directory named 'pdf-scrape-<timestamp>'")

	var selectedDirIndex int
	fmt.Print("Select the directory to store downloaded PDFs (enter the number or hit enter for default): ")
	_, err = fmt.Scanln(&selectedDirIndex)

	var selectedDir string
	if err != nil || selectedDirIndex == 0 {
		timestamp := time.Now().Format("20060102150405")
		selectedDir = fmt.Sprintf("pdf-scrape-%s", timestamp)
		if err := os.Mkdir(selectedDir, 0755); err != nil {
			log.Fatalf("Error creating directory: %s", err)
		}
		fmt.Printf("Created new directory: %s\n", selectedDir)
	} else if selectedDirIndex < 1 || selectedDirIndex > len(dirList) {
		log.Fatalf("Invalid directory selection")
	} else {
		selectedDir = dirList[selectedDirIndex-1].Name()
	}

	loadVisitedURLs()

	var startingURL string
	fmt.Print("Enter the starting URL to crawl: ")
	fmt.Scanln(&startingURL)

	startingURL = normalizeURL(startingURL)

	startingURL, err = sanitizeURL(startingURL)
	if err != nil {
		log.Fatalf("Error sanitizing starting URL: %s", err)
	}

	if err := validateURL(startingURL); err != nil {
		log.Fatalf("Error validating starting URL: %s", err)
	}

	c := colly.NewCollector()
	c.WithTransport(&http.Transport{
		Dial: (&net.Dialer{
			LocalAddr: &net.TCPAddr{
				IP: getInterfaceIP(selectedInterface),
			},
		}).Dial,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	})

	q, err := queue.New(3, &queue.InMemoryQueueStorage{MaxSize: 10000})
	if err != nil {
		log.Fatalf("Error creating queue: %s", err)
	}

	c.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL.String())
	})

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		absoluteURL := e.Request.AbsoluteURL(e.Attr("href"))
		if !hasVisited(absoluteURL) {
			saveVisitedURL(absoluteURL)
			q.AddURL(absoluteURL)
		}
	})

	c.OnHTML("a[href$='.pdf']", func(e *colly.HTMLElement) {
		pdfURL := e.Request.AbsoluteURL(e.Attr("href"))
		log.Printf("Found PDF URL: %s", pdfURL)
		if err := downloadFileWithTimeout(pdfURL, selectedDir); err != nil {
			log.Printf("Error downloading file: %s", err)
		}
	})

	q.AddURL(startingURL)
	go processDelayedQueue(selectedDir)
	log.Println("Starting the crawler...")
	q.Run(c)
	log.Println("Crawler finished.")
}

func loadVisitedURLs() {
	log.Println("Loading visited URLs...")
	data, err := os.ReadFile(visitedFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error reading visited URLs file: %s", err)
		}
		return
	}

	var visitedURLs VisitedURLs
	err = xml.Unmarshal(data, &visitedURLs)
	if err != nil {
		log.Printf("Error unmarshalling visited URLs: %s", err)
		return
	}

	for _, url := range visitedURLs.URLs {
		visitedURLsMap.Store(url, true)
	}
	log.Println("Visited URLs loaded.")
}

func hasVisited(url string) bool {
	_, found := visitedURLsMap.Load(url)
	return found
}

func saveVisitedURL(url string) {
	visitedURLsMap.Store(url, true)

	var visitedURLs VisitedURLs
	data, err := os.ReadFile(visitedFilePath)
	if err == nil {
		xml.Unmarshal(data, &visitedURLs)
	}

	visitedURLs.URLs = append(visitedURLs.URLs, url)

	xmlData, err := xml.MarshalIndent(visitedURLs, "", "  ")
	if err != nil {
		log.Printf("Error marshalling visited URLs: %s", err)
		return
	}

	err = os.WriteFile(visitedFilePath, xmlData, 0644)
	if err != nil {
		log.Printf("Error writing to file: %s", err)
	}
}

func downloadHTTPFile(URL, dir string) error {
	client := &http.Client{
		Timeout: downloadTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Get(URL)
	if err != nil {
		log.Printf("HTTP GET error for %s: %v", URL, err)
		delayedQueue <- URL
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("HTTP status %d for %s", resp.StatusCode, URL)
		return fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	filename := filepath.Base(URL)
	filepath := filepath.Join(dir, filename)

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func processDelayedQueue(selectedDir string) {
	for {
		select {
		case url := <-delayedQueue:
			log.Printf("Retrying download for URL: %s", url)
			err := downloadFileWithTimeout(url, selectedDir)
			if err != nil {
				log.Printf("Error retrying download for URL %s: %s", url, err)
			}
		case <-time.After(1 * time.Minute):
			log.Println("No delayed requests to process")
		}
	}
}

func getInterfaceIP(iface net.Interface) net.IP {
	addrs, err := iface.Addrs()
	if err != nil {
		log.Fatalf("Error getting addresses for interface %s: %s", iface.Name, err)
	}

	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip == nil || ip.IsLoopback() {
			continue
		}
		if ip.To4() != nil {
			return ip
		}
	}

	log.Fatalf("No valid IP address found for interface %s", iface.Name)
	return nil
}

func normalizeURL(urlStr string) string {
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}

	if !strings.HasPrefix(urlStr, "https://www.") {
		urlStr = strings.Replace(urlStr, "https://", "https://www.", 1)
	}

	return urlStr
}

func sanitizeURL(urlStr string) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	// Remove invalid prefixes like "chrome-extension//"
	if strings.HasPrefix(u.Path, "//") {
		u.Path = strings.TrimPrefix(u.Path, "//")
	}

	// Replace colons in the path with URL-encoded equivalent
	u.Path = strings.ReplaceAll(u.Path, ":", "%3A")

	// Rebuild the URL
	return u.String(), nil
}

func downloadFileWithTimeout(URL, dir string) error {
	log.Printf("Downloading %s", URL)

	// Sanitize the URL before processing
	sanitizedURL, err := sanitizeURL(URL)
	if err != nil {
		return fmt.Errorf("error sanitizing URL: %s", err)
	}

	if strings.HasPrefix(sanitizedURL, "http://") || strings.HasPrefix(sanitizedURL, "https://") {
		return downloadHTTPFile(sanitizedURL, dir)
	}

	return fmt.Errorf("unsupported protocol: %s", sanitizedURL)
}

func validateURL(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return err
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported protocol scheme: %s", u.Scheme)
	}

	if u.Host == "" {
		return fmt.Errorf("missing host in URL: %s", urlStr)
	}

	return nil
}
