// Copyright (C) 2019  Allen Li
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

// Package saucenao implements a SauceNAO API client.
//
// This package does not implement rate limiting.
// Consider using a rate limiting package like golang.org/x/time/rate.
package saucenao

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Client is a SauceNAO API client.
type Client struct {
	C http.Client
	// Service is the SauceNAO service to call,
	// e.g. https://saucenao.com
	Service string
	APIKey  string
}

// NewClient returns a new Client for saucenao.com.
func NewClient(apiKey string) *Client {
	return &Client{
		Service: "https://saucenao.com",
		APIKey:  apiKey,
	}
}

// SearchRequest describes a search request.
// See the SauceNAO API page for details.
type SearchRequest struct {
	// URL of image to search.
	// Should not be provided with Image.
	URL string
	// ImageBytes is the image to search.
	// Should not be provided with URL.
	ImageBytes io.Reader
	// TestMode limits matches per index to one.
	TestMode bool
	// DBMask is a bitmap indicating indexes to search.
	DBMask DBMask
	// DBMaskI is a bitmap indicating indexes to ignore.
	DBMaskI DBMask
	// NumRes is the number of results to request.
	NumRes uint32
}

// DBMask is a bitmask for selecting database indexes.
type DBMask uint64

// These are database index constants.
const (
	Pixiv    int = 5
	Danbooru int = 9
	Yandere  int = 12
	Gelbooru int = 25
	Konachan int = 26

	PixivBit    DBMask = 1 << Pixiv
	DanbooruBit DBMask = 1 << Danbooru
	YandereBit  DBMask = 1 << Yandere
	GelbooruBit DBMask = 1 << Gelbooru
	KonachanBit DBMask = 1 << Konachan
)

// Search calls the SauceNAO search API.
func (c *Client) Search(ctx context.Context, r *SearchRequest) (*SearchResponse, error) {
	req, err := c.requestForSearch(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("saucenao search: %w", err)
	}
	resp, err := c.C.Do(req)
	if err != nil {
		return nil, fmt.Errorf("saucenao search: %s", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200:
	case 429:
		return nil, fmt.Errorf("saucenao search: %w", QuotaError{})
	default:
		return nil, fmt.Errorf("saucenao search: unexpected status %v", resp.Status)
	}
	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("saucenao search: %s", err)
	}
	var sr SearchResponse
	if err := json.Unmarshal(d, &sr); err != nil {
		return nil, fmt.Errorf("saucenao search: %s", err)
	}
	return &sr, nil
}

func (c *Client) requestForSearch(ctx context.Context, r *SearchRequest) (*http.Request, error) {
	switch r.ImageBytes {
	case nil:
		req, err := http.NewRequestWithContext(ctx, "GET", c.searchURL(r), nil)
		if err != nil {
			panic(fmt.Sprintf("failed to create request: %s", err))
		}
		return req, nil
	default:
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		part, err := w.CreateFormFile("file", "image")
		if err != nil {
			panic(fmt.Sprintf("failed to create form: %s", err))
		}
		_, err = io.Copy(part, r.ImageBytes)
		if err != nil {
			return nil, err
		}
		if err := w.Close(); err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, "POST", c.searchURL(r), &b)
		if err != nil {
			panic(fmt.Sprintf("failed to create request: %s", err))
		}
		req.Header.Set("Content-Type", w.FormDataContentType())
		return req, nil
	}
}

// searchURL returns the URL for performing a search request.
func (c *Client) searchURL(r *SearchRequest) string {
	var b strings.Builder
	b.WriteString(c.Service)
	b.WriteString("/search.php?output_type=2&api_key=")
	b.WriteString(c.APIKey)
	b.WriteString("&numres=")
	b.WriteString(strconv.FormatUint(uint64(r.NumRes), 10))
	if r.TestMode {
		b.WriteString("&testmode=1")
	}
	if r.DBMask != 0 {
		b.WriteString("&dbmask=")
		b.WriteString(strconv.FormatUint(uint64(r.DBMask), 10))
	}
	if r.DBMaskI != 0 {
		b.WriteString("&dbmaski=")
		b.WriteString(strconv.FormatUint(uint64(r.DBMaskI), 10))
	}
	if r.URL != "" {
		b.WriteString("&url=")
		b.WriteString(url.QueryEscape(r.URL))
	}
	return b.String()
}

// SearchResponse is the parsed search response.
type SearchResponse struct {
	Header  SearchHeader   `json:"header"`
	Results []SearchResult `json:"results"`
}

// SearchHeader is the header for a search response.
type SearchHeader struct {
	Status           int `json:"status"`
	ResultsRequested int `json:"results_requested"`
	ResultsReturned  int `json:"results_returned"`

	ShortRemaining int `json:"short_remaining"`
	LongRemaining  int `json:"long_remaining"`
	ShortLimit     int `json:"short_limit,string"`
	LongLimit      int `json:"Long_limit,string"`

	MinimumSimilarity float64 `json:"minimum_similarity"`
}

// SearchResult is one result from a search.
type SearchResult struct {
	Header SearchResultHeader `json:"header"`
	Data   json.RawMessage    `json:"data"`
}

// AsDanbooru returns the result data parsed for Danbooru.
func (r *SearchResult) AsDanbooru() (*DanbooruData, error) {
	var d DanbooruData
	if err := json.Unmarshal(r.Data, &d); err != nil {
		return nil, fmt.Errorf("search result as danbooru: %w", err)
	}
	return &d, nil
}

// SearchResultHeader is the header of a SearchResult.
type SearchResultHeader struct {
	IndexName  string  `json:"index_name"`
	IndexID    int     `json:"index_id"`
	Thumbnail  string  `json:"thumbnail"`
	Similarity float64 `json:"similarity,string"`
}

// A QuotaError is returned when requests are rate limited.
type QuotaError struct{}

func (QuotaError) Error() string {
	return "rate limited"
}
