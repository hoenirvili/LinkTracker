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
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hoenirvili/Skapt"
	"golang.org/x/net/html"
)

const defaultPath = "./link.txt"

// read all content into the p.body page
func readPage(resp *http.Response, saveTo string, pref string) {
	var (
		z *html.Tokenizer
	)

	// read page
	if saveTo == "" {
		saveTo = defaultPath
	}

	// create file
	file, err := os.Create(saveTo)
	if err != nil {
		panic(err)
	}

	// take value per key
	value := resp.Header.Get("Content-Encoding")

	// test if the value is gzip or no
	switch value {
	case "gzip":
		r, err := gzip.NewReader(resp.Body)
		if err != nil {
			panic(err)
		}
		z = html.NewTokenizer(r)
	default:
		rb, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		z = html.NewTokenizer(bytes.NewReader(rb))
	}

	for {
		// get next html token
		tt := z.Next()
		switch {
		case tt == html.ErrorToken:
			if z.Err() == io.EOF {
				if err := file.Close(); err != nil {
					panic(err)
				}
				return
			}
		case tt == html.StartTagToken:
			t := z.Token()
			isAnchor := t.Data == "a" || t.Data == "img" || t.Data == "meta" || t.Data == "link" || t.Data == "script"
			if !isAnchor {
				continue
			}
			ok, url := getHref(t)
			if !ok {
				continue
			}

			hasProto := strings.Index(url, "http") == 0 || strings.Index(url, "https") == 0

			if hasProto {
				// append to file
				if _, err := file.WriteString(url + "\n"); err != nil {
					panic(err)
				}
			} else if _, err := file.WriteString(pref + url + "\n"); err != nil {
				panic(err)
			}

		}
	}

}

func getHref(t html.Token) (bool, string) {
	var (
		href string
		ok   bool
	)

	// Iterate all over the Token's attribute until we find "href"
	for _, a := range t.Attr {
		if a.Key == "href" || a.Key == "src" || a.Key == "content" {
			href = a.Val
			ok = true
		}
	}

	return ok, href
}

// create a new cunstom request setting
// headers and body
func newCustomRequest(method, url string) (*http.Request, error) {

	req, err := http.NewRequest(method, url, new(bytes.Buffer))

	if err != nil {
		return nil, fmt.Errorf(
			"Can't create this custom request with method %s, and url %s\n ERROR: %s\n",
			method, url, err.Error(),
		)
	}

	req.Header.Add("User-Agent", "LinkTracker1.0.0")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("Connection", "close")
	req.Header.Add("Accept", "text/html")
	req.Header.Add("Accept-Charset", "utf8")
	req.Header.Add("Accept-Encoding", "gzip")

	return req, nil
}

// create a new custom client with a specific timeout duration
// on every request
func newCustomClient(time time.Duration) *http.Client {
	return &http.Client{Timeout: time}
}

// main request method
func request() {
	url := app.String("-u")
	customPath := app.String("-f")

	req, err := newCustomRequest("GET", url)
	if err != nil {
		fmt.Println(err)
		return
	}

	// make new http Client object
	// setting the timeout duration for every request
	cli := newCustomClient(time.Duration(5 * time.Second))

	// make the request
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

	switch resp.StatusCode {
	case 200:
		readPage(resp, customPath, url)
		return
	default:
		fmt.Println(fmt.Sprintf(
			"Error occured with status %d\n", resp.StatusCode,
		))
		return
	}
}

var app *Skapt.App

func main() {
	// init all the command line stuff from Skapt framework
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

	// run the app
	app.Run()
}
