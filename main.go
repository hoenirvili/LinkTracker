// Copyright 2016 hoenirvili
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

	"github.com/hoenirvili/Skapt"
	"golang.org/x/net/html"
)

var app *Skapt.App

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

func request() {
	url := app.String("-u")

	req, err := newRequest(url)
	if err != nil {
		fmt.Println(err)
		return
	}

	cli := &http.Client{
		Timeout: time.Duration(5 * time.Second),
	}

	resp, err := cli.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			err = errClose
		}
	}()

	out := os.Stdout
	path := app.String("-f")
	if path != "" {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			panic(err)
		}
		out = file
		defer func() {
			if err = out.Close(); err != nil {
				fmt.Printf(err.Error())
			}
		}()
	}

	switch resp.StatusCode {
	case 200:
		if err = pageInto(resp, out, url); err != nil {
			fmt.Printf("Reading pager error occured: %s\n", err.Error())
		}
	default:
		fmt.Printf(
			"Http error occured with status %d\n", resp.StatusCode,
		)
	}
}

func init() {
	app = Skapt.NewApp()

	app.SetName("LinkTracker")
	app.SetUsage("Track all link into one file")
	app.SetDescription("This nitty gritty command line app watches a link, parsing every token, saving the link to a corresponding file")
	app.SetAuthors([]string{"Hoenirvili"})
	app.SetVersion(false, "1.0.0")
	app.AppendNewOption(Skapt.OptionParams{
		Name:        "-u",
		Alias:       "--url",
		Description: "Link to the specific url you wish to make the request",
		Type:        Skapt.STRING,
		Action:      request,
	})

	app.AppendNewOption(Skapt.OptionParams{
		Name:  "-f",
		Alias: "--file", Description: "File you wish to pipe the results",
		Type:   Skapt.STRING,
		Action: nil,
	})
}

func main() {
	app.Run()
}
