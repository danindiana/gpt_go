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
