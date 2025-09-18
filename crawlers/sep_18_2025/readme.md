```mermaid
flowchart TD
    Start[Start Crawler] --> Input[Read Starting URL from User]
    Input --> Init[Initialize Bloom Filter, File, Counters]
    
    Init --> StartVisit[Visit Starting URL]
    
    StartVisit --> HTMLParser[Parse HTML Content]
    
    HTMLParser --> FindLinks[Find All Anchor Tags]
    
    FindLinks --> Preprocess[Preprocess URL]
    Preprocess --> CheckEmpty{URL Empty?}
    
    CheckEmpty -->|Yes| Skip[Skip URL]
    CheckEmpty -->|No| CheckBloom[Check Bloom Filter]
    
    CheckBloom -->|Already Exists| Skip
    CheckBloom -->|New URL| AddToBloom[Add to Bloom Filter]
    
    AddToBloom --> UpdateCounters[Increment Counters]
    UpdateCounters --> Output[Write to Console & File]
    Output --> ScheduleVisit[Schedule URL Visit]
    
    Skip --> WaitForMore
    
    ScheduleVisit --> WaitForMore[Wait for More Links]
    
    WaitForMore --> Finished{All Pages Processed?}
    Finished -->|No| HTMLParser
    Finished -->|Yes| FinalOutput[Write Final Telemetry]
    
    FinalOutput --> End[End Crawler]
    
    style Start fill:#9f9,color:black
    style End fill:#f9f,color:black
    style CheckBloom fill:#bbf,color:black
    style Output fill:#ff9,color:black
```

#Process Explanation:
Initialization: The crawler starts by reading the input URL and initializing components:

Bloom filter for duplicate detection

Output file for results

Counters for telemetry

HTML Parsing: For each page visited, the crawler extracts all anchor tags (<a href>)

URL Processing:

Preprocesses URLs to handle variations (www, http/https, etc.)

Checks if URL is valid

Uses Bloom filter to detect duplicates

Output Handling:

New URLs are written to both console and file

Counters are updated for telemetry

URLs are scheduled for visiting

Completion: When all pages are processed, final telemetry is written and the crawler exits

The Bloom filter acts as a probabilistic cache that prevents revisiting the same URL, while the file output provides a persistent record of all unique URLs discovered during the crawl.
#

# Go Web Crawler Architecture

Visualizing the structure and dependencies of the concurrent web crawler.

---

## System Architecture

```mermaid
graph TD
    A[User Input] --> B(Read Starting URL)
    B --> C[Initialize Crawler]
    C --> D[Bloom Filter]
    C --> E[Output File]
    C --> F[Colly Collector]
    
    subgraph "Concurrency Control"
        G[WaitGroup]
        H[Rate Limiter]
    end
    
    F --> H
    
    subgraph "URL Processing"
        I[Extract Links]
        J[Preprocess URLs]
        K[Check Bloom Filter]
    end
    
    F --> I
    I --> J
    J --> K
    
    K -->|New URL| L[Visit URL]
    K -->|Duplicate| M[Skip]
    
    L --> F
    
    subgraph "Output System"
        N[Console Output]
        O[File Output]
        P[Telemetry]
    end
    
    I --> N
    I --> O
    I --> P
    
    style A fill:#74b9ff
    style D fill:#a29bfe
    style F fill:#fd79a8
    style G fill:#55efc4
    style K fill:#ffeaa7
    style N fill:#fab1a0
```
Go Version & Dependencies
Required Go Version
The code requires Go 1.16 or later due to its use of specific standard library features and module support.

Dependencies
The crawler uses these external packages:

github.com/gocolly/colly - Web scraping framework
github.com/willf/bloom - Bloom filter implementation
Key Features
Concurrent crawling with configurable parallelism
Bloom filter for efficient duplicate URL detection
URL normalization and preprocessing
Real-time telemetry and progress reporting
Persistent output to timestamped files
Respectful crawling with rate limiting
Sample go.mod
module my-crawler go 1.21 require ( github.com/gocolly/colly v1.2.0 github.com/willf/bloom v2.0.3+incompatible )
