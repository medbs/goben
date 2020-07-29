package core

import (
	"crypto/tls"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

func ListenTCP(app *Config, wg *sync.WaitGroup, h string) {
	log.Printf("listenTCP: TLS=%v spawning TCP listener: %s", app.Tls, h)

	// first try TLS
	if app.Tls {
		listener, errTLS := listenTLS(app, h)
		if errTLS == nil {
			spawnAcceptLoopTCP(app, wg, listener, true)
			return
		}
		log.Printf("listenTLS: %v", errTLS)
		// TLS failed, try plain TCP
	}

	listener, errListen := net.Listen("tcp", h)
	if errListen != nil {
		log.Printf("listenTCP: TLS=%v %s: %v", app.Tls, h, errListen)
		return
	}
	spawnAcceptLoopTCP(app, wg, listener, false)
}

func spawnAcceptLoopTCP(app *Config, wg *sync.WaitGroup, listener net.Listener, isTLS bool) {
	wg.Add(1)
	go handleTCP(app, wg, listener, isTLS)
}

func listenTLS(app *Config, h string) (net.Listener, error) {
	cert, errCert := tls.LoadX509KeyPair(app.TlsCert, app.TlsKey)
	if errCert != nil {
		log.Printf("listenTLS: failure loading TLS key pair: %v", errCert)
		app.Tls = false // disable TLS
		return nil, errCert
	}

	config := &tls.Config{Certificates: []tls.Certificate{cert}}
	listener, errListen := tls.Listen("tcp", h, config)
	return listener, errListen
}

func handleTCP(app *Config, wg *sync.WaitGroup, listener net.Listener, isTLS bool) {
	defer wg.Done()

	var id int

	var aggReader aggregate
	var aggWriter aggregate

	for {
		conn, errAccept := listener.Accept()
		if errAccept != nil {
			log.Printf("handle: accept: %v", errAccept)
			break
		}
		go handleConnection(conn, id, 0, isTLS, &aggReader, &aggWriter)
		id++
	}
}

func handleConnection(conn net.Conn, c, connections int, isTLS bool, aggReader, aggWriter *aggregate) {
	defer conn.Close()

	log.Printf("handleConnection: incoming: %s %v", protoLabel(isTLS), conn.RemoteAddr())

	// receive options
	var opt Options
	dec := gob.NewDecoder(conn)
	if errOpt := dec.Decode(&opt); errOpt != nil {
		log.Printf("handleConnection: options failure: %v", errOpt)
		return
	}
	log.Printf("handleConnection: options received: %v", opt)

	// send ack
	a := newAck()
	if errAck := ackSend(false, conn, a); errAck != nil {
		log.Printf("handleConnection: sending ack: %v", errAck)
		return
	}

	go serverReader(conn, opt, c, connections, isTLS, aggReader)

	if !opt.PassiveServer {
		go serverWriter(conn, opt, c, connections, isTLS, aggWriter)
	}

	tickerPeriod := time.NewTimer(opt.TotalDuration)

	<-tickerPeriod.C
	log.Printf("handleConnection: %v timer", opt.TotalDuration)

	tickerPeriod.Stop()

	log.Printf("handleConnection: closing: %v", conn.RemoteAddr())
}

func serverReader(conn net.Conn, opt Options, c, connections int, isTLS bool, agg *aggregate) {

	log.Printf("serverReader: starting: %s %v", protoLabel(isTLS), conn.RemoteAddr())

	connIndex := fmt.Sprintf("%d/%d", c, connections)

	buf := make([]byte, opt.ReadSize)

	workLoop(connIndex, "serverReader", "rcv/s", conn.Read, buf, opt.ReportInterval, 0, nil)

	log.Printf("serverReader: exiting: %v", conn.RemoteAddr())
}



func serverWriter(conn net.Conn, opt Options, c, connections int, isTLS bool, agg *aggregate) {

	log.Printf("serverWriter: starting: %s %v", protoLabel(isTLS), conn.RemoteAddr())

	connIndex := fmt.Sprintf("%d/%d", c, connections)

	buf := randBuf(opt.WriteSize)

	workLoop(connIndex, "serverWriter", "snd/s", conn.Write, buf, opt.ReportInterval, opt.MaxSpeed, nil)

	log.Printf("serverWriter: exiting: %v", conn.RemoteAddr())
}
