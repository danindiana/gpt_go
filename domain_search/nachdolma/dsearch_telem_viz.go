package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"os"
	"regexp"
	"strings"
	"time"
)

func main() {
	fmt.Print("\033[32m\033[40m") // Green text on black background
	defer fmt.Print("\033[0m")    // Reset terminal formatting

	typewriterPrint("Welcome to the Domain Search Telemetry Tool!\n")
	typewriterPrint("Enter the URL to search for domains: ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	url := scanner.Text()

	domains := searchDomains(url)

	typewriterPrint("Discovered domains:\n")
	for _, domain := range domains {
		fmt.Println(domain)
	}
	typewriterPrint("Thank you for using the tool. Goodbye!\n")
}

func searchDomains(url string) []string {
	fmt.Printf("\nFetching %s...\n", url)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	var start, dnsStart, connectStart, tlsStart, firstByte time.Time
	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			dnsStart = time.Now()
			fmt.Printf("DNS Start: %s\n", info.Host)
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			fmt.Printf("DNS Done: %s (time: %v)\n", info.Addrs, time.Since(dnsStart))
		},
		ConnectStart: func(network, addr string) {
			connectStart = time.Now()
			fmt.Printf("Connecting to %s (%s)...\n", addr, network)
		},
		ConnectDone: func(network, addr string, err error) {
			fmt.Printf("Connection established: %s (%s) (time: %v)\n", addr, network, time.Since(connectStart))
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
			fmt.Println("Starting TLS handshake...")
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			fmt.Printf("TLS handshake complete (time: %v)\n", time.Since(tlsStart))
		},
		GotFirstResponseByte: func() {
			firstByte = time.Now()
			fmt.Println("First byte received!")
		},
	}
	ctx := httptrace.WithClientTrace(context.Background(), trace)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		fmt.Printf("Error creating request for %s: %v\n", url, err)
		return nil
	}

	start = time.Now()
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error fetching %s: %v\n", url, err)
		return nil
	}
	defer resp.Body.Close()

	fmt.Printf("Total time for request: %v\n", time.Since(start))
	fmt.Printf("Time to first byte: %v\n", time.Since(firstByte))

	fmt.Println("\nReading response body...")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body for %s: %v\n", url, err)
		return nil
	}

	fmt.Println("Searching for domains...")
	progressBar(30, time.Millisecond*50) // Simulate progress bar
	domainRegex := regexp.MustCompile(`https?://([\w-]+\.)+[\w-]+`)
	matches := domainRegex.FindAllString(string(body), -1)

	var domains []string
	for _, match := range matches {
		domain := strings.ToLower(match)
		fmt.Printf("Found domain: %s\n", domain)
		domains = append(domains, domain)
	}

	return domains
}

// Simulate typing effect
func typewriterPrint(text string) {
	for _, char := range text {
		fmt.Print(string(char))
		time.Sleep(50 * time.Millisecond)
	}
}

// Simulate a progress bar
func progressBar(steps int, delay time.Duration) {
	for i := 0; i <= steps; i++ {
		bar := strings.Repeat("#", i) + strings.Repeat(" ", steps-i)
		fmt.Printf("\r[%s] %d%%", bar, (i*100)/steps)
		time.Sleep(delay)
	}
	fmt.Println()
}
