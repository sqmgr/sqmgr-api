/*
Copyright 2019 Tom Peters

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"net/url"
	"strconv"
)

const (
	// how many elements to show at beginning and end
	defaultCapBuffer = 1

	// how many elements to show in the current window. Includes current page so it must be >= 1
	defaultWindowBuffer = 5
)

// Pagination is a list of pages that have been paginated
type Pagination struct {
	baseURL      string
	capBuffer    int
	windowBuffer int
	currentPage  int
	total        int
	pages        []int
}

// NewPagination will return a new pagination object
func NewPagination(total, currentPage int) *Pagination {
	return &Pagination{
		capBuffer:    defaultCapBuffer,
		windowBuffer: defaultWindowBuffer,
		total:        total,
		currentPage:  currentPage,
		baseURL:      "#",
	}
}

// Link will return the link to use
func (p *Pagination) Link(page int) string {
	// XXX: will this be a bottleneck???
	u, err := url.Parse(p.baseURL)
	if err != nil {
		panic(err)
	}

	query := u.Query()
	query.Set("page", strconv.Itoa(page))

	u.RawQuery = query.Encode()

	return u.String()
}

// SetBaseURL will set the base URL
func (p *Pagination) SetBaseURL(baseURL string) {
	p.baseURL = baseURL
}

// Total returns the total number of pages
func (p *Pagination) Total() int {
	return p.total
}

// CurrentPage returns the current page
func (p *Pagination) CurrentPage() int {
	return p.currentPage
}

// PrevPage will return the previous page. If at the start, it will return 1
func (p *Pagination) PrevPage() int {
	if p.pages == nil {
		p.build()
	}

	if p.currentPage <= 1 {
		return 1
	}

	return p.currentPage - 1
}

// NextPage will return the next page. If at the end, it will return the last page
func (p *Pagination) NextPage() int {
	if p.pages == nil {
		p.build()
	}

	if p.currentPage >= p.total {
		return p.total
	}

	return p.currentPage + 1
}

// Pages will return an array of pages. A 0 is a placeholder for an ellipses
func (p *Pagination) Pages() []int {
	if p.pages == nil {
		p.build()
	}

	return p.pages
}

// SetCapBuffer will set how many items should appear at the beginning and end
func (p *Pagination) SetCapBuffer(capBuffer int) {
	if p.pages != nil {
		panic("Build() already called")
	}

	p.capBuffer = capBuffer
}

// SetWindowBuffer will determine how many elements (including the current page) should be in the active window
func (p *Pagination) SetWindowBuffer(windowBuffer int) {
	if p.pages != nil {
		panic("Build() already called")
	}

	if windowBuffer < 1 {
		panic("windowBuffer cannot be < 1")
	}

	p.windowBuffer = windowBuffer
}

func (p *Pagination) build() {
	if p.pages != nil {
		panic("Build() already called")
	}

	visible := p.capBuffer*2 + p.windowBuffer + 1

	if p.total <= visible {
		items := make([]int, p.total)
		for i := 0; i < p.total; i++ {
			items[i] = i + 1
		}

		p.pages = items
		return
	}

	items := make([]int, 0, visible+2) // make room for ellipses
	for i := 1; i <= p.capBuffer; i++ {
		items = append(items, i)
	}

	windowEnd := p.currentPage + (p.windowBuffer / 2)
	windowStart := windowEnd - p.windowBuffer + 1

	capRightStart := p.total - p.capBuffer + 1
	capRightEnd := p.total

	if windowStart <= p.capBuffer {
		windowStart = p.capBuffer + 1
		windowEnd = p.capBuffer + p.windowBuffer - 1
	}

	if windowEnd >= capRightStart {
		windowEnd = capRightStart - 1

		if p.total-p.windowBuffer+1 < windowStart {
			windowStart = p.total - p.windowBuffer + 1
		}
	}

	if p.capBuffer+1 < windowStart {
		items = append(items, 0)
	}

	for i := windowStart; i <= windowEnd; i++ {
		items = append(items, i)
	}

	if windowEnd+1 < capRightStart {
		items = append(items, 0)
	}

	for i := capRightStart; i <= capRightEnd; i++ {
		items = append(items, i)
	}

	p.pages = items
}
