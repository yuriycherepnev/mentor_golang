package main

import (
	"fmt"
	"github.com/braintree/manners"
	"net/http"
	"os"
	"os/signal"
)

func main() {
	handler := &handler{}
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, os.Kill)
	go listenForShutdown(ch)
	err := manners.ListenAndServe(":8083", handler)
	if err != nil {
		return
	}
}

type handler struct{}

func (h *handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
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

func listenForShutdown(ch <-chan os.Signal) {
	<-ch
	manners.Close()
}
