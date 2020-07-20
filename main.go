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

	flag.Var(&app.Hosts, "hosts", "comma-separated list of hosts\nyou may append an optional port to every host: host[:port]")
	flag.Var(&app.Listeners, "listeners", "comma-separated list of listen addresses\nyou may prepend an optional host to every port: [host]:port")
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
	flag.StringVar(&app.Chart, "chart", "", "output filename for rendering chart on client\n'%d' is parallel connection index to host\n'%s' is hostname:port\nexample: -chart chart-%d-%s.png")
	flag.StringVar(&app.Export, "export", "", "output filename for YAML exporting test results on client\n'%d' is parallel connection index to host\n'%s' is hostname:port\nexample: -export export-%d-%s.yaml")
	flag.StringVar(&app.Csv, "csv", "", "output filename for CSV exporting test results on client\n'%d' is parallel connection index to host\n'%s' is hostname:port\nexample: -csv export-%d-%s.csv")
	flag.BoolVar(&app.Ascii, "ascii", true, "plot ascii chart")
	flag.StringVar(&app.TlsKey, "key", "key.pem", "TLS key file")
	flag.StringVar(&app.TlsCert, "cert", "cert.pem", "TLS cert file")
	flag.BoolVar(&app.Tls, "tls", true, "set to false to disable TLS")
	flag.StringVar(&app.LocalAddr, "localAddr", "", "bind specific local address:port\nexample: -localAddr 127.0.0.1:2000")

	flag.Parse()

	if errChart := core.BadExportFilename("-chart", app.Chart); errChart != nil {
		log.Panicf("%s", errChart.Error())
	}

	if errExport := core.BadExportFilename("-export", app.Export); errExport != nil {
		log.Panicf("%s", errExport.Error())
	}

	if errCsv := core.BadExportFilename("-csv", app.Csv); errCsv != nil {
		log.Panicf("%s", errCsv.Error())
	}

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

	if len(app.Listeners) == 0 {
		app.Listeners = []string{app.DefaultPort}
	}

	log.Printf("goben version " + core.Version + " runtime " + runtime.Version() + " GOMAXPROCS=" + strconv.Itoa(runtime.GOMAXPROCS(0)))
	log.Printf("connections=%d defaultPort=%s listeners=%q hosts=%q",
		app.Connections, app.DefaultPort, app.Listeners, app.Hosts)
	log.Printf("reportInterval=%s totalDuration=%s", app.Opt.ReportInterval, app.Opt.TotalDuration)

	if len(app.Hosts) == 0 {
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

