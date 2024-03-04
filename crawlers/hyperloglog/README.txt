 go run -race .
# mycrawler
./CBTCP.go:18:2: filterSize redeclared in this block
	./CBCP.go:18:2: other declaration of filterSize
./CBTCP.go:21:6: main redeclared in this block
	./CBCP.go:21:6: other declaration of main
./CBTCP.go:110:6: preprocessURL redeclared in this block
	./CBCP.go:113:6: other declaration of preprocessURL
./CBTCP.go:134:6: urlToFileName redeclared in this block
	./CBCP.go:137:6: other declaration of urlToFileName
./CBTWC_Ulinux.go:18:7: filterSize redeclared in this block
	./CBCP.go:18:2: other declaration of filterSize
./CBTWC_Ulinux.go:20:6: main redeclared in this block
	./CBCP.go:21:6: other declaration of main
./CBTWC_Ulinux.go:111:6: preprocessURL redeclared in this block
	./CBCP.go:113:6: other declaration of preprocessURL
./CBTWC_Ulinux.go:135:6: urlToFileName redeclared in this block
	./CBCP.go:137:6: other declaration of urlToFileName
./CBTWC_Ulinuxv2.go:18:7: filterSize redeclared in this block
	./CBCP.go:18:2: other declaration of filterSize
./CBTWC_Ulinuxv2.go:20:6: main redeclared in this block
	./CBCP.go:21:6: other declaration of main
./CBTWC_Ulinuxv2.go:20:6: too many errors

The error messages you're seeing indicate that there are multiple Go files in the same package that declare the same variables and functions, such as `filterSize`, `main`, `preprocessURL`, and `urlToFileName`. This is causing a conflict because in Go, each variable, function, or type must have a unique name within the same package unless they are declared in a block (like `const`, `var`, or `type`).

Here are the specific issues pointed out by the error messages:

- `filterSize redeclared in this block`: The `filterSize` variable is declared in multiple files.
- `main redeclared in this block`: The `main` function, which is the entry point of the program, is declared in multiple files. A Go program can only have one `main` function in the main package.
- `preprocessURL redeclared in this block`: The `preprocessURL` function is declared in multiple files.
- `urlToFileName redeclared in this block`: The `urlToFileName` function is declared in multiple files.

To resolve these issues, you need to ensure that each variable, function, and type is uniquely named across all files in the package or to consolidate duplicate declarations into a single file. If the logic in these functions and variables is supposed to be shared across different parts of your program, then they should be declared only once.

Given this context, I cannot directly refactor your code without seeing the contents of the files mentioned (`CBTCP.go`, `CBCP.go`, `CBTWC_Ulinux.go`, `CBTWC_Ulinuxv2.go`). However, I can suggest a general approach to resolve the issue:

1. **Consolidate Declarations**: If the duplicated declarations (`filterSize`, `preprocessURL`, `urlToFileName`) are identical across files, consider keeping them in only one file (perhaps a `util.go` or similarly named file) and remove the duplicates from the other files.

2. **Rename Functions or Variables If Different**: If the functions or variables serve different purposes in different files, consider renaming them to reflect their specific use cases.

3. **Single `main` Function**: Ensure there's only one `main` function across all the files in your package. If you have logically separate programs, they should be split into separate packages or modules.

After making these adjustments, you should be able to run your program without encountering the redeclaration errors. If you're working with multiple main packages (separate programs), ensure they are organized in separate directories and that each directory only contains one `main` function.

panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x1 addr=0x27 pc=0x7b8c57]

goroutine 14584 [running]:
github.com/PuerkitoBio/goquery.(*Selection).Text.func1(0x480428?)
	/home/jeb/go/pkg/mod/github.com/!puerkito!bio/goquery@v1.8.1/property.go:68 +0x17
github.com/PuerkitoBio/goquery.(*Selection).Text.func1(0xc0812c3c40?)
	/home/jeb/go/pkg/mod/github.com/!puerkito!bio/goquery@v1.8.1/property.go:74 +0x5f
github.com/PuerkitoBio/goquery.(*Selection).Text.func1(0xc0e3198cb0?)
	/home/jeb/go/pkg/mod/github.com/!puerkito!bio/goquery@v1.8.1/property.go:74 +0x5f
github.com/PuerkitoBio/goquery.(*Selection).Text.func1(0x30?)
	/home/jeb/go/pkg/mod/github.com/!puerkito!bio/goquery@v1.8.1/property.go:74 +0x5f
github.com/PuerkitoBio/goquery.(*Selection).Text(0x85c820?)
	/home/jeb/go/pkg/mod/github.com/!puerkito!bio/goquery@v1.8.1/property.go:79 +0x91
github.com/gocolly/colly.NewHTMLElementFromSelectionNode(0xc0defaa240, 0xc0e34bd080, 0xc0e3198c40, 0x689)
	/home/jeb/go/pkg/mod/github.com/gocolly/colly@v1.2.0/htmlelement.go:47 +0x10d
github.com/gocolly/colly.(*Collector).handleOnHTML.func1(0xa16f00?, 0xc0e34bd080)
	/home/jeb/go/pkg/mod/github.com/gocolly/colly@v1.2.0/colly.go:955 +0xbd
github.com/PuerkitoBio/goquery.(*Selection).Each(0xc0e31ac180, 0xc0812c3e18)
	/home/jeb/go/pkg/mod/github.com/!puerkito!bio/goquery@v1.8.1/iteration.go:10 +0x4f
github.com/gocolly/colly.(*Collector).handleOnHTML(0xc000278d00, 0xc0defaa240)
	/home/jeb/go/pkg/mod/github.com/gocolly/colly@v1.2.0/colly.go:953 +0x210
github.com/gocolly/colly.(*Collector).fetch(0xc000278d00, {0x0?, 0x0?}, {0x957dbf, 0x3}, 0x5, {0x0?, 0x0}, 0x0?, 0xc0035abb90, ...)
	/home/jeb/go/pkg/mod/github.com/gocolly/colly@v1.2.0/colly.go:623 +0x677
created by github.com/gocolly/colly.(*Collector).scrape in goroutine 1042
	/home/jeb/go/pkg/mod/github.com/gocolly/colly@v1.2.0/colly.go:532 +0x66d
exit status 2

The panic you're encountering is a `nil` pointer dereference error occurring within the `goquery` library, which is used by `colly` for parsing and interacting with HTML documents. This kind of error happens when you attempt to access a member of a `nil` pointer, in this case, likely a `*goquery.Selection` object that doesn't point to an initialized instance.

Here's a breakdown of the stack trace leading to the panic:

1. `github.com/PuerkitoBio/goquery.(*Selection).Text.func1` and subsequently `Text` method calls indicate the error originates from attempting to get text content from a selection that is nil or doesn't exist.
2. `github.com/gocolly/colly.NewHTMLElementFromSelectionNode` suggests that Colly is trying to create a new `HTMLElement` from a node selected by `goquery`, but the selection is nil.
3. `github.com/gocolly/colly.(*Collector).handleOnHTML.func1` points out that this is happening in the context of handling an HTML element based on a CSS selector.
4. The error ultimately leads to the panic during the execution of `colly.(*Collector).fetch` method, which is part of Colly's process of fetching and processing a web page.

To address this issue:

- **Check Your Selectors**: Ensure that the CSS selectors you're using in `OnHTML` callbacks are correct and match elements in the HTML documents you are scraping. If a selector doesn't match any elements, `goquery` may return a `nil` `*Selection`, leading to a panic when you try to use it.

- **Validate `*goquery.Selection` Before Use**: Before calling methods like `Text()` on a `*goquery.Selection`, check if the selection is not `nil`. However, note that `goquery.Selection` itself is rarely `nil`; more commonly, it might be an empty selection (i.e., selecting no elements). You can check if a selection is empty by checking `selection.Length() == 0`.

Here's an example of how you might adjust your code to prevent this error:

```go
c.OnHTML("a[href]", func(e *colly.HTMLElement) {
    // Example: Safely getting text from a selection
    selection := e.DOM
    if selection.Length() > 0 { // Ensure the selection is not empty
        text := selection.Text()
        fmt.Println(text)
    } else {
        log.Println("No elements match the provided selector.")
    }
})
```

- **Debug and Log**: Add more logging before the line that causes the panic to see what URLs or content might be causing the issue. This can help identify if specific pages are structured in a way that causes your selectors to fail.

Addressing these points should help you avoid encountering a nil pointer dereference when using Colly and `goquery`.

The errors you're seeing seem related to internal operations within the `colly` package, specifically when sending requests (`Do` method) and handling caching (`Cache` method). These errors might indicate a problem with managing concurrency or resources within your scraping tasks, but they do not directly point to misuse of the API or incorrect handling on your part. 

### Potential Causes and Solutions

1. **Concurrency Overload**: If your program is launching a high number of concurrent scraping tasks, it might be overwhelming either the local system resources (like file descriptors or network connections) or the target server. 
   - **Solution**: Try reducing the `Parallelism` setting in your `LimitRule` or introduce delays between requests to spread the load more evenly over time.

2. **Resource Leaks**: The runtime errors and goroutine stack traces suggest that your program might be running into issues where resources are not being released properly, possibly leading to resource exhaustion.
   - **Solution**: Ensure that all response bodies are being closed properly. Although `colly` abstracts away many of the HTTP client details, improper handling of resources in callbacks or elsewhere in your code might still lead to leaks. Review your code to ensure you're not holding onto resources longer than necessary.

3. **Target Server Behavior**: The target server might be responding in a way that causes unexpected behavior in the `colly` HTTP backend, especially under high load or when encountering non-standard HTTP responses.
   - **Solution**: Test your scraper against a variety of target URLs to see if the problem is isolated to specific sites. Implement error handling in your callbacks to gracefully handle failed requests or unexpected responses.

4. **Error Handling in Callbacks**: Ensure that your `OnHTML` and other callbacks are properly handling errors, especially for actions like visiting new links.
   - **Solution**: Add comprehensive error checking within callbacks and consider adding a `OnRequest` callback to log or handle request errors more gracefully.

5. **Update Dependencies**: Ensure you're using the latest version of `colly` and its dependencies. There might have been fixes to issues similar to what you're experiencing.
   - **Solution**: Run `go get -u github.com/gocolly/colly` to update `colly` to the latest version, and then test your scraper again.

6. **Debugging and Logging**: Increase the visibility into what your program is doing, especially around the points where errors occur.
   - **Solution**: Utilize `colly`'s debugging capabilities by setting a debugger via `colly.Debugger()` or by adding more logging in your code, especially before making requests or in error handling paths.

Here's a small adjustment to introduce error handling in the visitation logic, which might help identify issues:

```go
c.OnHTML("a[href]", func(e *colly.HTMLElement) {
    link := e.Request.AbsoluteURL(e.Attr("href"))
    link = preprocessURL(link)

    mu.Lock()
    if visited[link] {
        mu.Unlock()
        return
    }
    visited[link] = true
    mu.Unlock()

    linksProcessed++
    hll.Insert([]byte(link))

    log.Printf("Visiting: %s\n", link)
    file.WriteString(link + "\n")

    // Error handling for visit
    err := e.Request.Visit(link)
    if err != nil {
        log.Printf("Error visiting %s: %v", link, err)
    }
})
```

Remember, the nature of web scraping means dealing with unpredictable web content and server behavior, so robust error handling and adaptability in your code are crucial.
