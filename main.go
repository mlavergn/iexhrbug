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
	Pack           *gopack.Pack
	bugReports     []BugReport
	browserReports map[string]BrowserReport
	serviceReports map[string]ServiceReport
}

// NewTestCase export
func NewTestCase() *TestCase {
	id := &TestCase{
		bugReports:     []BugReport{},
		browserReports: map[string]BrowserReport{},
		serviceReports: map[string]ServiceReport{},
	}

	pack := gopack.NewPack()
	_, err := pack.Load()
	if err == nil {
		fmt.Println("[Binary packed assets loaded]")
		id.Pack = pack
	}

	return id
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

// ServiceReport type
type ServiceReport struct {
	Bytes      int   `json:"bytes"`
	Payloads   int   `json:"payloads"`
	ReportTime int64 `json:"reportTime"`
}

// NewServiceStatus export
func NewServiceStatus(bytes int, payloads int) ServiceReport {
	return ServiceReport{
		Bytes:      bytes,
		Payloads:   payloads,
		ReportTime: time.Now().UnixNano() / 1000000,
	}
}

func (id *TestCase) handlerEvents(resp http.ResponseWriter, req *http.Request) {
	// fmt.Println("handlerData")
	resp.Header().Set("Access-Control-Allow-Origin", "*")
	resp.Header().Set("Content-Type", "text/event-stream")
	resp.Header().Set("Cache-Control", "no-cache")
	resp.WriteHeader(http.StatusOK)

	flusher, _ := resp.(http.Flusher)

	fmt.Println(req.URL.Query())
	sendBytes, _ := strconv.Atoi(req.URL.Query().Get("sendBytes"))
	pauseAfter, _ := strconv.Atoi(req.URL.Query().Get("pauseAfter"))

	agent := id.parseAgent(req.UserAgent())

	fmt.Printf("Agent %s sending %d bytes, flushing every %d sends\n", agent, sendBytes, pauseAfter)

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
			id.serviceReports[agent] = NewServiceStatus(bytesSent, payloadsSent)
			time.Sleep(1000 * time.Millisecond)
			pause = pauseAfter
		}
	}

	fmt.Printf("Test complete, sent %d / bytes sent %d\n", payloadsSent, bytesSent)
}

// BrowserReport type
type BrowserReport struct {
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

	var report BrowserReport
	json.Unmarshal(data, &report)
	agent := id.parseAgent(req.UserAgent())
	id.browserReports[agent] = report
	fmt.Println("Agent: ", agent, " | Buffer: ", report.Bytes, " bytes | Lag: ", report.BrowserTime-report.PayloadTime, " ms")

	resp.Write([]byte("OK"))
}

// BugReport type
type BugReport struct {
	Error       string `json:"error"`
	Bytes       int    `json:"bytes"`
	BrowserTime int    `json:"browserTime"`
}

func (id *TestCase) handlerBug(resp http.ResponseWriter, req *http.Request) {
	// fmt.Println("handlerReport")
	resp.WriteHeader(http.StatusOK)

	data, _ := ioutil.ReadAll(req.Body)
	defer req.Body.Close()

	var bug BugReport
	json.Unmarshal(data, &bug)
	id.bugReports = append(id.bugReports, bug)
	agent := id.parseAgent(req.UserAgent())
	fmt.Println("!!!EXCEPTION!!! Agent: ", agent, " | Error: ", bug.Error, "Buffer: ", bug.Bytes, " bytes | Time: ", bug.BrowserTime, " ms")

	resp.Write([]byte("OK"))
}

func (id *TestCase) handlerResults(resp http.ResponseWriter, req *http.Request) {
	// fmt.Println("handlerResult")

	bugData, bugErr := json.Marshal(id.bugReports)
	if bugErr != nil {
		resp.Write([]byte("OK"))
		return
	}

	svcData, svcErr := json.Marshal(id.serviceReports)
	if svcErr != nil {
		resp.Write([]byte("OK"))
		return
	}

	progData, progErr := json.Marshal(id.browserReports)
	if progErr != nil {
		resp.Write([]byte("OK"))
		return
	}

	resp.Header().Set("Content-Type", "text/plain")
	resp.WriteHeader(http.StatusOK)
	resp.Write(bugData)
	resp.Write([]byte("\n\n"))
	resp.Write(svcData)
	resp.Write([]byte("\n\n"))
	resp.Write(progData)
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
	port := "8000"
	fmt.Println("IE11 bug proof of concept on port " + port)

	id := NewTestCase()

	http.Handle("/", http.HandlerFunc(id.handlerStatic))
	http.Handle("/events", http.HandlerFunc(id.handlerEvents))
	http.Handle("/report", http.HandlerFunc(id.handlerReport))
	http.Handle("/bug", http.HandlerFunc(id.handlerBug))
	http.Handle("/results", http.HandlerFunc(id.handlerResults))

	// Start the server and listen forever on port 8000.
	// http.ListenAndServe(":8000", nil)

	cert, _ := id.Pack.File("iexhr.crt")
	defer os.Remove(*cert)
	key, _ := id.Pack.File("iexhr.key")
	defer os.Remove(*key)
	err := http.ListenAndServeTLS(":"+port, *cert, *key, nil)
	if err != nil {
		fmt.Println("Failed to start listener", err)
	}
}
