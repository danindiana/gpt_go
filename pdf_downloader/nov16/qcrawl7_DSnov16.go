package main

import (
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
	visitedFilePath = "visitedURLs.txt"
)

var (
	excludedDomains = []string{
		"facebook.com", "youtube.com", "reddit.com", "linkedin.com", 
		"wikipedia.org", "twitter.com", "pubchem.ncbi.nlm.nih.gov", "ncbi.nlm.nih.gov",
	}
	visitedURLsMap = &sync.Map{}
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

	// Load visited URLs
	loadVisitedURLs()

	// Prompt for starting URL
	var startingURL string
	fmt.Println("Enter the starting URL to crawl: ")
	fmt.Scanln(&startingURL)

	// Lazy evaluation of the initial URL
	startingURL = normalizeURL(startingURL)

	c := colly.NewCollector()
	c.WithTransport(transport) // Set the transport directly on the collector

	// Create a request queue with 16 consumer threads (to utilize the 16-core CPU)
	q, err := queue.New(16, &queue.InMemoryQueueStorage{MaxSize: 10000})
	if err != nil {
		log.Fatalf("Error creating queue: %s", err)
	}

	c.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL.String())
	})

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")

		u, err := url.Parse(link)
		if err != nil {
			log.Printf("Error parsing URL %s: %s", link, err)
			return
		}

		if isExcludedDomain(u.Host) {
			return
		}

		absoluteURL := e.Request.AbsoluteURL(link)
		if !hasVisited(absoluteURL) {
			saveVisitedURL(absoluteURL)
			q.AddURL(absoluteURL)
		}
	})

	c.OnHTML("a[href$='.pdf']", func(e *colly.HTMLElement) {
		pdfURL := e.Request.AbsoluteURL(e.Attr("href"))
		err := downloadFile(pdfURL, selectedDir)
		if err != nil {
			log.Printf("Error downloading file: %s", err)
		}
	})

	q.AddURL(startingURL)
	q.Run(c)
}

func loadVisitedURLs() {
	data, err := os.ReadFile(visitedFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error reading visited URLs file: %s", err)
		}
		return
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line != "" {
			visitedURLsMap.Store(line, true)
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
	if err != nil {
		log.Printf("Error saving URL to file: %s", err)
		return
	}
	defer f.Close()

	_, err = f.WriteString(url + "\n")
	if err != nil {
		log.Printf("Error writing to file: %s", err)
	}
}

func downloadFile(URL, dir string) error {
	resp, err := http.Get(URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	segments := strings.Split(URL, "/")
	fileName := segments[len(segments)-1]
	filePath := filepath.Join(dir, fileName)

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func isExcludedDomain(host string) bool {
	for _, domain := range excludedDomains {
		if host == domain {
			return true
		}
	}
	return false
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
