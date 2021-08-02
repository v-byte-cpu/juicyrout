package main

import (
	"flag"
	"log"
	"net/http"
)

// TODO cli args
func main() {
	var port string
	flag.StringVar(&port, "p", "8091", "listening port")
	flag.Parse()
	// TODO static map www.example.com -> mail.com
	// TODO configure transport
	client := &http.Client{}
	conv := NewDomainConverter("host.juicyrout:" + port)
	req := NewRequestProcessor(conv)
	resp := NewResponseProcessor(conv)
	handler := NewProxyHandler(client, req, resp)
	if err := http.ListenAndServeTLS(":"+port, "cert.pem", "key.pem", handler); err != nil && err != http.ErrServerClosed {
		log.Fatalln("listen error: ", err)
	}
}
