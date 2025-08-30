package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
	"golang.org/x/time/rate"
)

// MULTI-NIC BEAST MODE CONFIGURATION
const (
	maxDepth          = 13                     // Max crawl depth
	requestTimeout    = 60 * time.Second       // Longer timeout for large files
	concurrentWorkers = 256                    // 8x your core count for crawling
	politeDelay       = 10 * time.Millisecond  // INSANELY aggressive crawling
	userAgent         = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0 Safari/537.36"
	
	// MULTI-NIC download configuration
	initialDownloadWorkers = 1000              // Start with 1000 workers!
	maxDownloadWorkers     = 8000              // Scale up to 8000 concurrent downloads!
	queueGrowthThreshold   = 0.4               // Scale at 40% full
	scaleCheckInterval     = 500 * time.Millisecond // Check twice per second
	scaleUpAmount          = 300               // Add 300 workers at a time
	maxQueueSize           = 5000000           // 5 MILLION item queue!
	
	// Multi-NIC network beast mode
	maxConnectionsTotal    = 20000             // 20K total connections across all NICs
	maxConnectionsPerHost  = 2000              // 2K per host
	connectionTimeout      = 3 * time.Second   // Ultra-fast connection establishment
	keepAliveTimeout       = 600 * time.Second // 10-minute keep-alive
	
	// Hardware-optimized settings
	downloadBufferSize     = 32 * 1024 * 1024  // 32MB buffer for 10GbE!
	maxRetries            = 3                  // Fewer retries for speed
	retryBackoff          = 200 * time.Millisecond // Very fast retry
	
	// Memory settings for your beast
	targetMemoryUsageGB    = 100               // Use up to 100GB of your 128GB
	gcTargetPercent        = 500               // Even less frequent GC
)

// Network interface configuration
type NetworkInterface struct {
	Name        string
	IP          string
	IsActive    bool
	Speed       string
	WorkerCount int
	Clients     []*http.Client
}

// Global variables
var (
	visitedURLsMap   = make(map[string]bool)
	downloadedFiles  = make(map[string]bool)
	pendingDownloads = make(map[string]bool)
	failedDownloads  = make(map[string]int)
	mapMutex         = &sync.RWMutex{}
	targetDir        string
	logFilePath      string
	downloadLogPath  string
	firstRequestOnce sync.Once
	startURL         string

	// Multi-NIC system
	networkInterfaces []NetworkInterface
	downloadQueues    []chan downloadTask      // One queue per interface
	priorityQueue     = make(chan downloadTask, 50000)
	downloadLimiter   *rate.Limiter
	downloadWG        sync.WaitGroup
	activeWorkers     int64
	shutdownChan      = make(chan struct{})
	scalerWG          sync.WaitGroup
	
	// Performance counters
	stats struct {
		downloadAttempts int64
		downloadSuccess  int64
		downloadFailed   int64
		bytesDownloaded  int64
		startTime        time.Time
	}
	
	// Interface load balancing
	currentInterfaceIndex int64
)

type downloadTask struct {
	url         string
	depth       int
	retry       int
	priority    bool
	interfaceID int // Which network interface to use
}

func main() {
	stats.startTime = time.Now()
	
	// BEAST MODE SYSTEM CONFIGURATION
	setupBeastMode()
	
	fmt.Printf("🔥🔥🔥 MULTI-NIC BEAST MODE ACTIVATED! 🔥🔥🔥\n")
	fmt.Printf("🖥️ System: AMD Ryzen 9 5950X (%d cores) with 128GB RAM\n", runtime.NumCPU())
	fmt.Printf("⚡ GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))
	fmt.Printf("💾 Memory target: %dGB\n", targetMemoryUsageGB)
	
	// Detect and configure network interfaces
	err := detectNetworkInterfaces()
	if err != nil {
		fmt.Printf("❌ Failed to detect network interfaces: %v\n", err)
		return
	}
	
	// Let user select which interfaces to use
	selectedInterfaces := selectNetworkInterfaces()
	if len(selectedInterfaces) == 0 {
		fmt.Println("❌ No network interfaces selected")
		return
	}
	
	// Configure selected interfaces
	err = configureSelectedInterfaces(selectedInterfaces)
	if err != nil {
		fmt.Printf("❌ Failed to configure interfaces: %v\n", err)
		return
	}
	
	// Increase system limits
	increaseFileDescriptorLimit()
	optimizeNetworkSettings()
	
	// Get user input
	fmt.Println("\nEnter the starting URL to crawl:")
	fmt.Scanln(&startURL)

	fmt.Println("Enter the target directory to save files:")
	fmt.Scanln(&targetDir)

	// URL validation
	parsedStart, err := url.Parse(startURL)
	if err != nil || parsedStart.Scheme == "" || parsedStart.Host == "" {
		fmt.Printf("❌ Invalid URL: %s\n", startURL)
		return
	}
	if parsedStart.Scheme != "http" && parsedStart.Scheme != "https" {
		parsedStart.Scheme = "https"
		startURL = parsedStart.String()
	}

	// Create target directory
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		fmt.Printf("📁 Creating directory: %s\n", targetDir)
		err = os.MkdirAll(targetDir, 0755)
		if err != nil {
			fmt.Printf("❌ Failed to create directory: %v\n", err)
			return
		}
	}

	// Initialize log files
	timestamp := time.Now().Format("20060102_150405")
	logFilePath = fmt.Sprintf("visitedURLs_%s.txt", timestamp)
	downloadLogPath = fmt.Sprintf("downloads_%s.txt", timestamp)

	// Initialize queues and HTTP clients for each interface
	initializeMultiNICSystem()

	// Ultra-permissive rate limiting
	downloadLimiter = rate.NewLimiter(rate.Every(10*time.Microsecond), maxDownloadWorkers*3)

	// Start massive number of workers distributed across interfaces
	startMultiNICWorkers()

	// Start multiple scalers and monitors
	for i := 0; i < 16; i++ { // 16 concurrent scalers for ultra-fast response
		scalerWG.Add(1)
		go downloadScaler()
	}

	scalerWG.Add(1)
	go performanceMonitor()
	
	scalerWG.Add(1)
	go memoryMonitor()
	
	scalerWG.Add(1)
	go networkMonitor()

	// Create ultra-aggressive collector
	c := createBeastCollector()
	
	// Setup crawling callbacks
	setupCrawlingCallbacks(c)

	// UNLEASH THE MULTI-NIC BEAST!
	printStartupInfo()

	err = c.Visit(startURL)
	if err != nil {
		fmt.Printf("❌ Failed to start crawl: %v\n", err)
		return
	}

	c.Wait()
	
	// Shutdown sequence
	close(shutdownChan)
	scalerWG.Wait()
	close(priorityQueue)
	for _, queue := range downloadQueues {
		close(queue)
	}
	downloadWG.Wait()

	printFinalStats()
}

// detectNetworkInterfaces discovers available network interfaces
func detectNetworkInterfaces() error {
    fmt.Println("\n🔍 Detecting network interfaces...")
    
    interfaces, err := net.Interfaces()
    if err != nil {
        return err
    }
    
    networkInterfaces = make([]NetworkInterface, 0)
    
    for _, iface := range interfaces {
        // Skip loopback and virtual interfaces, but keep tun for VPN
        if iface.Flags&net.FlagLoopback != 0 || 
           strings.Contains(iface.Name, "vir") ||
           strings.Contains(iface.Name, "docker") {
            continue
        }
        
        addrs, err := iface.Addrs()
        if err != nil || len(addrs) == 0 {
            continue
        }
        
        // Get first valid IP
        var ip string
        for _, addr := range addrs {
            if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
                if ipnet.IP.To4() != nil {
                    ip = ipnet.IP.String()
                    break
                }
            }
        }
        
        // Even if no IP is found, include the interface (like tun0 might not have a typical IP)
        // if it's active and we want to use it. For tun0, you might need to handle routing differently.
        if ip == "" && !(iface.Flags&net.FlagUp != 0) {
            continue // Only skip if no IP *and* not active
        }
        
        // Determine if interface is active and get speed info
        isActive := iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagRunning != 0
        speed := getInterfaceSpeed(iface.Name)
        
        networkInterfaces = append(networkInterfaces, NetworkInterface{
            Name:     iface.Name,
            IP:       ip, // Will be empty string if not found
            IsActive: isActive,
            Speed:    speed,
        })
        
        status := "DOWN"
        if isActive {
            status = "UP"
        }
        
        fmt.Printf("🌐 Found: %s (%s) - %s - %s\n", iface.Name, ip, status, speed)
    }
    
    return nil
}

// getInterfaceSpeed attempts to determine interface speed
func getInterfaceSpeed(ifname string) string {
	// Try to read speed from /sys
	speedPath := fmt.Sprintf("/sys/class/net/%s/speed", ifname)
	if data, err := os.ReadFile(speedPath); err == nil {
		if speed, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			if speed >= 10000 {
				return fmt.Sprintf("%dGbE", speed/1000)
			} else if speed >= 1000 {
				return fmt.Sprintf("%dGbE", speed/1000)
			} else {
				return fmt.Sprintf("%dMbE", speed)
			}
		}
	}
	
	// Fallback: guess based on interface name
	if strings.Contains(ifname, "enp3s0f") {
		return "10GbE" // X540-AT2
	} else if strings.Contains(ifname, "enp9s0") {
		return "1GbE"  // I211
	}
	
	return "Unknown"
}

// selectNetworkInterfaces lets user choose which interfaces to use
func selectNetworkInterfaces() []int {
	fmt.Println("\n🎯 Select network interfaces for crawling:")
	fmt.Println("Available interfaces:")
	
	activeCount := 0
	for i, iface := range networkInterfaces {
		status := "❌"
		if iface.IsActive {
			status = "✅"
			activeCount++
		}
		fmt.Printf("%d) %s %s (%s) - %s - %s\n", 
			i+1, status, iface.Name, iface.IP, iface.Speed, 
			map[bool]string{true: "ACTIVE", false: "INACTIVE"}[iface.IsActive])
	}
	
	if activeCount == 0 {
		fmt.Println("❌ No active interfaces found!")
		return nil
	}
	
	fmt.Printf("\nRecommendation: Use all active high-speed interfaces for maximum performance\n")
	fmt.Printf("Enter interface numbers (comma-separated, e.g., 1,2,3) or 'all' for all active: ")
	
	var input string
	fmt.Scanln(&input)
	
	if input == "all" {
		var selected []int
		for i, iface := range networkInterfaces {
			if iface.IsActive {
				selected = append(selected, i)
			}
		}
		return selected
	}
	
	// Parse comma-separated list
	parts := strings.Split(input, ",")
	var selected []int
	for _, part := range parts {
		if num, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
			if num > 0 && num <= len(networkInterfaces) {
				idx := num - 1
				if networkInterfaces[idx].IsActive {
					selected = append(selected, idx)
				} else {
					fmt.Printf("⚠️ Interface %s is not active, skipping\n", networkInterfaces[idx].Name)
				}
			}
		}
	}
	
	return selected
}

// configureSelectedInterfaces sets up the selected network interfaces
func configureSelectedInterfaces(selected []int) error {
	fmt.Println("\n⚙️ Configuring selected interfaces...")
	
	var activeInterfaces []NetworkInterface
	totalBandwidth := 0
	
	for _, idx := range selected {
		iface := networkInterfaces[idx]
		
		// Calculate worker distribution based on interface speed
		var workers int
		if strings.Contains(iface.Speed, "10G") {
			workers = 2000 // More workers for 10GbE
			totalBandwidth += 10000
		} else if strings.Contains(iface.Speed, "1G") {
			workers = 500  // Fewer workers for 1GbE
			totalBandwidth += 1000
		} else {
			workers = 200  // Conservative for unknown speed
			totalBandwidth += 100
		}
		
		iface.WorkerCount = workers
		activeInterfaces = append(activeInterfaces, iface)
		
		fmt.Printf("✅ %s (%s) - %s - %d workers\n", 
			iface.Name, iface.IP, iface.Speed, workers)
	}
	
	networkInterfaces = activeInterfaces
	
	fmt.Printf("🚀 Total bandwidth: %d Mbps across %d interfaces\n", 
		totalBandwidth, len(activeInterfaces))
	
	return nil
}

// initializeMultiNICSystem sets up queues and HTTP clients for each interface
func initializeMultiNICSystem() {
	fmt.Println("\n🔧 Initializing multi-NIC system...")
	
	downloadQueues = make([]chan downloadTask, len(networkInterfaces))
	
	for i, iface := range networkInterfaces {
		// Create queue for this interface
		queueSize := maxQueueSize / len(networkInterfaces)
		downloadQueues[i] = make(chan downloadTask, queueSize)
		
		// Create HTTP clients for this interface
		clientCount := 64 // 64 clients per interface
		clients := make([]*http.Client, clientCount)
		
		for j := 0; j < clientCount; j++ {
			clients[j] = createInterfaceClient(iface)
		}
		
		networkInterfaces[i].Clients = clients
		
		fmt.Printf("🌐 Interface %s: %d queue slots, %d HTTP clients\n", 
			iface.Name, queueSize, clientCount)
	}
}

// createInterfaceClient creates an HTTP client bound to a specific interface
func createInterfaceClient(iface NetworkInterface) *http.Client {
	// Create custom dialer that binds to specific interface
	localAddr, err := net.ResolveIPAddr("ip", iface.IP)
	if err != nil {
		fmt.Printf("⚠️ Warning: Could not resolve IP %s for %s\n", iface.IP, iface.Name)
		localAddr = nil
	}
	
	dialer := &net.Dialer{
		Timeout:   connectionTimeout,
		KeepAlive: keepAliveTimeout,
	}
	
	// Bind to local interface IP if possible
	if localAddr != nil {
		dialer.LocalAddr = &net.TCPAddr{IP: localAddr.IP}
	}
	
	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		MaxIdleConns:          maxConnectionsTotal / len(networkInterfaces) / 64,
		MaxIdleConnsPerHost:   maxConnectionsPerHost / len(networkInterfaces) / 64,
		MaxConnsPerHost:       maxConnectionsPerHost / len(networkInterfaces) / 64,
		IdleConnTimeout:       keepAliveTimeout,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    false,
		ForceAttemptHTTP2:     true,
	}
	
	// Enable TCP optimizations for high-bandwidth interfaces
	if strings.Contains(iface.Speed, "10G") {
		// Enable TCP window scaling and other optimizations
		// These would normally require root privileges to set via syscalls
	}
	
	return &http.Client{
		Timeout:   requestTimeout,
		Transport: transport,
	}
}

// startMultiNICWorkers starts workers distributed across interfaces
func startMultiNICWorkers() {
	fmt.Println("\n👥 Starting multi-NIC workers...")
	
	totalWorkers := 0
	for i, iface := range networkInterfaces {
		workers := min(iface.WorkerCount, initialDownloadWorkers/len(networkInterfaces)+100)
		for j := 0; j < workers; j++ {
			downloadWG.Add(1)
			go multiNICDownloadWorker(i, j%len(iface.Clients))
			atomic.AddInt64(&activeWorkers, 1)
			totalWorkers++
		}
		
		fmt.Printf("🚀 %s: Started %d workers\n", iface.Name, workers)
	}
	
	fmt.Printf("💪 Total workers started: %d\n", totalWorkers)
}

// multiNICDownloadWorker processes downloads on a specific network interface
func multiNICDownloadWorker(interfaceID, clientIndex int) {
	defer downloadWG.Done()
	defer atomic.AddInt64(&activeWorkers, -1)
	
	iface := networkInterfaces[interfaceID]
	client := iface.Clients[clientIndex]
	workerName := fmt.Sprintf("%s-W%d", iface.Name, clientIndex)
	
	for {
		var task downloadTask
		var ok bool
		
		// Check priority queue first
		select {
		case task, ok = <-priorityQueue:
			if !ok {
				// Priority queue closed
			} else {
				goto processTask
			}
		default:
			// Priority queue empty
		}
		
		// Check interface-specific queue
		select {
		case task, ok = <-downloadQueues[interfaceID]:
			if !ok {
				// Interface queue closed
				return
			}
		default:
			// No work available, sleep briefly
			time.Sleep(1 * time.Millisecond)
			continue
		}
		
	processTask:
		// Rate limiting
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		downloadLimiter.Wait(ctx)
		cancel()
		
		atomic.AddInt64(&stats.downloadAttempts, 1)
		
		err := downloadDocumentMultiNIC(task.url, client, workerName)
		if err != nil {
			atomic.AddInt64(&stats.downloadFailed, 1)
			
			if task.retry < maxRetries {
				task.retry++
				task.priority = true
				task.interfaceID = interfaceID
				
				go func(t downloadTask) {
					time.Sleep(retryBackoff * time.Duration(t.retry))
					select {
					case priorityQueue <- t:
						// Successfully re-queued
					default:
						markDownloadFailed(t.url)
					}
				}(task)
			} else {
				markDownloadFailed(task.url)
			}
		} else {
			atomic.AddInt64(&stats.downloadSuccess, 1)
			markDownloadCompleted(task.url)
		}
	}
}

// setupBeastMode configures system for maximum performance
func setupBeastMode() {
	runtime.GOMAXPROCS(runtime.NumCPU() * 4) // Even more OS threads for networking
	
	// Optimize GC for high throughput
	runtime.GC()
	debug := os.Getenv("GODEBUG")
	if debug == "" {
		os.Setenv("GODEBUG", fmt.Sprintf("gctrace=0,gcpacertarget=%d", gcTargetPercent))
	}
}

// increaseFileDescriptorLimit increases system limits
func increaseFileDescriptorLimit() {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		fmt.Printf("⚠️ Could not get file descriptor limit: %v\n", err)
		return
	}
	
	fmt.Printf("📁 Current FD limit: %d\n", rLimit.Cur)
	
	rLimit.Cur = rLimit.Max
	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		fmt.Printf("⚠️ Could not increase FD limit: %v\n", err)
	} else {
		fmt.Printf("📁 Increased FD limit to: %d\n", rLimit.Cur)
	}
}

// optimizeNetworkSettings attempts to optimize network settings
func optimizeNetworkSettings() {
	fmt.Println("🔧 Optimizing network settings...")
	
	// These would require root privileges, so we'll just report what should be done
	optimizations := []string{
		"net.core.rmem_max = 134217728",
		"net.core.wmem_max = 134217728", 
		"net.ipv4.tcp_rmem = 4096 87380 134217728",
		"net.ipv4.tcp_wmem = 4096 65536 134217728",
		"net.core.netdev_max_backlog = 30000",
		"net.core.netdev_budget = 600",
		"net.ipv4.tcp_congestion_control = bbr",
	}
	
	fmt.Println("💡 For optimal performance, run as root:")
	for _, opt := range optimizations {
		fmt.Printf("   sysctl -w %s\n", opt)
	}
}

// createBeastCollector creates an ultra-aggressive collector
func createBeastCollector() *colly.Collector {
	c := colly.NewCollector(
		colly.UserAgent(userAgent),
		colly.Async(true),
		colly.IgnoreRobotsTxt(),
	)

	extensions.RandomUserAgent(c)
	extensions.Referer(c)
	c.SetRequestTimeout(requestTimeout)

	err := c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: concurrentWorkers,
		Delay:       politeDelay,
		RandomDelay: 5 * time.Millisecond,
	})
	
	if err != nil {
		fmt.Printf("❌ Failed to set crawl limits: %v\n", err)
	}

	cacheDir := ".colly_cache"
	os.RemoveAll(cacheDir)
	c.CacheDir = cacheDir

	return c
}

// setupCrawlingCallbacks configures the crawler callbacks
func setupCrawlingCallbacks(c *colly.Collector) {
	c.OnRequest(func(r *colly.Request) {
		if r.URL.String() == startURL {
			firstRequestOnce.Do(func() {
				ctx := colly.NewContext()
				ctx.Put("depth", "0")
				r.Ctx = ctx
				fmt.Printf("🚀 [0] Multi-NIC crawl started: %s\n", r.URL)
			})
		}
	})

	c.OnResponse(func(r *colly.Response) {
		// Minimal logging for performance
		if atomic.LoadInt64(&stats.downloadAttempts) < 50 {
			depth := 0
			if d := r.Ctx.Get("depth"); d != "" {
				fmt.Sscanf(d, "%d", &depth)
			}
			if depth <= 1 {
				fmt.Printf("✅ [%d] Response %d: %s\n", depth, r.StatusCode, r.Request.URL)
			}
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		if atomic.LoadInt64(&stats.downloadFailed) < 20 {
			fmt.Printf("❌ Crawl error: %v\n", err)
		}
	})

	// Link discovery
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		absURL := e.Request.AbsoluteURL(href)

		parsed, err := url.Parse(absURL)
		if err != nil || parsed.Host == "" {
			return
		}

		currentDepth := 0
		if d := e.Request.Ctx.Get("depth"); d != "" {
			fmt.Sscanf(d, "%d", &currentDepth)
		}

		if currentDepth >= maxDepth {
			return
		}

		cleanURL := normalizeParsedURL(parsed)
		if hasVisited(cleanURL) {
			return
		}

		saveVisitedURL(cleanURL)

		newCtx := colly.NewContext()
		newCtx.Put("depth", fmt.Sprintf("%d", currentDepth+1))

		e.Request.Visit(absURL)
	})

	// Document detection and queuing
	docExtensions := []string{
		".pdf", 
	}
	
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		docURL := e.Request.AbsoluteURL(href)
		
		if !isDocumentURL(docURL, docExtensions) {
			return
		}

		depth := 0
		if d := e.Request.Ctx.Get("depth"); d != "" {
			fmt.Sscanf(d, "%d", &depth)
		}
		
		if isDownloadedOrPending(docURL) {
			return
		}

		// Load-balanced interface selection
		interfaceID := int(atomic.AddInt64(&currentInterfaceIndex, 1)) % len(networkInterfaces)
		
		task := downloadTask{
			url:         docURL, 
			depth:       depth, 
			retry:       0, 
			priority:    false,
			interfaceID: interfaceID,
		}
		
		// Try interface-specific queue
		select {
		case downloadQueues[interfaceID] <- task:
			markPendingDownload(docURL)
		default:
			// Queue full, try priority queue
			select {
			case priorityQueue <- task:
				markPendingDownload(docURL)
			default:
				// Both queues full - force scaling
				go forceScaleUp()
				go persistentEnqueue(task)
			}
		}
	})
}

// Network and performance monitoring functions
func networkMonitor() {
	defer scalerWG.Done()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-shutdownChan:
			return
		case <-ticker.C:
			printNetworkStats()
		}
	}
}

func printNetworkStats() {
	fmt.Printf("🌐 Network Status:\n")
	for i, iface := range networkInterfaces {
		queueLen := len(downloadQueues[i])
		queueCap := cap(downloadQueues[i])
		utilization := float64(queueLen) / float64(queueCap) * 100
		
		fmt.Printf("   %s (%s): Queue %d/%d (%.1f%%), %d clients\n", 
			iface.Name, iface.Speed, queueLen, queueCap, utilization, len(iface.Clients))
	}
}

func printStartupInfo() {
	fmt.Printf("\n🔥🔥🔥 MULTI-NIC BEAST UNLEASHED! 🔥🔥🔥\n")
	fmt.Printf("🎯 Target: %s (max depth %d)\n", startURL, maxDepth)
	fmt.Printf("📁 Output: %s\n", targetDir)
	fmt.Printf("👥 Workers: %d initial → %d max\n", initialDownloadWorkers, maxDownloadWorkers)
	fmt.Printf("🌐 Interfaces: %d active\n", len(networkInterfaces))
	for _, iface := range networkInterfaces {
		fmt.Printf("   • %s (%s) - %s - %d workers\n", 
			iface.Name, iface.IP, iface.Speed, iface.WorkerCount)
	}
	fmt.Printf("⚡ Crawl delay: %v (INSANE MODE)\n", politeDelay)
	fmt.Printf("💾 Buffer size: %dMB per download\n", downloadBufferSize/1024/1024)
	fmt.Printf("📦 Total queue capacity: %d items\n\n", maxQueueSize)
}

// Continue with remaining functions...
func downloadScaler() {
	defer scalerWG.Done()
	ticker := time.NewTicker(scaleCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-shutdownChan:
			return
		case <-ticker.C:
			checkAndScaleMultiNIC()
		}
	}
}

func checkAndScaleMultiNIC() {
	totalQueued := len(priorityQueue)
	totalCapacity := cap(priorityQueue)
	
	for _, queue := range downloadQueues {
		totalQueued += len(queue)
		totalCapacity += cap(queue)
	}
	
	utilization := float64(totalQueued) / float64(totalCapacity)
	currentWorkers := atomic.LoadInt64(&activeWorkers)
	
	if utilization > queueGrowthThreshold && currentWorkers < maxDownloadWorkers {
		// Distribute new workers across interfaces
		scaleAmount := scaleUpAmount
		if utilization > 0.8 {
			scaleAmount = scaleUpAmount * 4 // Quad scaling when critically full
		} else if utilization > 0.6 {
			scaleAmount = scaleUpAmount * 2 // Double scaling when very full
		}
		
		newWorkersTotal := min(scaleAmount, maxDownloadWorkers-int(currentWorkers))
		if newWorkersTotal > 0 {
			// Distribute workers across interfaces based on their capacity
			workersPerInterface := newWorkersTotal / len(networkInterfaces)
			remainder := newWorkersTotal % len(networkInterfaces)
			
			for i, iface := range networkInterfaces {
				workers := workersPerInterface
				if i < remainder {
					workers++ // Distribute remainder
				}
				
				for j := 0; j < workers; j++ {
					downloadWG.Add(1)
					go multiNICDownloadWorker(i, j%len(iface.Clients))
					atomic.AddInt64(&activeWorkers, 1)
				}
			}
			
			fmt.Printf("📈 Multi-NIC scaled: +%d workers across %d interfaces (util: %.1f%%)\n", 
				newWorkersTotal, len(networkInterfaces), utilization*100)
		}
	}
}

func forceScaleUp() {
	currentWorkers := atomic.LoadInt64(&activeWorkers)
	if currentWorkers < maxDownloadWorkers {
		newWorkersTotal := min(scaleUpAmount*3, maxDownloadWorkers-int(currentWorkers))
		workersPerInterface := newWorkersTotal / len(networkInterfaces)
		
		for i, iface := range networkInterfaces {
			for j := 0; j < workersPerInterface; j++ {
				downloadWG.Add(1)
				go multiNICDownloadWorker(i, j%len(iface.Clients))
				atomic.AddInt64(&activeWorkers, 1)
			}
		}
		
		fmt.Printf("🚀 EMERGENCY Multi-NIC scale: +%d workers (now %d)\n", 
			newWorkersTotal, currentWorkers+int64(newWorkersTotal))
	}
}

func persistentEnqueue(task downloadTask) {
	maxAttempts := 50
	for attempt := 0; attempt < maxAttempts; attempt++ {
		time.Sleep(time.Duration(attempt*50) * time.Millisecond)
		
		// Try priority queue first
		select {
		case priorityQueue <- task:
			markPendingDownload(task.url)
			return
		default:
			// Try interface-specific queues
			for i := range downloadQueues {
				select {
				case downloadQueues[i] <- task:
					markPendingDownload(task.url)
					return
				default:
					continue
				}
			}
		}
	}
	fmt.Printf("❌ [%d] Multi-NIC dropped after %d attempts: %s\n", task.depth, maxAttempts, task.url)
}

func performanceMonitor() {
	defer scalerWG.Done()
	ticker := time.NewTicker(3 * time.Second) // Very frequent updates
	defer ticker.Stop()
	
	for {
		select {
		case <-shutdownChan:
			return
		case <-ticker.C:
			printPerformanceStats()
		}
	}
}

func printPerformanceStats() {
	attempts := atomic.LoadInt64(&stats.downloadAttempts)
	success := atomic.LoadInt64(&stats.downloadSuccess)
	failed := atomic.LoadInt64(&stats.downloadFailed)
	bytes := atomic.LoadInt64(&stats.bytesDownloaded)
	elapsed := time.Since(stats.startTime)
	workers := atomic.LoadInt64(&activeWorkers)
	
	totalQueued := len(priorityQueue)
	for _, queue := range downloadQueues {
		totalQueued += len(queue)
	}
	
	if attempts > 0 {
		successRate := float64(success) / float64(attempts) * 100
		throughput := float64(success) / elapsed.Seconds()
		mbps := float64(bytes) * 8 / elapsed.Seconds() / 1024 / 1024 // Mbps
		
		fmt.Printf("🔥 MULTI-NIC: %d workers, %d queued | %d attempts, %d success, %d failed (%.1f%%) | %.1f dl/s, %.1f Mbps | %s\n",
			workers, totalQueued, attempts, success, failed, successRate, throughput, mbps, formatBytes(bytes))
	}
}

func memoryMonitor() {
	defer scalerWG.Done()
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-shutdownChan:
			return
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			
			allocGB := float64(m.Alloc) / 1024 / 1024 / 1024
			sysGB := float64(m.Sys) / 1024 / 1024 / 1024
			
			fmt.Printf("🧠 Memory: %.1fGB allocated, %.1fGB system (target: %dGB), GC: %d\n", 
				allocGB, sysGB, targetMemoryUsageGB, m.NumGC)
			
			if allocGB > float64(targetMemoryUsageGB)*0.95 {
				fmt.Printf("🧹 Triggering GC (approaching %dGB limit)\n", targetMemoryUsageGB)
				runtime.GC()
			}
		}
	}
}

func printFinalStats() {
	attempts := atomic.LoadInt64(&stats.downloadAttempts)
	success := atomic.LoadInt64(&stats.downloadSuccess)
	failed := atomic.LoadInt64(&stats.downloadFailed)
	bytes := atomic.LoadInt64(&stats.bytesDownloaded)
	elapsed := time.Since(stats.startTime)
	
	fmt.Printf("\n🔥🔥🔥 MULTI-NIC BEAST MODE COMPLETE! 🔥🔥🔥\n")
	fmt.Printf("⏱️ Total time: %v\n", elapsed)
	fmt.Printf("📊 Downloads: %d attempts, %d success, %d failed\n", attempts, success, failed)
	fmt.Printf("💾 Data downloaded: %s\n", formatBytes(bytes))
	fmt.Printf("⚡ Average throughput: %.2f downloads/sec\n", float64(success)/elapsed.Seconds())
	fmt.Printf("🌐 Average bandwidth: %.2f Mbps\n", float64(bytes)*8/elapsed.Seconds()/1024/1024)
	fmt.Printf("💪 Peak workers: %d across %d interfaces\n", atomic.LoadInt64(&activeWorkers), len(networkInterfaces))
	fmt.Printf("🧠 Final memory: %s\n", formatMemory(getMemStats()))
	
	fmt.Printf("\n🌐 Per-Interface Stats:\n")
	for _, iface := range networkInterfaces {
		fmt.Printf("   %s (%s): %s - %d workers configured\n", 
			iface.Name, iface.IP, iface.Speed, iface.WorkerCount)
	}
}

func downloadDocumentMultiNIC(docURL string, client *http.Client, workerName string) error {
	req, err := http.NewRequestWithContext(context.Background(), "GET", docURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	filename := extractFilename(docURL, resp.Header)
	path := filepath.Join(targetDir, filename)

	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	// Use massive buffer optimized for 10GbE
	buf := make([]byte, downloadBufferSize)
	written, err := io.CopyBuffer(out, resp.Body, buf)
	
	if err == nil {
		atomic.AddInt64(&stats.bytesDownloaded, written)
	}
	
	return err
}

// Utility functions
func isDocumentURL(docURL string, extensions []string) bool {
	lowerURL := strings.ToLower(docURL)
	for _, ext := range extensions {
		if strings.HasSuffix(lowerURL, ext) || 
		   strings.Contains(lowerURL, ext+"?") ||
		   strings.Contains(lowerURL, ext+"&") {
			return true
		}
	}
	return false
}

func normalizeParsedURL(u *url.URL) string {
	u.Fragment = ""
	u.RawQuery = ""
	return strings.ToLower(u.String())
}

func hasVisited(url string) bool {
	mapMutex.RLock()
	defer mapMutex.RUnlock()
	_, exists := visitedURLsMap[url]
	return exists
}

func isDownloadedOrPending(url string) bool {
	mapMutex.RLock()
	defer mapMutex.RUnlock()
	_, downloaded := downloadedFiles[url]
	_, pending := pendingDownloads[url]
	return downloaded || pending
}

func markPendingDownload(url string) {
	mapMutex.Lock()
	pendingDownloads[url] = true
	mapMutex.Unlock()
}

func markDownloadCompleted(url string) {
	mapMutex.Lock()
	delete(pendingDownloads, url)
	downloadedFiles[url] = true
	mapMutex.Unlock()
	
	// Async logging to avoid blocking worker
	go func() {
		f, err := os.OpenFile(downloadLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return
		}
		defer f.Close()
		f.WriteString(url + "\n")
	}()
}

func markDownloadFailed(url string) {
	mapMutex.Lock()
	delete(pendingDownloads, url)
	failedDownloads[url]++
	mapMutex.Unlock()
}

func saveVisitedURL(url string) {
	mapMutex.Lock()
	visitedURLsMap[url] = true
	mapMutex.Unlock()

	// Async logging for performance
	go func() {
		f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return
		}
		defer f.Close()
		f.WriteString(url + "\n")
	}()
}

func extractFilename(docURL string, headers http.Header) string {
	if cd := headers.Get("Content-Disposition"); cd != "" {
		if strings.HasPrefix(cd, "attachment; filename=") {
			filename := strings.TrimPrefix(cd, "attachment; filename=")
			filename = strings.Trim(filename, `"`)
			if filename != "" {
				return sanitizeFilename(filename)
			}
		}
	}

	segments := strings.Split(docURL, "/")
	filename := segments[len(segments)-1]
	
	if idx := strings.Index(filename, "?"); idx != -1 {
		filename = filename[:idx]
	}
	
	if filename == "" || !strings.Contains(filename, ".") {
		filename = fmt.Sprintf("download_%d", time.Now().UnixNano())
	}
	
	return sanitizeFilename(filename)
}

func sanitizeFilename(name string) string {
	for _, ch := range []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|", "\x00"} {
		name = strings.ReplaceAll(name, ch, "_")
	}
	if len(name) > 200 {
		ext := filepath.Ext(name)
		name = name[:200-len(ext)] + ext
	}
	return name
}

func getMemStats() runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}

func formatMemory(m runtime.MemStats) string {
	return fmt.Sprintf("Alloc: %dMB, Sys: %dMB", 
		m.Alloc/1024/1024, m.Sys/1024/1024)
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
