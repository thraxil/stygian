package main

import (
	"bytes"
	"fmt"
	"github.com/elazarl/goproxy"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
)

var host_blacklist = []*regexp.Regexp{
	regexp.MustCompile("localhost.*"),
	regexp.MustCompile("127.0.0.1"),
	regexp.MustCompile(".*thraxil.org"),
	regexp.MustCompile(".*doubleclick.net"),
	regexp.MustCompile(".*google-analytics.com"),
	regexp.MustCompile(".*ccnmtl.columbia.edu"),
	regexp.MustCompile(".*pagead\\d+.googlesyndication.com"),
	regexp.MustCompile(".*adnxs.com"),
	regexp.MustCompile(".*serving-sys.com"),
	regexp.MustCompile("skimresources.com"),
	regexp.MustCompile(".*facebook.com"),
	regexp.MustCompile(".*gravatar.com"),
	regexp.MustCompile("mint.com"),
	regexp.MustCompile("chase.com"),
	regexp.MustCompile("ingdirect.com"),
}

var path_suffix_blacklist = []string{
	".ico",
	".jpg",
	".jpeg",
	".png",
	".gif",
	".css",
	".js",
	".flv",
	".woff",
	"crossdomain.xml",
	"ad_iframe.html",
}

type BodyHandler struct {
	R    io.ReadCloser
	W    *bytes.Buffer
	Resp *http.Response
}

func (c *BodyHandler) Read(b []byte) (n int, err error) {
	n, err = c.R.Read(b)
	if n > 0 {
		if n, err := c.W.Write(b[:n]); err != nil {
			return n, err
		}
	}
	return
}

func (c *BodyHandler) Close() error {
	contentType := c.Resp.Header.Get("Content-Type")
	content := c.W.String()
	fmt.Println(content)
	fmt.Println("--- 200", contentType, c.Resp.ContentLength, c.Resp.Request.URL, "---")
	return c.R.Close()
}

func filter(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	// ignore non 200/304s
	if resp.StatusCode != 200 && resp.StatusCode != 304 {
		return resp
	}

	for _, suffix := range path_suffix_blacklist {
		if strings.HasSuffix(resp.Request.URL.Path, suffix) {
			return resp
		}
	}

	for _, host := range host_blacklist {
		if host.MatchString(resp.Request.URL.Host) {
			return resp
		}
	}

	if resp.StatusCode == 200 {
		contentType := resp.Header.Get("Content-Type")
		if strings.HasPrefix(contentType, "text/") {
			if strings.HasPrefix(contentType, "text/css") ||
				strings.HasPrefix(contentType, "text/javascript") ||
				strings.HasPrefix(contentType, "text/json") {
				return resp
			}
			// full response with a body. save it, index it, etc.

			length := resp.ContentLength
			if length <= 0 {
				length = 1024
			}
			buf := bytes.NewBuffer(make([]byte, 0, length))
			resp.Body = &BodyHandler{resp.Body, buf, resp}
			// log()
			// save()
			// index()
		}
	} else if resp.StatusCode == 304 {
		// just a 304. don't need to re-index or save
		// just log it. How to filter out only text/* types though?
		// 304s don't include Content-Type headers...
		fmt.Println("304", resp.Request.URL)
		// log()
	}
	return resp
}

func main() {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = false
	proxy.OnResponse().DoFunc(filter)
	log.Fatal(http.ListenAndServe(":8080", proxy))
}
