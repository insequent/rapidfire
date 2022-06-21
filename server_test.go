package main

import (
	"fmt"
	"net/http"
)

type testServer struct {
	server *http.Server
	count  int
}

func newTestServer() *testServer {
	s := &testServer{
		count:  0,
		server: &http.Server{Addr: ":8080"},
	}

	http.HandleFunc("/hello", func(w http.ResponseWriter, req *http.Request) {
		s.count++
		fmt.Fprintf(w, "hello!\n")
	})

	s.Start()
	return s
}

func (s *testServer) Start() {
	go s.server.ListenAndServe()
}

func (s *testServer) Stop() error {
	return s.server.Close()
}
