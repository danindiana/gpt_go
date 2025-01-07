It uses the **Colly library** for web scraping and includes robust error handling, retry mechanisms, and concurrency to ensure efficient and reliable operation. Below is a detailed explanation of how the code works, its components, and the changes we’ve made to improve it.

---

### **What the Code Does**
1. **Crawls Websites**:
   - Starts from a user-provided URL.
   - Follows links on the page to discover additional URLs.
   - Specifically looks for links ending in `.pdf` to identify PDF files.

2. **Downloads PDF Files**:
   - Downloads discovered PDF files to a specified directory.
   - Handles both HTTP/HTTPS and FTP URLs.

3. **Handles Errors and Retries**:
   - Retries failed downloads with exponential backoff.
   - Skips retries for permanent errors (e.g., `404 Not Found`, `403 Forbidden`).
   - Logs errors for debugging.

4. **Runs Concurrently**:
   - Uses goroutines to download PDF files concurrently.
   - Processes delayed requests in the background.

5. **Provides User Interaction**:
   - Prompts the user to select a network interface and download directory.
   - Allows the user to specify the starting URL.

---

### **How It Works**
1. **Initialization**:
   - The program starts by listing available network interfaces and prompting the user to select one.
   - It then lists directories in the current working directory and allows the user to select a download directory or create a new one.

2. **Starting the Crawler**:
   - The user provides a starting URL, which is sanitized and validated.
   - A Colly collector is initialized with a custom HTTP transport to use the selected network interface.

3. **Crawling**:
   - The crawler visits the starting URL and follows all links (`<a href>` tags).
   - If a link points to a PDF file (ends with `.pdf`), it is added to the download queue.

4. **Downloading Files**:
   - PDF files are downloaded concurrently using goroutines.
   - The `downloadFileWithTimeout` function handles HTTP/HTTPS and FTP downloads.
   - Failed downloads are retried with exponential backoff.

5. **Error Handling**:
   - Errors are categorized (e.g., DNS errors, connection refused, timeouts).
   - Permanent errors (e.g., `404 Not Found`, `403 Forbidden`) are logged and skipped.
   - Transient errors (e.g., timeouts, connection issues) are retried.

6. **Delayed Requests**:
   - Failed requests are added to a `delayedQueue` channel.
   - A separate goroutine (`processDelayedQueue`) retries these requests.

7. **Completion**:
   - The program waits for all downloads to complete using a `sync.WaitGroup`.
   - Once finished, it logs a completion message.

---

### **Changes Made to the Code**
1. **Removed Unnecessary Features**:
   - Removed the `excludedDomains` list and `visitedURLsMap` to simplify the code.

2. **Enhanced Error Handling**:
   - Added detailed error categorization for HTTP requests (e.g., DNS errors, timeouts, connection refused).
   - Skipped retries for permanent errors (`404 Not Found`, `403 Forbidden`).

3. **Improved URL Sanitization**:
   - Added URL encoding for spaces (`%20`) and colons (`%3A`) to handle invalid characters.

4. **Added Custom User-Agent Header**:
   - Set a custom `User-Agent` header to mimic a real browser and bypass restrictions.

5. **Increased Buffer Size for `delayedQueue`**:
   - Increased the buffer size of the `delayedQueue` channel to `10000` to handle more retries.

6. **Added Timeouts for Channel Sends**:
   - Used a `select` statement with a timeout when sending to the `delayedQueue` channel to avoid blocking indefinitely.

7. **Concurrent Downloads**:
   - Used goroutines to download PDF files concurrently, improving performance.

8. **Exponential Backoff for Retries**:
   - Implemented exponential backoff for retries, with a maximum delay of 30 seconds.

9. **Verbose Logging**:
   - Added detailed logging for debugging, including:
     - URLs being visited.
     - PDF files found.
     - Errors encountered.
     - Retry attempts.

10. **Network Interface Selection**:
    - Allowed the user to select a network interface for outgoing requests.

11. **Directory Selection**:
    - Allowed the user to select an existing directory or create a new one for storing downloaded files.

---

### **Key Components**
1. **Colly Collector**:
   - Handles web crawling and link discovery.
   - Uses a queue to manage URLs to visit.

2. **HTTP Client**:
   - Customized with a timeout and insecure TLS configuration.
   - Uses the selected network interface for outgoing requests.

3. **Delayed Queue**:
   - A buffered channel (`delayedQueue`) stores URLs that need to be retried.
   - A separate goroutine processes this queue.

4. **Error Categorization**:
   - Errors are categorized into:
     - DNS errors.
     - Connection errors (e.g., refused, unreachable).
     - Timeouts.
     - HTTP errors (e.g., `404`, `403`).

5. **Exponential Backoff**:
   - Retries are delayed using exponential backoff to avoid overwhelming the server.

6. **Concurrency**:
   - Goroutines are used to download PDF files concurrently.
   - A `sync.WaitGroup` ensures the program waits for all downloads to complete.

---

### **Example Workflow**
1. **User Input**:
   - The user selects a network interface, download directory, and starting URL.

2. **Crawling**:
   - The crawler visits the starting URL and discovers links.
   - If a link points to a PDF file, it is added to the download queue.

3. **Downloading**:
   - PDF files are downloaded concurrently.
   - Failed downloads are retried with exponential backoff.

4. **Completion**:
   - The program logs the completion of all downloads.

---

### **Benefits of the Final Code**
1. **Robustness**:
   - Handles a wide range of errors gracefully.
   - Retries transient errors and skips permanent ones.

2. **Efficiency**:
   - Downloads files concurrently for faster performance.
   - Uses exponential backoff to avoid overwhelming servers.

3. **User-Friendly**:
   - Provides interactive prompts for network interface and directory selection.
   - Logs detailed information for debugging.

4. **Scalability**:
   - Can handle large numbers of URLs and retries efficiently.

---

To get this code running from the **bash terminal** on a **Linux system**, follow these steps:

---

### **Prerequisites**
1. **Go Installed**:
   - Ensure Go is installed on your system. You can check by running:
     ```bash
     go version
     ```
   - If Go is not installed, follow the official installation guide: [Install Go](https://golang.org/doc/install).

2. **Colly Library**:
   - The code uses the **Colly** library for web scraping. Install it by running:
     ```bash
     go get -u github.com/gocolly/colly
     ```

3. **Network Interface**:
   - Ensure you have a working network interface with internet access.

4. **Directory Permissions**:
   - Ensure you have write permissions in the directory where you plan to run the program and store downloaded files.

---

### **Steps to Run the Code**

#### 1. **Save the Code**
   - Save the final code to a file, e.g., `pdf_crawler.go`.

#### 2. **Navigate to the Code Directory**
   - Open a terminal and navigate to the directory where the `pdf_crawler.go` file is saved:
     ```bash
     cd /path/to/your/code
     ```

#### 3. **Build the Program**
   - Compile the Go code into an executable:
     ```bash
     go build -o pdf_crawler
     ```
   - This will create an executable file named `pdf_crawler` in the current directory.

#### 4. **Run the Program**
   - Execute the compiled program:
     ```bash
     ./pdf_crawler
     ```

#### 5. **Follow the Prompts**
   - The program will prompt you for the following inputs:
     1. **Network Interface**:
        - It will list available network interfaces. Enter the number corresponding to the interface you want to use.
     2. **Download Directory**:
        - It will list available directories. Enter the number corresponding to the directory where you want to save the downloaded PDFs, or enter `0` to create a new directory.
     3. **Starting URL**:
        - Enter the URL from which the crawler should start.

#### 6. **Monitor the Output**
   - The program will log its progress to the terminal, including:
     - URLs being visited.
     - PDF files found.
     - Errors encountered.
     - Retry attempts.

#### 7. **Check Downloaded Files**
   - Once the program finishes, check the selected download directory for the downloaded PDF files.

---

### **Example Workflow in the Terminal**

#### Step 1: Save the Code
```bash
nano pdf_crawler.go
```
- Paste the final code into the editor and save the file.

#### Step 2: Navigate to the Directory
```bash
cd /home/user/code
```

#### Step 3: Build the Program
```bash
go build -o pdf_crawler
```

#### Step 4: Run the Program
```bash
./pdf_crawler
```

#### Step 5: Follow the Prompts
```
Available network interfaces:
1: eth0
2: wlan0
Select the network interface to use (enter the number): 1

Available directories:
1: downloads
2: documents
0: Create a new directory named 'pdf-scrape-<timestamp>'
Select the directory to store downloaded PDFs (enter the number or hit enter for default): 0
Created new directory: pdf-scrape-20231010120000

Enter the starting URL to crawl: http://example.com
```

#### Step 6: Monitor the Output
```
Visiting http://example.com
Found link: http://example.com/about
Found PDF URL: http://example.com/file.pdf
Downloading file from URL: http://example.com/file.pdf
Successfully downloaded file: /path/to/pdf-scrape-20231010120000/file.pdf
```

#### Step 7: Check Downloaded Files
```bash
ls /path/to/pdf-scrape-20231010120000
```
- You should see the downloaded PDF files in the specified directory.

---

### **Troubleshooting**
1. **Missing Colly Library**:
   - If you see an error like `cannot find package "github.com/gocolly/colly"`, ensure you installed Colly:
     ```bash
     go get -u github.com/gocolly/colly
     ```

2. **Permission Denied**:
   - If you encounter permission issues, run the program with `sudo`:
     ```bash
     sudo ./pdf_crawler
     ```

3. **Network Issues**:
   - Ensure the selected network interface has internet access.

4. **Invalid URL**:
   - If the starting URL is invalid, the program will log an error and exit. Double-check the URL.

---

### **Automating the Process**
If you want to run the program without manual input, you can pass arguments directly to the executable. For example:
```bash
./pdf_crawler --interface=1 --dir=0 --url=http://example.com
```
- You would need to modify the code to accept command-line arguments using the `flag` package.

---

This step-by-step guide should help you get the code running on a Linux system. Let me know if you encounter any issues!
