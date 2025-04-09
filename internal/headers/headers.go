package headers

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
)

type Headers map[string]string

func NewHeaders() Headers {
	headers := Headers{}
	return headers
}

func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	idx := bytes.Index(data, []byte("\r\n"))
	if idx == -1 {
		return 0, false, nil
	}
	if idx == 0 {
		return 2, true, nil
	}

	before, after, found := bytes.Cut(data[:idx], []byte(":"))
	if !found {
		return 0, false, errors.New("invalid header field")
	}

	key := strings.ToLower(strings.TrimLeft(string(before), " "))
	if len(key) == 0 {
		return 0, false, errors.New("invalid header key")
	}
	if key != strings.TrimRight(key, " ") {
		return 0, false, errors.New("invalid header key")
	}

	match, err := regexp.MatchString("[^a-zA-Z0-9!#$%'*+-.^_`|~]", key)
	if err != nil {
		return 0, false, errors.New("invalid regex")
	}
	if match {
		return 0, false, errors.New("invalid character in header key")
	}

	val := strings.TrimSpace(string(after))

	h.Set(key, val)
	return idx + 2, false, nil
}

func (h Headers) Set(key, value string) {
	key = strings.ToLower(key)
	v, ok := h[key]
	if ok {
		value = strings.Join([]string{
			v,
			value,
		}, ", ")
	}
	h[key] = value
}

func (h Headers) Get(key string) (string, bool) {
	key = strings.ToLower(key)
	v, ok := h[key]
	return v, ok
}

func (h Headers) Override(key, value string) {
	key = strings.ToLower(key)
	h[key] = value
}

func (h Headers) Remove(key string) {
	key = strings.ToLower(key)
	delete(h, key)
}
