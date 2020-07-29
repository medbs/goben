package core

import (
	"bytes"
	"crypto/tls"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"
)

const fmtReport = "%s %7s %14s rate: %6d Mbps %6d %s"

type aggregate struct {
	Mbps  int64 // Megabit/s
	Cps   int64 // Call/s
	mutex sync.Mutex
}

type call func(p []byte) (n int, err error)

type account struct {
	prevTime  time.Time
	prevSize  int64
	prevCalls int
	size      int64
	calls     int
}



func Open(app *Config) {

	var proto string
	if app.Udp {
		proto = "udp"
	} else {
		proto = "tcp"
	}

	var wg sync.WaitGroup

	var aggReader aggregate
	var aggWriter aggregate

	dialer := net.Dialer{}

	if app.LocalAddr != "" {
		if app.Udp {
			addr, err := net.ResolveUDPAddr(proto, app.LocalAddr)
			if err != nil {
				log.Printf("open: resolve %s localAddr=%s: %v", proto, app.LocalAddr, err)
			}
			dialer.LocalAddr = addr
		} else {
			addr, err := net.ResolveTCPAddr(proto, app.LocalAddr)
			if err != nil {
				log.Printf("open: resolve %s localAddr=%s: %v", proto, app.LocalAddr, err)
			}
			dialer.LocalAddr = addr
		}
		log.Printf("open: localAddr: %s", dialer.LocalAddr)
	}



		hh := appendPortIfMissing(app.Host, app.DefaultPort)

		for i := 0; i < app.Connections; i++ {

			log.Printf("open: opening TLS=%v %s %d/%d: %s", app.Tls, proto, i, app.Connections, hh)

			if !app.Udp && app.Tls {
				// try TLS first
				log.Printf("open: trying TLS")
				conn, errDialTLS := tlsDial(dialer, proto, hh)
				if errDialTLS == nil {
					spawnClient(app, &wg, conn, i, app.Connections, true, &aggReader, &aggWriter)
					continue
				}
				log.Printf("open: trying TLS: failure: %s: %s: %v", proto, hh, errDialTLS)
			}

			if !app.Udp {
				log.Printf("open: trying non-TLS TCP")
			}

			conn, errDial := dialer.Dial(proto, hh)
			if errDial != nil {
				log.Printf("open: dial %s: %s: %v", proto, hh, errDial)
				continue
			}
			spawnClient(app, &wg, conn, i, app.Connections, false, &aggReader, &aggWriter)
		}


	wg.Wait()

	log.Printf("aggregate reading: %d Mbps %d recv/s", aggReader.Mbps, aggReader.Cps)
	log.Printf("aggregate writing: %d Mbps %d send/s", aggWriter.Mbps, aggWriter.Cps)
}

func spawnClient(app *Config, wg *sync.WaitGroup, conn net.Conn, c, connections int, isTLS bool, aggReader, aggWriter *aggregate) {
	wg.Add(1)
	go handleConnectionClient(app, wg, conn, c, connections, isTLS, aggReader, aggWriter)
}

func tlsDial(dialer net.Dialer, proto, h string) (net.Conn, error) {
	conf := &tls.Config{
		InsecureSkipVerify: true,
	}

	conn, err := tls.DialWithDialer(&dialer, proto, h, conf)

	return conn, err
}

func sendOptions(app *Config, conn io.Writer) error {
	opt := app.Opt
	if app.Udp {
		var optBuf bytes.Buffer
		enc := gob.NewEncoder(&optBuf)
		if errOpt := enc.Encode(&opt); errOpt != nil {
			log.Printf("handleConnectionClient: UDP Options failure: %v", errOpt)
			return errOpt
		}
		_, optWriteErr := conn.Write(optBuf.Bytes())
		if optWriteErr != nil {
			log.Printf("handleConnectionClient: UDP Options write: %v", optWriteErr)
			return optWriteErr
		}
	} else {
		enc := gob.NewEncoder(conn)
		if errOpt := enc.Encode(&opt); errOpt != nil {
			log.Printf("handleConnectionClient: TCP Options failure: %v", errOpt)
			return errOpt
		}
	}
	return nil
}

func handleConnectionClient(app *Config, wg *sync.WaitGroup, conn net.Conn, c, connections int, isTLS bool, aggReader, aggWriter *aggregate) {
	defer wg.Done()

	log.Printf("handleConnectionClient: starting %s %d/%d %v", protoLabel(isTLS), c, connections, conn.RemoteAddr())

	// send Options
	if errOpt := sendOptions(app, conn); errOpt != nil {
		return
	}
	opt := app.Opt
	log.Printf("handleConnectionClient: Options sent: %v", opt)

	// receive ack
	//log.Printf("handleConnectionClient: FIXME WRITEME server does not send ack for UDP")
	if !app.Udp {
		var a ack
		if errAck := ackRecv(app.Udp, conn, &a); errAck != nil {
			log.Printf("handleConnectionClient: receiving ack: %v", errAck)
			return
		}
		log.Printf("handleConnectionClient: %s ack received", protoLabel(isTLS))
	}

	doneReader := make(chan struct{})
	doneWriter := make(chan struct{})

	go clientReader(conn, c, connections, doneReader, opt, aggReader)
	if !app.PassiveClient {
		go clientWriter(conn, c, connections, doneWriter, opt, aggWriter)
	}

	tickerPeriod := time.NewTimer(app.Opt.TotalDuration)

	<-tickerPeriod.C
	log.Printf("handleConnectionClient: %v timer", app.Opt.TotalDuration)

	tickerPeriod.Stop()

	conn.Close() // force reader/writer to quit

	<-doneReader // wait reader exit
	if !app.PassiveClient {
		<-doneWriter // wait writer exit
	}

	log.Printf("handleConnectionClient: closing: %d/%d %v", c, connections, conn.RemoteAddr())
}

func clientReader(conn net.Conn, c, connections int, done chan struct{}, opt Options, agg *aggregate) {
	log.Printf("clientReader: starting: %d/%d %v", c, connections, conn.RemoteAddr())

	connIndex := fmt.Sprintf("%d/%d", c, connections)

	buf := make([]byte, opt.ReadSize)

	workLoop(connIndex, "clientReader", "rcv/s", conn.Read, buf, opt.ReportInterval, 0, agg)

	close(done)

	log.Printf("clientReader: exiting: %d/%d %v", c, connections, conn.RemoteAddr())
}

func clientWriter(conn net.Conn, c, connections int, done chan struct{}, opt Options, agg *aggregate) {
	log.Printf("clientWriter: starting: %d/%d %v", c, connections, conn.RemoteAddr())

	connIndex := fmt.Sprintf("%d/%d", c, connections)

	buf := randBuf(opt.WriteSize)

	workLoop(connIndex, "clientWriter", "snd/s", conn.Write, buf, opt.ReportInterval, opt.MaxSpeed, agg)

	close(done)

	log.Printf("clientWriter: exiting: %d/%d %v", c, connections, conn.RemoteAddr())
}

func randBuf(size int) []byte {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		log.Printf("randBuf error: %v", err)
	}
	return buf
}



