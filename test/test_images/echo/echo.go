/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func handler(w http.ResponseWriter, r *http.Request) {
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
		return
	}
	if r.URL.Path != "/echo" {
		w.WriteHeader(http.StatusNotFound)
	}
	w.Write([]byte(dump))
}

func main() {
	var useHttp2 bool
	flag.BoolVar(&useHttp2, "http2", false, "Whether we need http2")
	flag.Parse()

	h1s := &http.Server{
		Addr: ":8080",
	}
	if useHttp2 {
		h1s.Handler = h2c.NewHandler(http.HandlerFunc(handler), &http2.Server{})
		log.Print("Echo started, serving HTTP/2")
	} else {
		h1s.Handler = http.HandlerFunc(handler)
		log.Print("Echo started, serving HTTP/1.1")
	}
	log.Fatal(h1s.ListenAndServe())
}
