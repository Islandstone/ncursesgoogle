//google stuff
package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
)

var (
	googleApi = `http://ajax.googleapis.com/ajax/services/search/web?v=1.0&rsz=8&q=`
	tags      = regexp.MustCompile(`<[^>]*>`)
)

type Result struct {
	GsearchResultClass string
	UnescapedUrl       string
	Url                string
	VisibleUrl         string
	CacheUrl           string
	Title              string
	TitleNoFormatting  string
	Content            string
}

type Cursor struct {
	ResultCount string
}

type ResponseData struct {
	Results *[]Result
	Cursor  *Cursor
}

type Response struct {
	ResponseData    *ResponseData
	ResponseDetails string
	ResponseStatus  float64
}

type GoogleSearch struct {
}

func Google(query string) (results Response, err error) {
	var resp *http.Response

	resp, err = http.Get(googleApi + url.QueryEscape(query))
	if err != nil {
		return
	}

	defer resp.Body.Close()

	var response Response

	var dec = json.NewDecoder(resp.Body)
	err = dec.Decode(&response)
	if err != nil {
		return
	}

	results = response
	return
}
