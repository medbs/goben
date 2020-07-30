package core

import (
	"log"
	"runtime"
	"time"
)

func workLoop(conn, label, cpsLabel string, f call, buf []byte, reportInterval time.Duration, maxSpeed float64, agg *aggregate) (minMbPerSec float64, maxMbPerSec float64, avgMbPerSec int64) {

	start := time.Now()
	acc := &account{}
	acc.prevTime = start

	var minMbps float64 = 50000
	var maxMbps float64 = 0


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

		mbps:= acc.update(n, reportInterval, conn, label, cpsLabel)

		if mbps < minMbps {
			minMbps = mbps
		}

		if mbps > maxMbps {
			maxMbps = mbps
		}
	}

	avgMbps:= acc.average(start, conn, label, cpsLabel, agg)

	return minMbps,maxMbps,avgMbps
}

func (a *account) update(n int, reportInterval time.Duration, conn, label, cpsLabel string) (mbps float64) {
	a.calls++
	a.size += int64(n)
	var megaBytesPerSec float64

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
		megaBytesPerSec = mbps

	}
	return megaBytesPerSec
}

func (a *account) average(start time.Time, conn, label, cpsLabel string, agg *aggregate) (avgMbPerSec int64){
	elapSec := time.Since(start).Seconds()
	mbps := int64(float64(8*a.size) / (1000000 * elapSec))
	cps := int64(float64(a.calls) / elapSec)
	log.Printf(fmtReport, conn, "average", label, mbps, cps, cpsLabel)

	agg.mutex.Lock()
	agg.Mbps += mbps
	agg.Cps += cps
	agg.mutex.Unlock()
	return mbps
}

