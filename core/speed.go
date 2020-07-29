package core

import (
	"log"
	"runtime"
	"time"
)

func workLoop(conn, label, cpsLabel string, f call, buf []byte, reportInterval time.Duration, maxSpeed float64, agg *aggregate) {

	start := time.Now()
	acc := &account{}
	acc.prevTime = start

	for {
		runtime.Gosched()

		if maxSpeed > 0 {
			elapSec := time.Since(acc.prevTime).Seconds()
			if elapSec > 0 {
				mbps := float64(8*(acc.size-acc.prevSize)) / (1000000 * elapSec)
				if mbps > maxSpeed {
					time.Sleep(time.Millisecond)
					continue
				}
			}
		}

		n, errCall := f(buf)
		if errCall != nil {
			log.Printf("workLoop: %s %s: %v", conn, label, errCall)
			break
		}

		acc.update(n, reportInterval, conn, label, cpsLabel)
	}

	acc.average(start, conn, label, cpsLabel, agg)
}

func (a *account) update(n int, reportInterval time.Duration, conn, label, cpsLabel string) {
	a.calls++
	a.size += int64(n)

	now := time.Now()
	elap := now.Sub(a.prevTime)
	if elap > reportInterval {
		elapSec := elap.Seconds()
		mbps := float64(8*(a.size-a.prevSize)) / (1000000 * elapSec)
		cps := int64(float64(a.calls-a.prevCalls) / elapSec)
		log.Printf(fmtReport, conn, "report", label, int64(mbps), cps, cpsLabel)
		a.prevTime = now
		a.prevSize = a.size
		a.prevCalls = a.calls

		log.Printf("zboub")
	}
}

func (a *account) average(start time.Time, conn, label, cpsLabel string, agg *aggregate) {
	elapSec := time.Since(start).Seconds()
	mbps := int64(float64(8*a.size) / (1000000 * elapSec))
	cps := int64(float64(a.calls) / elapSec)
	log.Printf(fmtReport, conn, "average", label, mbps, cps, cpsLabel)

	agg.mutex.Lock()
	agg.Mbps += mbps
	agg.Cps += cps
	agg.mutex.Unlock()
}
