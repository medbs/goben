package core

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

const Version = "0.4"

type HostList []string

type Config struct {
	Host           string
	Listener       string
	DefaultPort    string
	Connections    int
	ReportInterval string
	TotalDuration  string
	Opt            Options
	PassiveClient  bool // suppress client send
	Udp            bool
	TlsCert        string
	TlsKey         string
	Tls            bool
	LocalAddr      string
}

type Options struct {
	ReportInterval time.Duration
	TotalDuration  time.Duration
	ReadSize       int
	WriteSize      int
	PassiveServer  bool              // suppress server send
	MaxSpeed       float64           // mbps
	Table          map[string]string // send optional information client->server
}

func (h *HostList) String() string {
	return fmt.Sprint(*h)
}

func (h *HostList) Set(value string) error {
	for _, hh := range strings.Split(value, ",") {
		*h = append(*h, hh)
	}
	return nil
}

func BadExportFilename(parameter, filename string) error {
	if filename == "" {
		return nil
	}

	if strings.Contains(filename, "%d") && strings.Contains(filename, "%s") {
		return nil
	}

	return fmt.Errorf("badExportFilename %s: filename requires '%%d' and '%%s': %s", parameter, filename)
}

// append "s" (second) to time string
func DefaultTimeUnit(s string) string {
	if len(s) < 1 {
		return s
	}
	if unicode.IsDigit(rune(s[len(s)-1])) {
		return s + "s"
	}
	return s
}
