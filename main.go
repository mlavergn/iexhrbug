package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// EventSourcePayload type
type EventSourcePayload struct {
	Type        string
	Data        []byte
	Origin      string
	LastEventID int
	Source      string
}

func handlerStatic(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusOK)

	file, err := os.Open("index.html")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	_, err = io.Copy(resp, file)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func handlerData(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusOK)

	payload := &EventSourcePayload{
		Type:        "message",
		Data:        []byte("{\"content\": \"The quick brown fox jumped over the lazy dog.\""),
		Origin:      "localhost",
		LastEventID: int(time.Now().UTC().Unix() * 1000),
		Source:      "iexhrbug",
	}

	sent := 0
	// for sent < 10000000 {
	for sent < 5000000 {
		data, err := json.Marshal(payload)
		if err != nil {
			fmt.Println(err)
			return
		}

		sent = sent + len(data)

		resp.Write(data)
		fmt.Printf("%d bytes sent\n", sent)
	}
}

func main() {
	fmt.Println("IE11 bug proof of concept")

	http.Handle("/", http.HandlerFunc(handlerStatic))
	http.Handle("/data", http.HandlerFunc(handlerData))

	// Start the server and listen forever on port 8000.
	http.ListenAndServe(":8000", nil)
}
