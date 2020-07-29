package main

import (
	"flag"
	"github.com/udhos/goben/core"
	"log"
	"runtime"
	"strconv"
	"time"
)



func main() {

	app := core.Config{}

	//flag.Var(&app.Hosts, "hosts", "comma-separated list of hosts\nyou may append an optional port to every host: host[:port]")
	//flag.Var(&app.Listeners, "listeners", "comma-separated list of listen addresses\nyou may prepend an optional host to every port: [host]:port")
	flag.StringVar(&app.Host, "hosts", "","")
	flag.StringVar(&app.Listener, "listeners", "","")
	flag.StringVar(&app.DefaultPort, "defaultPort", ":8080", "default port")
	flag.IntVar(&app.Connections, "connections", 1, "number of parallel connections")
	flag.StringVar(&app.ReportInterval, "reportInterval", "2s", "periodic report interval\nunspecified time unit defaults to second")
	flag.StringVar(&app.TotalDuration, "totalDuration", "10s", "test total duration\nunspecified time unit defaults to second")
	flag.IntVar(&app.Opt.ReadSize, "readSize", 50000, "read buffer size in bytes")
	flag.IntVar(&app.Opt.WriteSize, "writeSize", 50000, "write buffer size in bytes")
	flag.BoolVar(&app.PassiveClient, "passiveClient", false, "suppress client writes")
	flag.BoolVar(&app.Opt.PassiveServer, "passiveServer", false, "suppress server writes")
	flag.Float64Var(&app.Opt.MaxSpeed, "maxSpeed", 0, "bandwidth limit in mbps (0 means unlimited)")
	flag.BoolVar(&app.Udp, "udp", false, "run client in UDP mode")
	flag.StringVar(&app.TlsKey, "key", "key.pem", "TLS key file")
	flag.StringVar(&app.TlsCert, "cert", "cert.pem", "TLS cert file")
	flag.BoolVar(&app.Tls, "tls", true, "set to false to disable TLS")
	flag.StringVar(&app.LocalAddr, "localAddr", "", "bind specific local address:port\nexample: -localAddr 127.0.0.1:2000")

	flag.Parse()

	app.ReportInterval = core.DefaultTimeUnit(app.ReportInterval)
	app.TotalDuration = core.DefaultTimeUnit(app.TotalDuration)

	var errInterval error
	app.Opt.ReportInterval, errInterval = time.ParseDuration(app.ReportInterval)
	if errInterval != nil {
		log.Panicf("bad reportInterval: %q: %v", app.ReportInterval, errInterval)
	}

	var errDuration error
	app.Opt.TotalDuration, errDuration = time.ParseDuration(app.TotalDuration)
	if errDuration != nil {
		log.Panicf("bad totalDuration: %q: %v", app.TotalDuration, errDuration)
	}

	log.Printf("goben version " + core.Version + " runtime " + runtime.Version() + " GOMAXPROCS=" + strconv.Itoa(runtime.GOMAXPROCS(0)))
	log.Printf("connections=%d defaultPort=%s listeners=%q hosts=%q",
		app.Connections, app.DefaultPort, app.Listener, app.Host)
	log.Printf("reportInterval=%s totalDuration=%s", app.Opt.ReportInterval, app.Opt.TotalDuration)

	if len(app.Host) == 0 {
		log.Printf("server mode (use -hosts to switch to client mode)")
		core.Serve(&app)
		return
	}

	var proto string
	if app.Udp {
		proto = "udp"
	} else {
		proto = "tcp"
	}

	log.Printf("client mode, %s protocol", proto)
	core.Open(&app)
}

