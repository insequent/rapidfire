package main

import (
	"flag"
	"fmt"
	"os"
)

// Main
func main() {
	duration := flag.Int("t", 0, "Length of time in seconds to run the queries. 0 for indefinite")
	method := flag.String("m", "GET", "The HTTP method to use with the request")
	payload := flag.String("d", "", "Data (Body) to send with the request")
	rps := flag.Int("r", 1, "Number of request to make each second")
	url := flag.String("u", "", "URL to make requests against")
	verbose := flag.Bool("v", false, "Enables verbose logging")
	flag.Parse()

	if *url == "" {
		fmt.Println("ERROR: A url is required")
		os.Exit(1)
	}

	runner := NewRunner(*verbose)
	runner.Start(*duration, *rps, *method, *url, *payload)
}
