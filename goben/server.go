package main

import (
	"log"
	"os"
	"sync"
)

func serve(app *Config) {

	if app.tls && !fileExists(app.tlsKey) {
		log.Printf("key file not found: %s - disabling TLS", app.tlsKey)
		app.tls = false
	}

	if app.tls && !fileExists(app.tlsCert) {
		log.Printf("cert file not found: %s - disabling TLS", app.tlsCert)
		app.tls = false
	}

	var wg sync.WaitGroup

	for _, h := range app.listeners {
		hh := appendPortIfMissing(h, app.defaultPort)
		ListenTCP(app, &wg, hh)
		ListenUDP(app, &wg, hh)
	}

	wg.Wait()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func appendPortIfMissing(host, port string) string {

LOOP:
	for i := len(host) - 1; i >= 0; i-- {
		c := host[i]
		switch c {
		case ']':
			break LOOP
		case ':':
			/*
				if i == len(host)-1 {
					return host[:len(host)-1] + port // drop repeated :
				}
			*/
			return host
		}
	}

	return host + port
}

func protoLabel(isTLS bool) string {
	if isTLS {
		return "TLS"
	}
	return "TCP"
}
