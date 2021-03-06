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

package saucenao

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestClient_Search_image(t *testing.T) {
	t.Parallel()
	st := newSpyTransport(t)
	c := Client{
		C: http.Client{
			Transport: &st,
		},
		Service: "https://example.com",
		APIKey:  "amiya",
	}
	req := SearchRequest{
		ImageBytes: bytes.NewReader([]byte("eyjafjalla")),
	}
	_, _ = c.Search(context.Background(), &req)
	checkRequestFile(t, st.req, []byte("eyjafjalla"))
}

func TestSearchResponse_unmarshal(t *testing.T) {
	t.Parallel()
	d, err := ioutil.ReadFile(filepath.Join("testdata", "response.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got SearchResponse
	if err := json.Unmarshal(d, &got); err != nil {
		t.Fatal(err)
	}
	want := SearchResponse{
		Header: SearchHeader{
			Status:            0,
			ResultsRequested:  16,
			ResultsReturned:   16,
			ShortRemaining:    5,
			LongRemaining:     199,
			ShortLimit:        6,
			LongLimit:         200,
			MinimumSimilarity: 24.6,
		},
		Results: []SearchResult{
			{
				Header: SearchResultHeader{
					IndexName:  "Index #9: Danbooru - cf735b2a59302bf96aac3960c4e075a1_0.jpg",
					IndexID:    9,
					Thumbnail:  "https://img3.saucenao.com/booru/c/f/cf735b2a59302bf96aac3960c4e075a1_0.jpg",
					Similarity: 18.71,
				},
				Data: json.RawMessage(`{
        "source": "http://img10.pixiv.net/img/howard19862002/12897460.jpg",
        "characters": "elis (touhou), kikuri (touhou), konngara, mima, sariel, shingyoku, shingyoku (male), yuugenmagan",
        "material": "highly responsive to prayers, touhou, touhou (pc-98)",
        "creator": "nichimatsu seri",
        "danbooru_id": 736634,
        "ext_urls": [
          "https://danbooru.donmai.us/post/show/736634"
        ]
      }`),
			},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Response mismatch (-want +got):\n%s", diff)
	}
}

func TestSearchResult_AsDanbooru(t *testing.T) {
	t.Parallel()
	r := SearchResult{
		Data: json.RawMessage(`{
        "source": "http://img10.pixiv.net/img/howard19862002/12897460.jpg",
        "characters": "elis (touhou), kikuri (touhou), konngara, mima, sariel, shingyoku, shingyoku (male), yuugenmagan",
        "material": "highly responsive to prayers, touhou, touhou (pc-98)",
        "creator": "nichimatsu seri",
        "danbooru_id": 736634,
        "ext_urls": [
          "https://danbooru.donmai.us/post/show/736634"
        ]
      }`),
	}
	got, err := r.AsDanbooru()
	if err != nil {
		t.Fatal(err)
	}
	want := &DanbooruData{
		CommonData: CommonData{
			ExtURLs: []string{"https://danbooru.donmai.us/post/show/736634"},
		},
		DanbooruID: 736634,
		Source:     "http://img10.pixiv.net/img/howard19862002/12897460.jpg",
		Characters: "elis (touhou), kikuri (touhou), konngara, mima, sariel, shingyoku, shingyoku (male), yuugenmagan",
		Material:   "highly responsive to prayers, touhou, touhou (pc-98)",
		Creator:    "nichimatsu seri",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("data mismatch (-want +got):\n%s", diff)
	}
}

func checkRequestFile(t *testing.T, req *http.Request, want []byte) {
	t.Helper()
	media, params, err := mime.ParseMediaType(req.Header["Content-Type"][0])
	if err != nil {
		t.Error(err)
		return
	}
	if media != "multipart/form-data" {
		t.Errorf("Got media type %v", media)
		return
	}
	r := multipart.NewReader(req.Body, params["boundary"])
	p, err := r.NextPart()
	if err != nil {
		t.Error(err)
		return
	}
	defer p.Close()

	b, err := ioutil.ReadAll(p)
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(b, want) {
		t.Errorf("Got %v; want %v", b, want)
	}
}

type spyTransport struct {
	req  *http.Request
	resp *http.Response
	err  error
}

func newSpyTransport(t *testing.T) spyTransport {
	d, err := os.Open(filepath.Join("testdata", "response.json"))
	if err != nil {
		t.Fatal(err)
	}
	return spyTransport{
		resp: &http.Response{
			StatusCode: 200,
			Body:       d,
		},
	}
}

func (t *spyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.req = req
	return t.resp, t.err
}
