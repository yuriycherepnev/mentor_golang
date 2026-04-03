package main

import (
	"fmt"
	"net/http"
	"strings"
)

func main() {
	http.HandleFunc("/hello", hello)
	http.HandleFunc("/goodbye/", goodbye)
	http.HandleFunc("/", homePage)
	err := http.ListenAndServe(":8082", nil)
	if err != nil {
		return
	}
}

func hello(res http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	name := query.Get("name")
	if name == "" {
		name = "Ivan Ivanovich"
	}
	_, err := fmt.Fprint(res, "Hello, my name is ", name)
	if err != nil {
		return
	}
}

func goodbye(res http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	parts := strings.Split(path, "/")
	name := parts[2]
	if name == "" {
		name = "Ivan Ivanovich"
	}
	_, err := fmt.Fprint(res, "Goodbye ", name)
	if err != nil {
		return
	}
}

func homePage(res http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(res, req)
		return
	}
	_, err := fmt.Fprint(res, "The homepage.")
	if err != nil {
		return
	}
}
