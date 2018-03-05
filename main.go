package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/net/html"
)

func pageInto(resp *http.Response, out io.Writer, pref string) error {
	var z *html.Tokenizer

	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		r, err := gzip.NewReader(resp.Body)
		if err != nil {
			return err
		}
		z = html.NewTokenizer(r)
	default:
		z = html.NewTokenizer(resp.Body)
	}

	z.SetMaxBuf(2048)
	for {
		tt := z.Next()
		switch {
		case tt == html.ErrorToken:
			return nil
		case tt == html.StartTagToken:
			t := z.Token()
			isAnchor := t.Data == "a" ||
				t.Data == "img" ||
				t.Data == "meta" ||
				t.Data == "link" ||
				t.Data == "script" ||
				t.Data == "iframe"
			if !isAnchor {
				continue
			}

			if ok, url := link(t); ok {
				if err := writeURL(out, pref, url); err != nil {
					return err
				}
			}
		}
	}
}

func writeURL(out io.Writer, pref, url string) error {
	if url == "" {
		return fmt.Errorf("Empty URL given")
	}

	hasProto := strings.Index(url, "http") == 0 ||
		strings.Index(url, "https") == 0

	var data []byte
	switch hasProto {
	case true:
		data = []byte(url + "\n")
	default:
		data = []byte(pref + url + "\n")
	}

	if _, err := out.Write(data); err != nil {
		return err
	}

	return nil
}

func link(t html.Token) (bool, string) {
	var link string
	ok := false

	for _, a := range t.Attr {
		switch a.Key {
		case "href", "src":
			link = a.Val
			if link != "" {
				ok = true
			}
		case "content":
			hasProto := strings.Index(a.Val, "http") == 0 ||
				strings.Index(a.Val, "https") == 0
			if hasProto {
				link = a.Val
				ok = true
			}
		}
	}

	return ok, link
}

func newRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, url, new(bytes.Buffer))
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", "LinkTracker1.0.0")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("Connection", "close")
	req.Header.Add("Accept", "text/html")
	req.Header.Add("Accept-Charset", "utf8")
	req.Header.Add("Accept-Encoding", "gzip")

	return req, nil
}

func request(cmd *cobra.Command, args []string) error {
	url := args[0]
	req, err := newRequest(url)
	if err != nil {
		return err
	}

	cli := &http.Client{
		Timeout: time.Duration(5 * time.Second),
	}

	resp, err := cli.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			err = errClose
		}
	}()

	out := os.Stdout
	if path != "" {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		out = file
		defer func() {
			if errClose := out.Close(); errClose != nil {
				err = errClose
			}
		}()
	}

	switch resp.StatusCode {
	case 200:
		if err = pageInto(resp, out, url); err != nil {
			return err
		}
	default:
		return fmt.Errorf("http error occured status %d", resp.StatusCode)
	}

	return nil
}

var path string

func main() {
	root := cobra.Command{
		Use:     "linktracker",
		Short:   "linktracker print or saves links from a web page",
		Long:    "Simple and easy way to parse and save links from a given web page",
		Version: "1.0.0",
		RunE:    request,
		Example: `linktracker -f /home/user/linkts.txt https://webpage.com
linktracker https://webpage.com`,
		Args: cobra.MinimumNArgs(1),
	}

	flags := root.Flags()
	flags.StringVarP(&path, "file", "f", "", "full path where to write results")

	if err := root.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
