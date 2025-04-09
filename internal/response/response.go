package response

import (
	"fmt"
	"io"
	"strconv"

	"github.com/TheOTG/httpfromtcp/internal/headers"
)

type StatusCode int
type Writer struct {
	writer io.Writer
}

const (
	OK          StatusCode = 200
	BADREQUEST  StatusCode = 400
	SERVERERROR StatusCode = 500
)

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		writer: w,
	}
}

func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
	phrase := ""
	switch statusCode {
	case OK:
		phrase = "HTTP/1.1 200 OK\r\n"
	case BADREQUEST:
		phrase = "HTTP/1.1 400 Bad Request\r\n"
	case SERVERERROR:
		phrase = "HTTP/1.1 500 Internal Server Error\r\n"
	}

	_, err := w.writer.Write([]byte(phrase))
	return err
}

func GetDefaultHeaders(contentLen int) headers.Headers {
	header := headers.NewHeaders()
	header.Set("Content-Length", strconv.Itoa(contentLen))
	header.Set("Connection", "close")
	header.Set("Content-Type", "text/plain")

	return header
}

func (w *Writer) WriteHeaders(header headers.Headers) error {
	for k, v := range header {
		_, err := w.writer.Write([]byte(fmt.Sprintf("%s: %s\r\n", k, v)))
		if err != nil {
			return err
		}
	}
	_, err := w.writer.Write([]byte("\r\n"))
	return err
}

func (w *Writer) WriteBody(p []byte) (int, error) {
	return w.writer.Write(p)
}

func (w *Writer) WriteChunkedBody(p []byte) (int, error) {
	prefix := fmt.Appendf(nil, "%x\r\n", len(p))
	suffix := []byte("\r\n")
	chunk := append(prefix, p...)
	chunk = append(chunk, suffix...)
	return w.writer.Write(chunk)
}

func (w *Writer) WriteChunkedBodyDone() (int, error) {
	return w.writer.Write([]byte("0\r\n"))
}

func (w *Writer) WriteTrailers(h headers.Headers) error {
	for k, v := range h {
		_, err := w.writer.Write(fmt.Appendf(nil, "%s: %v\r\n", k, v))
		if err != nil {
			return err
		}
	}
	_, err := w.writer.Write([]byte("\r\n"))
	return err
}
