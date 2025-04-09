package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/TheOTG/httpfromtcp/internal/headers"
	"github.com/TheOTG/httpfromtcp/internal/request"
	"github.com/TheOTG/httpfromtcp/internal/response"
	"github.com/TheOTG/httpfromtcp/internal/server"
)

const port = 42069

func main() {

	server, err := server.Serve(port, handler)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer server.Close()
	log.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}

func handler(w *response.Writer, req *request.Request) {
	if strings.HasPrefix(req.RequestLine.RequestTarget, "/httpbin/") {
		proxyHandler(w, req)
	} else if req.RequestLine.RequestTarget == "/yourproblem" {
		handler400(w, req)
	} else if req.RequestLine.RequestTarget == "/myproblem" {
		handler500(w, req)
	} else if req.RequestLine.RequestTarget == "/video" {
		videoHandler(w, req)
	} else {
		handler200(w, req)
	}
}

func handler400(w *response.Writer, _ *request.Request) {
	w.WriteStatusLine(400)
	body := writeHTML("400 Bad Request", "Bad Request", "Your request honestly kinda sucked.")
	header := response.GetDefaultHeaders(len(body))
	header.Override("Content-Type", "text/html")
	w.WriteHeaders(header)
	w.WriteBody(body)
}

func handler500(w *response.Writer, _ *request.Request) {
	w.WriteStatusLine(500)
	body := writeHTML("500 Internal Server Error", "Internal Server Error", "Okay, you know what? This one is on me.")
	header := response.GetDefaultHeaders(len(body))
	header.Override("Content-Type", "text/html")
	w.WriteHeaders(header)
	w.WriteBody(body)
}

func handler200(w *response.Writer, _ *request.Request) {
	w.WriteStatusLine(200)
	body := writeHTML("200 OK", "Success!", "Your request was an absolute banger.")
	header := response.GetDefaultHeaders(len(body))
	header.Override("Content-Type", "text/html")
	w.WriteHeaders(header)
	w.WriteBody(body)
}

func proxyHandler(w *response.Writer, req *request.Request) {
	resp, err := http.Get(fmt.Sprintf("https://httpbin.org/%s", strings.TrimPrefix(req.RequestLine.RequestTarget, "/httpbin/")))
	if err != nil {
		handler500(w, req)
		return
	}
	defer resp.Body.Close()

	w.WriteStatusLine(200)

	header := response.GetDefaultHeaders(0)
	header.Remove("Content-Length")
	if contentType := resp.Header.Get("Content-Type"); contentType != "" {
		header.Override("Content-Type", contentType)
	}

	header.Set("Transfer-Encoding", "chunked")
	header.Set("Trailer", "X-Content-SHA256, X-Content-Length")

	w.WriteHeaders(header)
	buf := make([]byte, 1024)
	fullBody := make([]byte, 0)
	for {
		n, err := resp.Body.Read(buf)
		fmt.Println("Read", n, "bytes")
		if n > 0 {
			_, err = w.WriteChunkedBody(buf[:n])
			if err != nil {
				fmt.Println("Error writing chunked body:", err)
				break
			}
			fullBody = append(fullBody, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Error reading response body:", err)
			break
		}
	}

	_, err = w.WriteChunkedBodyDone()
	if err != nil {
		fmt.Println("Error writing chunked body done:", err)
	}

	trailers := headers.NewHeaders()
	hash := sha256.Sum256(fullBody)
	hashString := hex.EncodeToString(hash[:])
	trailers.Set("X-Content-SHA256", hashString)
	trailers.Set("X-Content-Length", strconv.Itoa(len(fullBody)))
	err = w.WriteTrailers(trailers)
	if err != nil {
		fmt.Println("Error writing trailers:", err)
	}
}

func videoHandler(w *response.Writer, _ *request.Request) {
	w.WriteStatusLine(200)
	header := response.GetDefaultHeaders(0)
	header.Override("Content-Type", "video/mp4")
	w.WriteHeaders(header)

	file, err := os.ReadFile("assets/vim.mp4")
	if err != nil {
		fmt.Println("unable to read file:", err)
	}
	w.WriteBody(file)
}

func writeHTML(title, bodyTitle, bodyMsg string) []byte {
	return []byte(fmt.Sprintf(`<html>
  <head>
    <title>%s</title>
  </head>
  <body>
    <h1>%s</h1>
    <p>%s</p>
  </body>
</html>`, title, bodyTitle, bodyMsg))
}
