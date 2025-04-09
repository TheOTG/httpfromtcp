package request

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/TheOTG/httpfromtcp/internal/headers"
)

type requestState int

type Request struct {
	RequestLine    RequestLine
	Headers        headers.Headers
	Body           []byte
	state          requestState
	bodyLengthRead int
}

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

const (
	INITIALIZED requestState = iota
	PARSINGHEADERS
	PARSINGBODY
	DONE
)
const bufferSize = 8

var IsCapitalLetters = regexp.MustCompile(`^[A-Z]+$`).MatchString

func RequestFromReader(reader io.Reader) (*Request, error) {
	buf := make([]byte, bufferSize)
	readToIndex := 0

	request := Request{
		RequestLine: RequestLine{
			HttpVersion:   "",
			RequestTarget: "",
			Method:        "",
		},
		Headers: headers.NewHeaders(),
		state:   INITIALIZED,
	}

	for request.state != DONE {
		if len(buf) == readToIndex {
			newBuf := make([]byte, len(buf)*2)
			copy(newBuf, buf)
			buf = newBuf
		}
		nBytesRead, err := reader.Read(buf[readToIndex:])
		if err != nil && err != io.EOF {
			return nil, err
		}
		if err == io.EOF {
			if request.state != DONE {
				return nil, errors.New("unexpected end of stream while parsing")
			}
			break
		}

		readToIndex += nBytesRead
		nBytesParsed, err := request.parse(buf[:readToIndex])
		if err != nil {
			return nil, err
		}

		if nBytesParsed > 0 {
			copy(buf, buf[nBytesParsed:readToIndex])
			readToIndex -= nBytesParsed
		}
	}

	return &request, nil
}

func parseRequestLine(data []byte) (RequestLine, int, error) {
	endIndex := bytes.Index(data, []byte("\r\n"))
	if endIndex == -1 {
		return RequestLine{}, 0, nil
	}
	parsed := data[:endIndex]
	consumed := endIndex + 2
	reqStr := string(parsed)

	splitReqLine := strings.Split(reqStr, " ")
	if len(splitReqLine) != 3 {
		return RequestLine{}, 0, errors.New("invalid request")
	}

	httpVersion := strings.TrimPrefix(splitReqLine[2], "HTTP/")
	requestTarget := splitReqLine[1]
	method := splitReqLine[0]

	if !IsCapitalLetters(method) {
		return RequestLine{}, 0, errors.New("invalid method")
	}

	if httpVersion != "1.1" {
		return RequestLine{}, 0, errors.New("invalid http version")
	}

	requestLine := RequestLine{
		HttpVersion:   httpVersion,
		RequestTarget: requestTarget,
		Method:        method,
	}

	return requestLine, consumed, nil
}

func (r *Request) parse(data []byte) (int, error) {
	totalBytesParsed := 0
	for r.state != DONE {
		n, err := r.parseSingle(data[totalBytesParsed:])
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return totalBytesParsed, nil
		}

		totalBytesParsed += n
	}
	return totalBytesParsed, nil
}

func (r *Request) parseSingle(data []byte) (int, error) {
	switch r.state {
	case INITIALIZED:
		requestLine, n, err := parseRequestLine(data)
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, nil
		}
		r.RequestLine.HttpVersion = requestLine.HttpVersion
		r.RequestLine.RequestTarget = requestLine.RequestTarget
		r.RequestLine.Method = requestLine.Method
		r.state = PARSINGHEADERS

		return n, nil
	case PARSINGHEADERS:
		n, done, err := r.Headers.Parse(data)
		if err != nil {
			return 0, err
		}
		if done {
			r.state = PARSINGBODY
		}
		return n, nil
	case PARSINGBODY:
		v, ok := r.Headers.Get("Content-Length")
		if !ok {
			r.state = DONE
			return len(data), nil
		}

		contentLength, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("malformed Content-Length: %s", err)
		}

		r.Body = append(r.Body, data...)
		r.bodyLengthRead += len(data)
		if r.bodyLengthRead > contentLength {
			return 0, errors.New("Content-Length too large")
		} else if r.bodyLengthRead == contentLength {
			r.state = DONE
		}

		return len(data), nil
	case DONE:
		return 0, errors.New("trying to read data in a done state")
	default:
		return 0, errors.New("unknown state")
	}
}
