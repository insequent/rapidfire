package main

import (
	"testing"
)

func TestRunner(t *testing.T) {
	t.Log("Creating and starting new test http server")
	s := newTestServer()
	defer s.Stop()

	t.Log("Starting new instance of Runner")
	r := NewRunner(false)
	r.Start(false, 2, 2, "GET", "http://localhost:8080/hello", "")

	// 4 requests from 2 rounds of 2 requests
	if s.count != 4 {
		t.Errorf("Incorrect number of requests made. Expected 4, found %d", s.count)
	}
}
