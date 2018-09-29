/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package leveldb

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/log"
)

const (
	writeDelayNThreshold       = 200
	writeDelayThreshold        = 350 * time.Millisecond
	writeDelayWarningThrottler = 1 * time.Minute
)

// DBMetrics defines the metrics used by leveldb
type DBMetrics struct {
	metricsCompTimeMeter    metrics.Meter // Meter for measuring the total time spent in database compaction
	metricsCompReadMeter    metrics.Meter // Meter for measuring the data read during compaction
	metricsCompWriteMeter   metrics.Meter // Meter for measuring the data written during compaction
	metricsWriteDelayNMeter metrics.Meter // Meter for measuring the write delay number due to database compaction
	metricsWriteDelayMeter  metrics.Meter // Meter for measuring the write delay duration due to database compaction
}

// StartMetrics create metrics and run a goroutine to collect
func StartMetrics(db database.Database, dbname string, log *log.SeeleLog) {
	m := DBMetrics{
		metricsCompTimeMeter:    metrics.GetOrRegisterMeter(dbname+".compact.time", nil),
		metricsCompReadMeter:    metrics.GetOrRegisterMeter(dbname+".compact.input", nil),
		metricsCompWriteMeter:   metrics.GetOrRegisterMeter(dbname+".compact.output", nil),
		metricsWriteDelayMeter:  metrics.GetOrRegisterMeter(dbname+".writedelay.duration", nil),
		metricsWriteDelayNMeter: metrics.GetOrRegisterMeter(dbname+".writedelay.counter", nil),
	}

	if lvdb, ok := db.(*LevelDB); ok {
		go collectDBMetrics(lvdb, &m, log)
	} else {
		log.Error(dbname, ": Error db type ! Expect type 'LevelDB'")
	}
}

func collectDBMetrics(db *LevelDB, m *DBMetrics, log *log.SeeleLog) {
	if metrics.UseNilMetrics {
		return
	}

	// Create the counters to store current and previous compaction values
	compactions := make([][]float64, 2)
	for i := 0; i < 2; i++ {
		compactions[i] = make([]float64, 3)
	}

	// Create storage and warning log tracer for write delay.
	var (
		delaystats      [2]int64
		lastWriteDelay  time.Time
		lastWriteDelayN time.Time
	)

	// Iterate ad infinitum and collect the stats
MetricsLoop:
	for i := 1; ; i++ {
		// Retrieve the database stats
		stats, err := db.db.GetProperty("leveldb.stats")
		if err != nil {
			log.Error("Failed to read database stats, err: %s", err)
			break MetricsLoop
		}
		// Find the compaction table, skip the header
		lines := strings.Split(stats, "\n")
		for len(lines) > 0 && strings.TrimSpace(lines[0]) != "Compactions" {
			lines = lines[1:]
		}
		if len(lines) <= 3 {
			log.Error("Compaction table not found")
			break MetricsLoop
		}
		lines = lines[3:]

		// Iterate over all the table rows, and accumulate the entries
		for j := 0; j < len(compactions[i%2]); j++ {
			compactions[i%2][j] = 0
		}
		for _, line := range lines {
			parts := strings.Split(line, "|")
			if len(parts) != 6 {
				break
			}
			for idx, counter := range parts[3:] {
				value, err := strconv.ParseFloat(strings.TrimSpace(counter), 64)
				if err != nil {
					log.Error("failed to compacte entry parsing, err: %s", err)
					break MetricsLoop
				}
				compactions[i%2][idx] += value
			}
		}
		// Update all the requested meters
		m.metricsCompTimeMeter.Mark(int64((compactions[i%2][0] - compactions[(i-1)%2][0]) * 1000 * 1000 * 1000))
		m.metricsCompReadMeter.Mark(int64((compactions[i%2][1] - compactions[(i-1)%2][1]) * 1024 * 1024))
		m.metricsCompWriteMeter.Mark(int64((compactions[i%2][2] - compactions[(i-1)%2][2]) * 1024 * 1024))

		// Retrieve the write delay statistic
		writedelay, err := db.db.GetProperty("leveldb.writedelay")
		if err != nil {
			log.Error("Failed to read database write delay statistic, err: %s", err)
			break MetricsLoop
		}
		var (
			delayN        int64
			delayDuration string
			duration      time.Duration
		)
		if n, err := fmt.Sscanf(writedelay, "DelayN:%d Delay:%s", &delayN, &delayDuration); n != 2 || err != nil {
			log.Error("Write delay statistic not found")
			break MetricsLoop
		}
		duration, err = time.ParseDuration(delayDuration)
		if err != nil {
			log.Error("Failed to parse delay duration, err: %s", err)
			break MetricsLoop
		}

		m.metricsWriteDelayNMeter.Mark(delayN - delaystats[0])
		// If the write delay number been collected in the last minute exceeds the predefined threshold,
		// print a warning log here.
		// If a warning that db performance is laggy has been displayed,
		// any subsequent warnings will be withhold for 1 minute to don't overwhelm the user.
		if int(m.metricsWriteDelayNMeter.Rate1()) > writeDelayNThreshold &&
			time.Now().After(lastWriteDelayN.Add(writeDelayWarningThrottler)) {
			log.Warn("Write delay number exceeds the threshold (200 per second) in the last minute")
			lastWriteDelayN = time.Now()
		}

		m.metricsWriteDelayMeter.Mark(duration.Nanoseconds() - delaystats[1])
		// If the write delay duration been collected in the last minute exceeds the predefined threshold,
		// print a warning log here.
		// If a warning that db performance is laggy has been displayed,
		// any subsequent warnings will be withhold for 1 minute to don't overwhelm the user.
		if int64(m.metricsWriteDelayMeter.Rate1()) > writeDelayThreshold.Nanoseconds() &&
			time.Now().After(lastWriteDelay.Add(writeDelayWarningThrottler)) {
			log.Warn("Write delay duration exceeds the threshold (35%% of the time) in the last minute")
			lastWriteDelay = time.Now()
		}

		delaystats[0], delaystats[1] = delayN, duration.Nanoseconds()

		// Sleep a bit, then repeat the stats collection
		select {
		case <-db.quitChan:
			return
		case <-time.After(time.Second * 3): // wait 3 seconds
		}
	}
}
