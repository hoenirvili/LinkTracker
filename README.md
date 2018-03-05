# LinkTracker

#### Command line tool for tracking links in a web page

#### Dependencies
	go get -v -u github.com/spf13/cobra
	go get -v -u golang.org/x/net/html
#### Output

```bash
Simple and easy way to parse and save links from a given web page

Usage:
  linktracker [flags]

Examples:
linktracker -f /home/user/linkts.txt https://webpage.com
linktracker https://webpage.com

Flags:
  -f, --file string   full path where to write results
  -h, --help          help for linktracker
      --version       version for linktracker
```
