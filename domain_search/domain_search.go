package main

import (
    "fmt"
    "io"
    "net/http"
    "regexp"
    "strings"
)

func main() {
    // Prompt the user to enter the URL
    fmt.Print("Enter the URL to search for domains: ")
    scanner := bufio.NewScanner(os.Stdin)
    scanner.Scan()
    url := scanner.Text()

    domains := searchDomains(url)

    // Print the discovered domains
    fmt.Println("Discovered domains:")
    for _, domain := range domains {
        fmt.Println(domain)
    }
}

func searchDomains(url string) []string {
    fmt.Printf("Fetching %s\n", url)
    resp, err := http.Get(url)
    if err != nil {
        fmt.Printf("Error fetching %s: %v\n", url, err)
        return nil
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        fmt.Printf("Failed to fetch %s: status %s\n", url, resp.Status)
        return nil
    }

    fmt.Printf("Reading response body for %s\n", url)
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        fmt.Printf("Error reading response body for %s: %v\n", url, err)
        return nil
    }

    fmt.Printf("Searching for domains in %s\n", url)
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
