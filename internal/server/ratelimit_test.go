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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/onsi/gomega"
	"golang.org/x/time/rate"
)

func TestNewRateLimiter(t *testing.T) {
	g := gomega.NewWithT(t)

	rl := NewRateLimiter(rate.Limit(10), 5)

	g.Expect(rl).ShouldNot(gomega.BeNil())
	g.Expect(rl.rate).Should(gomega.Equal(rate.Limit(10)))
	g.Expect(rl.burst).Should(gomega.Equal(5))
	g.Expect(rl.visitors).ShouldNot(gomega.BeNil())
}

func TestGetVisitor_NewVisitor(t *testing.T) {
	g := gomega.NewWithT(t)

	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate.Limit(10),
		burst:    5,
	}

	limiter := rl.getVisitor("192.168.1.1")

	g.Expect(limiter).ShouldNot(gomega.BeNil())
	g.Expect(rl.visitors).Should(gomega.HaveLen(1))
	g.Expect(rl.visitors["192.168.1.1"]).ShouldNot(gomega.BeNil())
}

func TestGetVisitor_ExistingVisitor(t *testing.T) {
	g := gomega.NewWithT(t)

	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate.Limit(10),
		burst:    5,
	}

	// First call creates visitor
	limiter1 := rl.getVisitor("192.168.1.1")
	// Second call should return the same limiter
	limiter2 := rl.getVisitor("192.168.1.1")

	g.Expect(limiter1).Should(gomega.BeIdenticalTo(limiter2))
	g.Expect(rl.visitors).Should(gomega.HaveLen(1))
}

func TestGetVisitor_MultipleVisitors(t *testing.T) {
	g := gomega.NewWithT(t)

	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate.Limit(10),
		burst:    5,
	}

	rl.getVisitor("192.168.1.1")
	rl.getVisitor("192.168.1.2")
	rl.getVisitor("10.0.0.1")

	g.Expect(rl.visitors).Should(gomega.HaveLen(3))
}

func TestGetIP_WithPort(t *testing.T) {
	g := gomega.NewWithT(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	ip := getIP(req)

	g.Expect(ip).Should(gomega.Equal("192.168.1.1"))
}

func TestGetIP_WithoutPort(t *testing.T) {
	g := gomega.NewWithT(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1"

	ip := getIP(req)

	g.Expect(ip).Should(gomega.Equal("192.168.1.1"))
}

func TestGetIP_IPv6WithPort(t *testing.T) {
	g := gomega.NewWithT(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "[::1]:12345"

	ip := getIP(req)

	g.Expect(ip).Should(gomega.Equal("::1"))
}

func TestGetIP_IPv6WithoutPort(t *testing.T) {
	g := gomega.NewWithT(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "::1"

	ip := getIP(req)

	g.Expect(ip).Should(gomega.Equal("::1"))
}

func TestLimit_AllowsRequestsUnderLimit(t *testing.T) {
	g := gomega.NewWithT(t)

	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate.Limit(10),
		burst:    5,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	rl.Limit(handler).ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))
}

func TestLimit_BlocksRequestsOverLimit(t *testing.T) {
	g := gomega.NewWithT(t)

	// Create a rate limiter with very low limits
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate.Limit(0.001), // Very low rate
		burst:    1,                 // Only 1 request allowed initially
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// First request should succeed
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	rec1 := httptest.NewRecorder()
	rl.Limit(handler).ServeHTTP(rec1, req1)
	g.Expect(rec1.Code).Should(gomega.Equal(http.StatusOK))

	// Second request should be rate limited
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	rec2 := httptest.NewRecorder()
	rl.Limit(handler).ServeHTTP(rec2, req2)
	g.Expect(rec2.Code).Should(gomega.Equal(http.StatusTooManyRequests))
}

func TestLimit_DifferentIPsHaveSeparateLimits(t *testing.T) {
	g := gomega.NewWithT(t)

	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate.Limit(0.001),
		burst:    1,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// First IP - first request
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	rec1 := httptest.NewRecorder()
	rl.Limit(handler).ServeHTTP(rec1, req1)
	g.Expect(rec1.Code).Should(gomega.Equal(http.StatusOK))

	// First IP - second request (should be blocked)
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	rec2 := httptest.NewRecorder()
	rl.Limit(handler).ServeHTTP(rec2, req2)
	g.Expect(rec2.Code).Should(gomega.Equal(http.StatusTooManyRequests))

	// Second IP - first request (should succeed)
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.RemoteAddr = "192.168.1.2:12345"
	rec3 := httptest.NewRecorder()
	rl.Limit(handler).ServeHTTP(rec3, req3)
	g.Expect(rec3.Code).Should(gomega.Equal(http.StatusOK))
}
