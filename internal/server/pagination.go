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

const (
	// how many elements to show at beginning and end
	paginationCapBuffer = 1

	// how many elements to show in the current window. Includes current page so it must be >= 1
	paginationWindowBuffer = 5
)

// Pagination is a list of pages that have been paginated
type Pagination []int

// PaginationBuilder will build a new pagination array
type PaginationBuilder struct {
	// CapBuffer is the number of items to show at beginning and end of pagination
	CapBuffer int
	// WindowBuffer is the size of the window
	// FIXME - change to match description
	WindowBuffer int
	currentPage  int
	pages        int
}

// NewPaginationBuilder will return a new pagination builder
func NewPaginationBuilder(currentPage, pages int) *PaginationBuilder {
	return &PaginationBuilder{
		CapBuffer:    paginationCapBuffer,
		WindowBuffer: paginationWindowBuffer,
		currentPage:  currentPage,
		pages:        pages,
	}
}

// Build will return a Pagination using the custom builder properties
func (b *PaginationBuilder) Build() Pagination {
	visible := b.CapBuffer*2 + b.WindowBuffer + 1

	if b.pages <= visible {
		items := make(Pagination, b.pages)
		for i := 0; i < b.pages; i++ {
			items[i] = i + 1
		}

		return items
	}

	items := make(Pagination, 0, visible+2) // make room for ellipses
	for i := 1; i <= b.CapBuffer; i++ {
		items = append(items, i)
	}

	windowEnd := b.currentPage + (b.WindowBuffer / 2)
	windowStart := windowEnd - b.WindowBuffer + 1

	capRightStart := b.pages - b.CapBuffer + 1
	capRightEnd := b.pages

	if windowStart <= b.CapBuffer {
		windowStart = b.CapBuffer + 1
		windowEnd = b.CapBuffer + b.WindowBuffer - 1
	}

	if windowEnd >= capRightStart {
		windowEnd = capRightStart - 1

		if b.pages-b.WindowBuffer+1 < windowStart {
			windowStart = b.pages - b.WindowBuffer + 1
		}
	}

	if b.CapBuffer+1 < windowStart {
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

	return items
}

// DefaultPagination creates a new Pagination using default configuration
func DefaultPagination(currentPage, pages int) Pagination {
	b := NewPaginationBuilder(currentPage, pages)
	return b.Build()
}
