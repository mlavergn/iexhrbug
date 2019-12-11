package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

// EventSourcePayload type
type EventSourcePayload struct {
	Type        string
	Data        string
	Origin      string
	Source      string
	LastEventID int
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

	flusher, _ := resp.(http.Flusher)

	fmt.Println(req.URL.Query())
	payloads, _ := strconv.Atoi(req.URL.Query().Get("payloads"))
	pauseAt, _ := strconv.Atoi(req.URL.Query().Get("pauseAt"))

	fmt.Printf("Sending %d payloads, pausing at %d\n", payloads, pauseAt)

	payload := &EventSourcePayload{
		Type:        "message",
		Data:        "{\"content\": \"The quick brown fox jumped over the lazy dog.\"}",
		Origin:      "localhost",
		LastEventID: 0,
		Source:      "iexhrbug",
	}

	bytesSent := 0
	payloadsSent := 0
	pause := pauseAt
	for payloadsSent < payloads {
		payload.LastEventID++

		raw := fmt.Sprintf("type:%s\ndata:%s\norigin:%s\nlastEventId:%d\nsource:%s\n\n", payload.Type, payload.Data, payload.Origin, payload.LastEventID, payload.Source)
		data := []byte(raw)

		bytesSent += len(data)
		payloadsSent++

		resp.Write(data)
		flusher.Flush()

		pause--
		if pause == 0 {
			fmt.Printf("Payloads sent %d / bytes sent %d\n", payloadsSent, bytesSent)
			time.Sleep(1 * time.Second)
			pause = pauseAt
		}
	}
}

// Report type
type Report struct {
	Payloads int `json:"payloads"`
	Bytes    int `json:"bytes"`
	Chunks   int `json:"chunks"`
}

func handlerReport(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusOK)

	data, _ := ioutil.ReadAll(req.Body)
	defer req.Body.Close()

	fmt.Println(string(data))
	var report Report
	json.Unmarshal(data, &report)
	fmt.Println("Report: ", report)

	resp.Write([]byte("OK"))
}

func main() {
	fmt.Println("IE11 bug proof of concept")

	http.Handle("/", http.HandlerFunc(handlerStatic))
	http.Handle("/events", http.HandlerFunc(handlerData))
	http.Handle("/report", http.HandlerFunc(handlerReport))

	// Start the server and listen forever on port 8000.
	http.ListenAndServe(":8000", nil)
}
