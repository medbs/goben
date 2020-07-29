package core

import (
	"log"
	"os"
	"sync"
)

func Serve(app *Config) {

	if app.Tls && !fileExists(app.TlsKey) {
		log.Printf("key file not found: %s - disabling TLS", app.TlsKey)
		app.Tls = false
	}

	if app.Tls && !fileExists(app.TlsCert) {
		log.Printf("cert file not found: %s - disabling TLS", app.TlsCert)
		app.Tls = false
	}

	var wg sync.WaitGroup

		hh := appendPortIfMissing(app.Listener, app.DefaultPort)
		ListenTCP(app, &wg, hh)
		ListenUDP(app, &wg, hh)


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
