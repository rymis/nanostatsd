package main

import (
	"strings"
	"encoding/base64"
	"io/ioutil"
	"mime"
	"net/http"
)

type staticWebPage struct {
	Content []byte
	ContentType string
}

var webPages map[string]*staticWebPage = make(map[string]*staticWebPage)

func newStaticWebPage(name, p string) *staticWebPage {
	reader := strings.NewReader(p)
	b64 := base64.NewDecoder(base64.StdEncoding, reader)
	data, err := ioutil.ReadAll(b64)
	if err != nil {
		panic(err.Error())
	}

	mtype := "application/octet-string"
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			m := mime.TypeByExtension(name[i:])
			if m != "" {
				mtype = m
			}
			break
		}
	}

	res := &staticWebPage{data, mtype}
	webPages[name] = res
	return res
}

func (p *staticWebPage) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Add("Content-Type", p.ContentType)
	resp.WriteHeader(http.StatusOK)
	resp.Write(p.Content)
}

type safeHandle struct {
	Url string
	Handle http.Handler
}

func (h safeHandle) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.URL.Path != h.Url {
		resp.Header().Add("Content-Type", "text/plain")
		resp.WriteHeader(http.StatusNotFound)
		resp.Write([]byte("File not found"))
		return
	}
	h.Handle.ServeHTTP(resp, req)
}

func handleStaticPages() {
	for k, p := range(webPages) {
		http.Handle("/" + k, safeHandle{"/" + k, p})
		if k == "index.html" {
			http.Handle("/", safeHandle{"/", p})
		}
	}
}

