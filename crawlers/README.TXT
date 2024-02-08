 **Here's a step-by-step explanation of the code:**

**1. Imports:**
- `bufio`: For reading user input from the console.
- `fmt`: For formatted printing to the console.
- `net/url`: For parsing and manipulating URLs.
- `os`: For accessing standard input.
- `strings`: For string manipulation.
- `github.com/gocolly/colly`: A web scraping framework for Go.
- `github.com/willf/bloom`: A Bloom filter implementation for efficient set membership testing.

**2. Constants:**
- `filterSize`: Sets the size of the Bloom filter to 100000 for tracking visited URLs.

**3. main Function:**
   - **Initializes a Bloom filter** to keep track of visited URLs.
   - **Creates a new Collector** using Colly for web scraping.
   - **Sets a callback function** for `a` elements with `href` attributes:
      - Extracts the link from the `href` attribute.
      - Preprocesses the link for consistency.
      - Checks if the link is already visited using the Bloom filter.
      - If not visited, prints a message and visits the link using Colly.
   - **Prompts the user for a starting URL** and reads it from the console.
   - **Preprocesses the starting URL.**
   - **Starts the crawling process** using Colly's `Visit` method.
   - **Waits for crawling to finish** using `c.Wait()`.

**4. preprocessURL Function:**
   - **Parses the input URL** using `url.Parse`.
   - **Handles potential errors** during parsing.
   - **Adds a default scheme** of `http` if missing.
   - **Checks for a valid host** and returns an empty string if invalid.
   - **Normalizes the URL:**
      - Converts the hostname to lowercase.
      - Removes the "www." prefix if present.
   - **Returns the preprocessed URL string.**

**Key Points:**

- The code creates a web crawler using Colly.
- It uses a Bloom filter to efficiently avoid visiting the same URL multiple times.
- It preprocesses URLs to handle variations in formatting.
- It starts crawling from a user-specified URL and continues crawling links found on pages.
