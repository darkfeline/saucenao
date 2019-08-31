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

// CommonData is the result data common to all indexes.
type CommonData struct {
	ExtURLs []string `json:"ext_urls"`
}

// DanbooruData is the result data for the Danbooru index.
type DanbooruData struct {
	CommonData
	DanbooruID int    `json:"danbooru_id"`
	Source     string `json:"source"`
	Characters string `json:"characters"`
	Material   string `json:"material"`
	Creator    string `json:"creator"`
}
