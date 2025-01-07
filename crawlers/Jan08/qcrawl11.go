package main

import (
	"crypto/tls"
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
	downloadTimeout = 90 * time.Second // Timeout for downloading PDFs
	maxRetries      = 5                // Maximum number of retries for failed downloads
	initialDelay    = 10 * time.Second  // Initial delay for retries
	maxDelay        = 50 * time.Second // Maximum delay for retries
)

var (
	delayedQueue = make(chan string, 10000) // Increased buffer size for delayed requests
	wg           sync.WaitGroup             // WaitGroup to wait for all downloads to complete
)

func main() {
	// Prompt for network interface
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Fatalf("Error listing network interfaces: %s", err)
	}

	fmt.Println("Available network interfaces:")
	for i, iface := range interfaces {
		fmt.Printf("%d: %s\n", i+1, iface.Name)
	}

	var selectedIndex int
	fmt.Println("Select the network interface to use (enter the number): ")
	fmt.Scanln(&selectedIndex)

	if selectedIndex < 1 || selectedIndex > len(interfaces) {
		log.Fatalf("Invalid network interface selection")
	}

	selectedInterface := interfaces[selectedIndex-1]

	// Set up HTTP transport to use the selected interface
	transport := &http.Transport{
		Dial: (&net.Dialer{
			LocalAddr: &net.TCPAddr{
				IP: getInterfaceIP(selectedInterface),
			},
		}).Dial,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Skip certificate verification
	}

	// Prompt for download directory
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
	fmt.Println("Select the directory to store downloaded PDFs (enter the number or hit enter for default): ")
	_, err = fmt.Scanln(&selectedDirIndex)

	var selectedDir string
	if err != nil || selectedDirIndex == 0 {
		// Create a new directory with timestamp
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

	// Prompt for starting URL
	var startingURL string
	fmt.Println("Enter the starting URL to crawl: ")
	fmt.Scanln(&startingURL)

	// Sanitize and validate the starting URL
	startingURL, err = sanitizeURL(startingURL)
	if err != nil {
		log.Fatalf("Error sanitizing starting URL: %s", err)
	}

	err = validateURL(startingURL)
	if err != nil {
		log.Fatalf("Error validating starting URL: %s", err)
	}

	c := colly.NewCollector()
	c.WithTransport(transport) // Set the transport directly on the collector

	// Create a request queue with 16 consumer threads (to utilize the 16-core CPU)
	q, err := queue.New(8, &queue.InMemoryQueueStorage{MaxSize: 10000})
	if err != nil {
		log.Fatalf("Error creating queue: %s", err)
	}

	c.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL.String())
	})

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		absoluteURL := e.Request.AbsoluteURL(link)
		log.Printf("Found link: %s", absoluteURL)
		q.AddURL(absoluteURL)
	})

	c.OnHTML("a[href$='.pdf']", func(e *colly.HTMLElement) {
		pdfURL := e.Request.AbsoluteURL(e.Attr("href"))
		log.Printf("Found PDF URL: %s", pdfURL)
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := downloadFileWithRetry(pdfURL, selectedDir, maxRetries, initialDelay)
			if err != nil {
				log.Printf("Error downloading file: %s", err)
			}
		}()
	})

	log.Printf("Adding starting URL to queue: %s", startingURL)
	q.AddURL(startingURL)
	go processDelayedQueue(selectedDir)
	log.Println("Starting the crawler...")
	q.Run(c)
	wg.Wait()
	log.Println("Crawler finished.")
}

func downloadFileWithRetry(URL, dir string, maxRetries int, initialDelay time.Duration) error {
	for i := 0; i < maxRetries; i++ {
		err := downloadFileWithTimeout(URL, dir)
		if err == nil {
			return nil
		}
		log.Printf("Attempt %d failed for URL %s: %s", i+1, URL, err)
		delay := time.Duration(int(initialDelay) * (1 << uint(i))) // Exponential backoff
		if delay > maxDelay {
			delay = maxDelay
		}
		time.Sleep(delay)
	}
	return fmt.Errorf("failed to download file after %d retries", maxRetries)
}

func downloadFileWithTimeout(URL, dir string) error {
	log.Printf("Downloading file from URL: %s", URL)
	u, err := url.Parse(URL)
	if err != nil {
		return err
	}

	switch u.Scheme {
	case "http", "https":
		return downloadHTTPFileWithTimeout(URL, dir)
	case "ftp":
		return downloadFTPFile(URL, dir)
	default:
		return fmt.Errorf("unsupported protocol scheme: %s", u.Scheme)
	}
}

func downloadHTTPFileWithTimeout(URL, dir string) error {
	log.Printf("Downloading file from URL: %s", URL)
	client := &http.Client{
		Timeout: downloadTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Skip certificate verification
		},
	}

	// Create a new request
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set a custom User-Agent header
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		// Categorize the error
		if urlErr, ok := err.(*url.Error); ok {
			switch {
			case urlErr.Timeout():
				log.Printf("Request timed out for URL %s: %s", URL, err)
			case strings.Contains(urlErr.Error(), "no such host"):
				log.Printf("DNS error for URL %s: %s", URL, err)
				return fmt.Errorf("DNS error: %w", err) // Skip retries for DNS errors
			case strings.Contains(urlErr.Error(), "connection refused"):
				log.Printf("Connection refused for URL %s: %s", URL, err)
			case strings.Contains(urlErr.Error(), "network is unreachable"):
				log.Printf("Network unreachable for URL %s: %s", URL, err)
			default:
				log.Printf("HTTP request failed for URL %s: %s", URL, err)
				return fmt.Errorf("HTTP request failed: %w", err)
			}
		} else {
			log.Printf("Unexpected error for URL %s: %s", URL, err)
			return fmt.Errorf("unexpected error: %w", err)
		}

		// Retry with timeout
		select {
		case delayedQueue <- URL:
			log.Printf("Added URL to retry queue: %s", URL)
		case <-time.After(1 * time.Second): // Timeout after 1 second
			log.Printf("Failed to add URL to retry queue (channel full): %s", URL)
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Unexpected status code: %d for URL: %s", resp.StatusCode, URL)
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
			return fmt.Errorf("file not accessible (status code: %d)", resp.StatusCode) // Skip retries for 404 and 403 errors
		}

		// Retry with timeout
		select {
		case delayedQueue <- URL:
			log.Printf("Added URL to retry queue: %s", URL)
		case <-time.After(1 * time.Second): // Timeout after 1 second
			log.Printf("Failed to add URL to retry queue (channel full): %s", URL)
		}
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	segments := strings.Split(URL, "/")
	fileName := segments[len(segments)-1]
	filePath := filepath.Join(dir, fileName)

	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.Printf("Successfully downloaded file: %s", filePath)
	return nil
}

func downloadFTPFile(URL, dir string) error {
	u, err := url.Parse(URL)
	if err != nil {
		return err
	}

	segments := strings.Split(u.Path, "/")
	fileName := segments[len(segments)-1]
	filePath := filepath.Join(dir, fileName)

	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	conn, err := net.DialTimeout("tcp", u.Host, downloadTimeout)
	if err != nil {
		log.Printf("FTP connection failed for URL %s: %s", URL, err)

		// Retry with timeout
		select {
		case delayedQueue <- URL:
			log.Printf("Added URL to retry queue: %s", URL)
		case <-time.After(1 * time.Second): // Timeout after 1 second
			log.Printf("Failed to add URL to retry queue (channel full): %s", URL)
		}
		return err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(fmt.Sprintf("USER anonymous\r\n")))
	if err != nil {
		return fmt.Errorf("FTP USER command failed: %w", err)
	}
	_, err = conn.Write([]byte(fmt.Sprintf("PASS \r\n")))
	if err != nil {
		return fmt.Errorf("FTP PASS command failed: %w", err)
	}
	_, err = conn.Write([]byte(fmt.Sprintf("RETR %s\r\n", u.Path)))
	if err != nil {
		return fmt.Errorf("FTP RETR command failed: %w", err)
	}

	_, err = io.Copy(out, conn)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.Printf("Successfully downloaded file: %s", filePath)
	return nil
}

func processDelayedQueue(selectedDir string) {
	for {
		select {
		case url := <-delayedQueue:
			log.Printf("Retrying download for URL: %s", url)
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := downloadFileWithRetry(url, selectedDir, maxRetries, initialDelay)
				if err != nil {
					log.Printf("Error retrying download for URL %s: %s", url, err)
				}
			}()
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

func sanitizeURL(urlStr string) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	// Encode the path to handle invalid characters
	u.Path = strings.ReplaceAll(u.Path, " ", "%20") // Replace spaces with %20
	u.Path = strings.ReplaceAll(u.Path, ":", "%3A") // Replace colons with %3A

	log.Printf("Sanitized URL: %s", u.String())
	return u.String(), nil
}

func validateURL(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return err
	}

	if u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "ftp" {
		return fmt.Errorf("unsupported protocol scheme: %s", u.Scheme)
	}

	if u.Host == "" {
		return fmt.Errorf("missing host in URL: %s", urlStr)
	}

	return nil
}
