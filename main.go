package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// EventSourcePayload type
type EventSourcePayload struct {
	Type        string
	Data        string
	Origin      string
	Source      string
	LastEventID int64
}

func handlerStatic(resp http.ResponseWriter, req *http.Request) {
	// fmt.Println("handlerStatic")
	resp.Header().Set("Access-Control-Allow-Origin", "*")
	resp.Header().Set("Content-Type", "text/html")
	resp.Header().Set("Cache-Control", "no-cache")
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
		updateSpinner(&frame)

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

func handlerReport(resp http.ResponseWriter, req *http.Request) {
	// fmt.Println("handlerReport")
	resp.WriteHeader(http.StatusOK)

	data, _ := ioutil.ReadAll(req.Body)
	defer req.Body.Close()

	var report Report
	json.Unmarshal(data, &report)
	agent := parseAgent(req.UserAgent())
	fmt.Println("Agent: ", agent, " | Buffer: ", report.Bytes, " bytes | Lag: ", report.BrowserTime-report.PayloadTime, " ms")

	resp.Write([]byte("OK"))
}

// Bug type
type Bug struct {
	Error       string `json:"error"`
	Bytes       int    `json:"bytes"`
	BrowserTime int    `json:"browserTime"`
}

func handlerBug(resp http.ResponseWriter, req *http.Request) {
	// fmt.Println("handlerReport")
	resp.WriteHeader(http.StatusOK)

	data, _ := ioutil.ReadAll(req.Body)
	defer req.Body.Close()

	var bug Bug
	json.Unmarshal(data, &bug)
	agent := parseAgent(req.UserAgent())
	fmt.Println("!!!EXCEPTION!!! Agent: ", agent, " | Error: ", bug.Error, "Buffer: ", bug.Bytes, " bytes | Time: ", bug.BrowserTime, " ms")

	resp.Write([]byte("OK"))
}

func parseAgent(userAgent string) string {
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

func updateSpinner(frame *int) {
	print("\b\b", spinner[*frame])
	*frame++
	if *frame >= len(spinner) {
		*frame = 0
	}
}

func extractPacked(executableName string) {
	file, _ := os.Open(executableName)
	defer file.Close()

	// read the packed length
	file.Seek(-10, 2)
	offsetBuffer := make([]byte, 10)
	readLen, readErr := file.Read(offsetBuffer)
	if readLen != 10 || readErr != nil {
		fmt.Println("Failed to read packed length")
		return
	}

	// convert packed length
	packLen, contentErr := strconv.Atoi(string(offsetBuffer))
	if contentErr != nil {
		fmt.Println("Failed to convert packed length")
		return
	}

	// read the packed data
	packOffset := int64((packLen + 10) * -1)
	file.Seek(packOffset, 2)
	packBuffer := make([]byte, packLen)
	readLen, readErr = file.Read(packBuffer)
	if readLen != packLen || readErr != nil {
		fmt.Println("Failed to read packed data")
		return
	}

	// unzip the packed data
	zipReader, zipErr := zip.NewReader(bytes.NewReader(packBuffer), int64(packLen))
	if zipErr != nil {
		fmt.Println("Failed to unzip packed data", zipErr)
		return
	}
	for _, zipFile := range zipReader.File {
		fmt.Println("Extracting: ", zipFile.Name)
		dirPath, _ := filepath.Split(zipFile.Name)
		if len(dirPath) > 0 {
			os.MkdirAll(dirPath, os.ModeDir|0770)
		}
		dest, destErr := os.Create(zipFile.Name)
		if destErr != nil {
			fmt.Println("Failed to extract", zipFile.Name, destErr)
			return
		}
		defer dest.Close()
		src, _ := zipFile.Open()
		defer src.Close()
		io.Copy(dest, src)
	}
}

func main() {
	fmt.Println("IE11 bug proof of concept")

	installPtr := flag.Bool("install", false, "Install packed content")
	flag.Parse()

	if *installPtr {
		extractPacked("main")
	}

	http.Handle("/", http.HandlerFunc(handlerStatic))
	http.Handle("/events", http.HandlerFunc(handlerData))
	http.Handle("/report", http.HandlerFunc(handlerReport))
	http.Handle("/bug", http.HandlerFunc(handlerBug))

	// Start the server and listen forever on port 8000.
	http.ListenAndServe(":8000", nil)
}
