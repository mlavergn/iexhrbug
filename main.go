package main

import (
	"encoding/json"
	"fmt"
	gopack "github.com/mlavergn/gopack/src/gopack"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// TestCase export
type TestCase struct {
	Pack *gopack.Pack
}

// EventSourcePayload type
type EventSourcePayload struct {
	Type        string
	Data        string
	Origin      string
	Source      string
	LastEventID int64
}

func (id *TestCase) handlerStatic(resp http.ResponseWriter, req *http.Request) {
	// fmt.Println("handlerStatic")
	resp.Header().Set("Access-Control-Allow-Origin", "*")
	resp.Header().Set("Content-Type", "text/html")
	resp.Header().Set("Cache-Control", "no-cache")
	resp.WriteHeader(http.StatusOK)

	var file io.Reader
	var err error
	if id.Pack != nil {
		file, err = id.Pack.Pipe("index.html")
	} else {
		ffile, ferr := os.Open("index.html")
		defer ffile.Close()
		file = ffile
		err = ferr
	}
	if err != nil {
		fmt.Println(err)
		return
	}

	_, err = io.Copy(resp, file)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func (id *TestCase) handlerData(resp http.ResponseWriter, req *http.Request) {
	// fmt.Println("handlerData")
	resp.Header().Set("Access-Control-Allow-Origin", "*")
	resp.Header().Set("Content-Type", "text/event-stream")
	resp.Header().Set("Cache-Control", "no-cache")
	resp.WriteHeader(http.StatusOK)

	flusher, _ := resp.(http.Flusher)

	fmt.Println(req.URL.Query())
	sendBytes, _ := strconv.Atoi(req.URL.Query().Get("sendBytes"))
	pauseAfter, _ := strconv.Atoi(req.URL.Query().Get("pauseAfter"))

	fmt.Printf("Sending %d bytes, flushing every %d sends\n", sendBytes, pauseAfter)

	payload := &EventSourcePayload{
		Type:        "message",
		Data:        "{\"content\": \"The quick brown fox jumped over the lazy dog.\"}",
		Origin:      "localhost",
		LastEventID: time.Now().UnixNano() / 1000000,
		Source:      "iexhrbug",
	}

	// agent := parseAgent(req.UserAgent())

	frame := 1
	print(spinner[0])

	bytesSent := 0
	payloadsSent := 0
	pause := pauseAfter
	for bytesSent < sendBytes {
		select {
		case <-req.Context().Done():
			return
		default:
			break
		}

		payload.LastEventID = time.Now().UnixNano() / 1000000
		raw := fmt.Sprintf("type:%s\ndata:%s\norigin:%s\nlastEventId:%d\nsource:%s\n\n", payload.Type, payload.Data, payload.Origin, payload.LastEventID, payload.Source)
		data := []byte(raw)

		payloadLen := len(data)
		bytesSent += payloadLen
		payloadsSent++

		resp.Write(data)
		time.Sleep(10 * time.Millisecond)
		flusher.Flush()
		id.updateSpinner(&frame)

		pause -= payloadLen
		if pause <= 0 {
			// fmt.Printf("Agent [%s] payloads sent %d / bytes sent %d\n", agent, payloadsSent, bytesSent)
			time.Sleep(1000 * time.Millisecond)
			pause = pauseAfter
		}
	}

	fmt.Printf("Test complete, sent %d / bytes sent %d\n", payloadsSent, bytesSent)
}

// Report type
type Report struct {
	Bytes       int `json:"bytes"`
	Chunks      int `json:"chunks"`
	PayloadTime int `json:"payloadTime"`
	BrowserTime int `json:"browserTime"`
}

func (id *TestCase) handlerReport(resp http.ResponseWriter, req *http.Request) {
	// fmt.Println("handlerReport")
	resp.WriteHeader(http.StatusOK)

	data, _ := ioutil.ReadAll(req.Body)
	defer req.Body.Close()

	var report Report
	json.Unmarshal(data, &report)
	agent := id.parseAgent(req.UserAgent())
	fmt.Println("Agent: ", agent, " | Buffer: ", report.Bytes, " bytes | Lag: ", report.BrowserTime-report.PayloadTime, " ms")

	resp.Write([]byte("OK"))
}

// Bug type
type Bug struct {
	Error       string `json:"error"`
	Bytes       int    `json:"bytes"`
	BrowserTime int    `json:"browserTime"`
}

func (id *TestCase) handlerBug(resp http.ResponseWriter, req *http.Request) {
	// fmt.Println("handlerReport")
	resp.WriteHeader(http.StatusOK)

	data, _ := ioutil.ReadAll(req.Body)
	defer req.Body.Close()

	var bug Bug
	json.Unmarshal(data, &bug)
	agent := id.parseAgent(req.UserAgent())
	fmt.Println("!!!EXCEPTION!!! Agent: ", agent, " | Error: ", bug.Error, "Buffer: ", bug.Bytes, " bytes | Time: ", bug.BrowserTime, " ms")

	resp.Write([]byte("OK"))
}

func (id *TestCase) parseAgent(userAgent string) string {
	moz13 := userAgent[13:]
	if strings.Contains(moz13, "Trident/") {
		return "IE"
	}
	if strings.Contains(moz13, "Edge/") {
		return "Edge"
	}
	if strings.Contains(moz13, "Chrome/") {
		return "Chrome"
	}
	if strings.Contains(moz13, "Safari/") {
		return "Safari"
	}
	return moz13[strings.LastIndex(moz13, " ")+1:]
}

var spinner = []string{"◒", "◑", "◓", "◐"}

func (id *TestCase) updateSpinner(frame *int) {
	print("\b\b", spinner[*frame])
	*frame++
	if *frame >= len(spinner) {
		*frame = 0
	}
}

func main() {
	fmt.Println("IE11 bug proof of concept")

	tc := &TestCase{}

	pack := gopack.NewPack()
	_, err := pack.Load()
	if err == nil {
		fmt.Println("Found packed assets")
		tc.Pack = pack
	}

	http.Handle("/", http.HandlerFunc(tc.handlerStatic))
	http.Handle("/events", http.HandlerFunc(tc.handlerData))
	http.Handle("/report", http.HandlerFunc(tc.handlerReport))
	http.Handle("/bug", http.HandlerFunc(tc.handlerBug))

	// Start the server and listen forever on port 8000.
	// http.ListenAndServe(":8000", nil)

	cert, _ := pack.File("iexhr.crt")
	defer os.Remove(*cert)
	key, _ := pack.File("iexhr.key")
	defer os.Remove(*key)
	err = http.ListenAndServeTLS(":8000", *cert, *key, nil)
	if err != nil {
		fmt.Println("Failed to start listener", err)
	}
}
