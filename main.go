package main

import (
	"fmt"
	"io"
	"net/http"

	"crypto/tls"
	"strconv"
	"strings"
	"syscall/js"
	"time"

	"github.com/gokitx/pkgs/limiter"
)

var (
	client = http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: 1000 * time.Millisecond,
	}
)

type Scaner struct {
	Host string
	c    *http.Client
}

func NewScaner(target string, timeout time.Duration) *Scaner {
	return &Scaner{
		Host: target,
		c: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
			Timeout: timeout,
		},
	}
}

func (s *Scaner) probe(port int) bool {
	target := fmt.Sprintf("http://%s", fmt.Sprintf("%s:%d", s.Host, port))
	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		return false
	}
	req.Header.Add("js.fetch:mode", "no-cors")

	resp, err := s.c.Do(req)
	if err != nil {
		// TODO: Get more exception strings for major browsers
		errs := strings.ToLower(err.Error())
		if strings.Contains(errs, "exceeded while awaiting") ||
			strings.Contains(errs, "ssl") ||
			strings.Contains(errs, "cors") ||
			strings.Contains(errs, "invalid") ||
			strings.Contains(errs, "protocol") {
			return true
		} else {
			return false
		}
	}

	defer func() {
		if resp.Body != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}()

	return true
}

func (s *Scaner) Scan(start, end int) map[int]struct{} {
	res := make(map[int]struct{})
	l := limiter.New(100)

	for port := start; port <= end; port++ {
		l.Allow()
		go func(port int) {
			defer l.Done()
			if open := s.probe(port); open {
				res[port] = struct{}{}
			}
		}(port)
	}

	l.Wait()
	return res
}

func main() {
	document := js.Global().Get("document")
	documentTitle := document.Call("createElement", "h1")
	documentTitle.Set("innerText", "WebAssembly TCP Port Scanner")
	document.Get("body").Call("appendChild", documentTitle)
	placeHolder := document.Call("createElement", "h3")
	placeHolder.Set("innerText", "Scanning...")
	document.Get("body").Call("appendChild", placeHolder)

	s := NewScaner("127.0.0.1", 1000*time.Millisecond)
	res := s.Scan(8000, 9000)

	placeHolder.Set("innerText", "Open Ports:")

	for k := range res {
		portString := strconv.Itoa(k)
		openPortsParagraph := document.Call("createElement", "li")
		openPortsParagraph.Set("innerText", portString)
		document.Get("body").Call("appendChild", openPortsParagraph)
	}
}
