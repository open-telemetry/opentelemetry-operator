package main

import (
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("starting http server")

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Hello")
	})
	if err := http.ListenAndServe(fmt.Sprintf(":%d", 8080), mux); err != nil {
		fmt.Println("error running server:", err)
	}
}
