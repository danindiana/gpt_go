To build the Go program `q_crawl.go` using the Bash command line on Ubuntu 22.04, you need to follow these steps:

### Step 1: Install Go

If Go is not already installed on your system, you need to install it. You can do this using the following commands:

```bash
sudo apt update
sudo apt install golang
```

### Step 2: Set Up Go Environment

Ensure that your Go environment variables are set correctly. You can add the following lines to your `~/.bashrc` or `~/.profile` file:

```bash
export GOPATH=$HOME/go
export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin
```

Then, reload the configuration:

```bash
source ~/.bashrc
```

### Step 3: Create the Go Module

Navigate to the directory containing your `q_crawl.go` file and initialize a new Go module:

```bash
cd /path/to/your/project
go mod init q_crawl
```

### Step 4: Add Dependencies

Your program uses external packages (`github.com/gocolly/colly`, `github.com/gocolly/colly/queue`, and `golang.org/x/time/rate`). You need to add these dependencies to your module:

```bash
go get github.com/gocolly/colly
go get github.com/gocolly/colly/queue
go get golang.org/x/time/rate
```

### Step 5: Build the Program

Now, you can build your program using the `go build` command:

```bash
go build q_crawl.go
```

This will generate an executable file named `q_crawl` (or `q_crawl.exe` on Windows) in the same directory.

### Step 6: Run the Program

You can run the compiled program using:

```bash
./q_crawl
```

### Summary of Commands

Here is a summary of the commands you need to run:

```bash
# Update package list and install Go
sudo apt update
sudo apt install golang

# Set up Go environment variables
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin' >> ~/.bashrc
source ~/.bashrc

# Navigate to your project directory
cd /path/to/your/project

# Initialize a new Go module
go mod init q_crawl

# Add dependencies
go get github.com/gocolly/colly
go get github.com/gocolly/colly/queue
go get golang.org/x/time/rate

# Build the program
go build q_crawl.go

# Run the program
./q_crawl
```

By following these steps, you should be able to build and run your Go program `q_crawl.go` on Ubuntu 22.04 using the Bash command line.



Let's go through the provided Go program step by step to understand how it functions:

1. **Imports and Configuration Struct**:
   - The program imports various packages, including `colly` for web scraping, `queue` for managing requests, and `rate` for rate limiting.
   - The `Configuration` struct holds settings for the crawler, such as the starting URL, file type to download, rate limit, excluded domains, maximum depth for crawling, and the file to store visited URLs.

2. **Global Variables**:
   - `visitedURLsMap`: A map to keep track of visited URLs.
   - `mapMutex`: A mutex to synchronize access to the map.
   - `config`: An instance of the `Configuration` struct to hold the current configuration.

3. **Main Function**:
   - **Initialize Configuration**: Sets default values for the `config` struct.
   - **Load Visited URLs**: Loads previously visited URLs from a file into the `visitedURLsMap`.
   - **Get User Input**: Prompts the user for the starting URL, file type to download, and rate limit.
   - **Initialize Rate Limiter**: Sets up a rate limiter to control the rate of requests.
   - **Initialize Collector**: Creates a new Colly collector with asynchronous mode enabled.
   - **Initialize Queue**: Creates a new queue to manage URLs to be crawled.
   - **Set Up Handlers**:
     - **OnRequest**: Limits the rate of requests and prints the URL being visited.
     - **OnHTML for Links**: Handles links found on pages, processes them, and adds them to the queue if not visited.
     - **OnHTML for File Download**: Handles downloading files of the specified type.
   - **Start Crawling**:
     - Creates an initial context with depth set to 1.
     - Adds the initial request to the queue.
   - **Run Queue**: Runs the queue and processes requests.
   - **Wait for Collector**: Waits for all requests to be processed before finishing.

4. **Functions**:
   - **getUserInput**: Prompts the user for input and sets the configuration values.
   - **handleLink**: Processes discovered links and adds them to the queue if they have not been visited. Also, handles depth tracking.
   - **handleFileDownload**: Downloads files of the specified type.
   - **loadVisitedURLs**: Loads visited URLs from a file into the `visitedURLsMap`.
   - **hasVisited**: Checks if a URL has been visited by looking it up in the `visitedURLsMap`.
   - **saveVisitedURL**: Saves a visited URL to the `visitedURLsMap` and appends it to the file.
   - **downloadFile**: Downloads a file from the specified URL.

### Potential Causes for Issues and Suggested Fixes:

#### Depth Tracking:
The depth is tracked using a context variable (`depth`) in the `handleLink` function. However, the initial request to the starting URL does not set this context variable correctly, which might cause issues with depth tracking.

**Fix**: Ensure the initial request to the starting URL includes the depth context.

#### Context Propagation:
The context with the depth information is created using `colly.NewContext()`, but it is not clear if this context is correctly propagated to the subsequent requests.

**Fix**: Ensure that the context with the depth information is correctly propagated to the subsequent requests.

#### Queue Management:
The queue is used to manage the URLs to be crawled, but the way the depth is incremented and passed to the next request might not be correctly handled.

**Fix**: Ensure that the queue correctly handles the depth context when adding new URLs.

### Revised Version with Fixes:

```go
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

	// Create the initial context with depth
	initialCtx := colly.NewContext()
	initialCtx.Put("depth", "1")

	// Start crawling from the starting URL with initial depth 1
	c.Request("GET", config.StartingURL, nil, initialCtx, nil)

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
```

### Key Adjustments:
1. **Initial Request Context**:
   - Correctly added depth context to the initial request.
   - Started crawling by directly adding the initial request to the collector.
2. **Context Propagation**:
   - Ensured depth context is correctly propagated in subsequent requests.

### Running the Program:
- Use `go run q_crawlerv4.go` to run the script.
- Monitor the debug outputs to ensure it visits multiple URLs and properly recurses beyond the first depth level.

This should address the depth tracking issues and ensure the crawler correctly recurses through the discovered links.
