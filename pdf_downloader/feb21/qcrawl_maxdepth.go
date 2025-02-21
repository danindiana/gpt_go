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
)

const (
	visitedFilePath = "visitedURLs.xml"
	downloadTimeout = 90 * time.Second
	maxRetries      = 3                // Maximum number of retries for failed downloads
	batchInterval   = 100 * time.Second // Interval for batching visited URL writes
)

var (
	visitedURLsMap = &sync.Map{}             // Thread-safe map for visited URLs
	visitedQueue   = make(chan string, 1000) // Channel for batching visited URLs
)

type VisitedURLs struct {
	XMLName xml.Name `xml:"visitedURLs"`
	URLs    []string `xml:"url"`
}

func main() {
	// --- Network Interface Selection ---
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

	// --- Directory Selection ---
	dirs, err := os.ReadDir(".")
	if err != nil {
		log.Fatalf("Error reading current directory: %s", err)
	}
	var dirList []os.DirEntry
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

	// --- Load Previously Visited URLs ---
	loadVisitedURLs()
	go visitedURLSaver()

	// --- Get and Prepare the Starting URL ---
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

	// --- Create the Collector with a MaxDepth (set to 5 here, use 0 for unlimited) ---
	c := colly.NewCollector(
		colly.Async(true),
		colly.MaxDepth(12),
	)

	// Configure HTTP transport with connection pooling and selected interface.
	transport := &http.Transport{
		Dial: (&net.Dialer{
			LocalAddr: &net.TCPAddr{
				IP: getInterfaceIP(selectedInterface),
			},
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			// Force HTTP/1.1 by setting NextProtos:
			NextProtos: []string{"http/1.1"},
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}

	c.WithTransport(transport)

	// --- Callbacks ---
	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Error on %s: %s", r.Request.URL.String(), err)
	})
	c.OnRequest(func(r *colly.Request) {
		log.Printf("Visiting: %s (Depth: %d)", r.URL.String(), r.Depth)
	})
	c.OnResponse(func(r *colly.Response) {
		log.Printf("Fetched %d bytes from %s (Depth: %d)", len(r.Body), r.Request.URL, r.Request.Depth)
	})

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		absoluteURL := e.Request.AbsoluteURL(e.Attr("href"))
		// Skip empty or fragment-only URLs.
		if absoluteURL == "" || strings.HasPrefix(absoluteURL, "#") {
			return
		}
		log.Printf("Discovered link (Parent Depth: %d): %s", e.Request.Depth, absoluteURL)
		if !hasVisited(absoluteURL) {
			log.Printf("Enqueuing new URL: %s", absoluteURL)
			markVisited(absoluteURL)
			// Use e.Request.Visit to enqueue the URL with inherited depth.
			if err := e.Request.Visit(absoluteURL); err != nil {
				log.Printf("Error visiting %s: %s", absoluteURL, err)
			}
		}
	})

	c.OnHTML("a[href$='.pdf']", func(e *colly.HTMLElement) {
		pdfURL := e.Request.AbsoluteURL(e.Attr("href"))
		log.Printf("Found PDF URL: %s (Depth: %d)", pdfURL, e.Request.Depth)
		if err := downloadFileWithTimeout(pdfURL, selectedDir); err != nil {
			log.Printf("Error downloading file: %s", err)
			go enqueueRetry(pdfURL, selectedDir, 1)
		}
	})

	// --- Start the Crawl ---
	if err := c.Visit(startingURL); err != nil {
		log.Fatalf("Error visiting starting URL: %s", err)
	}

	c.Wait() // Wait for all asynchronous tasks to finish
	log.Println("Crawler finished.")
	time.Sleep(2 * batchInterval)
}

func loadVisitedURLs() {
	log.Println("Loading visited URLs...")
	data, err := os.ReadFile(visitedFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("Visited URLs file not found, starting fresh.")
			return
		}
		log.Printf("Error reading visited URLs file: %s", err)
		return
	}
	if len(data) == 0 {
		log.Println("Visited URLs file is empty.")
		return
	}
	var visited VisitedURLs
	if err := xml.Unmarshal(data, &visited); err != nil {
		log.Printf("Error unmarshalling visited URLs: %s", err)
		return
	}
	for _, url := range visited.URLs {
		visitedURLsMap.Store(url, true)
	}
	log.Println("Visited URLs loaded.")
}

func visitedURLSaver() {
	ticker := time.NewTicker(batchInterval)
	defer ticker.Stop()
	var urls []string
	for {
		select {
		case url, ok := <-visitedQueue:
			if !ok {
				return
			}
			urls = append(urls, url)
		case <-ticker.C:
			if len(urls) > 0 {
				flushVisitedURLs(urls)
				urls = nil
			}
		}
	}
}

func flushVisitedURLs(urls []string) {
	var visited VisitedURLs
	data, err := os.ReadFile(visitedFilePath)
	if err == nil {
		xml.Unmarshal(data, &visited)
	}
	visited.URLs = append(visited.URLs, urls...)
	xmlData, err := xml.MarshalIndent(visited, "", "  ")
	if err != nil {
		log.Printf("Error marshalling visited URLs: %s", err)
		return
	}
	tempFile, err := os.CreateTemp(".", "visitedURLs*.tmp")
	if err != nil {
		log.Printf("Error creating temp file: %s", err)
		return
	}
	defer tempFile.Close()
	if _, err := tempFile.Write(xmlData); err != nil {
		log.Printf("Error writing to temp file: %s", err)
		return
	}
	if err := tempFile.Sync(); err != nil {
		log.Printf("Error syncing temp file: %s", err)
		return
	}
	if err := os.Rename(tempFile.Name(), visitedFilePath); err != nil {
		log.Printf("Error renaming temp file: %s", err)
		return
	}
}

func hasVisited(url string) bool {
	_, found := visitedURLsMap.Load(url)
	return found
}

func markVisited(url string) {
	visitedURLsMap.Store(url, true)
	select {
	case visitedQueue <- url:
	default:
		// Skip if channel is full.
	}
}

func enqueueRetry(URL, dir string, attempt int) {
	if attempt > maxRetries {
		log.Printf("Max retries exceeded for %s", URL)
		return
	}
	backoffDuration := time.Duration(1<<attempt) * time.Second
	time.Sleep(backoffDuration)
	log.Printf("Retrying download for %s (attempt %d)", URL, attempt)
	err := downloadFileWithTimeout(URL, dir)
	if err != nil {
		log.Printf("Retry %d failed for %s: %s", attempt, URL, err)
		enqueueRetry(URL, dir, attempt+1)
	}
}

func downloadHTTPFile(URL, dir string) error {
	transport := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}
	client := &http.Client{
		Timeout:   downloadTimeout,
		Transport: transport,
	}
	resp, err := client.Get(URL)
	if err != nil {
		return fmt.Errorf("HTTP GET error for %s: %w", URL, err)
	}
	if resp == nil {
		return fmt.Errorf("HTTP GET returned nil response for %s", URL)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP status %d for %s", resp.StatusCode, URL)
	}
	filename := filepath.Base(URL)
	filePath := filepath.Join(dir, filename)
	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file %s: %w", filePath, err)
	}
	defer out.Close()
	n, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("error copying data for %s: %w", URL, err)
	}
	log.Printf("Downloaded %s: %d bytes", URL, n)
	return nil
}

func downloadFileWithTimeout(URL, dir string) error {
	log.Printf("Downloading %s", URL)
	sanitizedURL, err := sanitizeURL(URL)
	if err != nil {
		return fmt.Errorf("error sanitizing URL: %s", err)
	}
	if strings.HasPrefix(sanitizedURL, "http://") || strings.HasPrefix(sanitizedURL, "https://") {
		return downloadHTTPFile(sanitizedURL, dir)
	}
	return fmt.Errorf("unsupported protocol: %s", sanitizedURL)
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
	return urlStr
}

func sanitizeURL(urlStr string) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(u.Path, "//") {
		u.Path = strings.TrimPrefix(u.Path, "//")
	}
	u.Path = strings.ReplaceAll(u.Path, ":", "%3A")
	return u.String(), nil
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
