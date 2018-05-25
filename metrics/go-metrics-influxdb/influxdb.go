package influxdb

import (
	"fmt"
	uurl "net/url"
	"time"

	"github.com/influxdata/influxdb/client"
	"github.com/rcrowley/go-metrics"
	"github.com/seeleteam/go-seele/log"
)

type reporter struct {
	reg      metrics.Registry
	interval time.Duration

	url      uurl.URL
	database string
	username string
	password string
	tags     map[string]string

	client *client.Client

	log *log.SeeleLog
}

// only one reporter is allowed
var rep *reporter

// InfluxDBWithTags starts a InfluxDB reporter which will post the metrics from the given registry at each d interval with the specified tags
func InfluxDBWithTags(r metrics.Registry, d time.Duration, url, database, username, password string, tags map[string]string, log *log.SeeleLog) {
	if rep != nil {
		log.Error("reporter already running!")
		return
	}
	url = parseHTTPHead(url)
	u, err := uurl.Parse(url)
	if err != nil {
		log.Error("unable to parse InfluxDB url %s. err=%v", url, err)
		return
	}

	rep = &reporter{
		reg:      r,
		interval: d,
		url:      *u,
		database: database,
		username: username,
		password: password,
		tags:     tags,
		log:      log,
	}
	if err := rep.makeClient(); err != nil {
		log.Error("unable to make InfluxDB client. err=%v", err)
		return
	}

	rep.run()
}

func (r *reporter) makeClient() (err error) {
	r.client, err = client.NewClient(client.Config{
		URL:      r.url,
		Username: r.username,
		Password: r.password,
	})

	return
}

func (r *reporter) run() {
	intervalTicker := time.Tick(r.interval)
	pingTicker := time.Tick(time.Second * 5)

	for {
		select {
		case <-intervalTicker:
			if err := r.send(); err != nil {
				r.log.Error("unable to send metrics to InfluxDB. err=%v", err)
			}
		case <-pingTicker:
			_, _, err := r.client.Ping()
			if err != nil {
				r.log.Error("got error while sending a ping to InfluxDB, trying to recreate client. err=%v", err)

				if err = r.makeClient(); err != nil {
					r.log.Error("unable to make InfluxDB client. err=%v", err)
				}
			}
		}
	}
}

func (r *reporter) send() error {
	var pts []client.Point

	r.reg.Each(func(name string, i interface{}) {
		now := time.Now()

		switch metric := i.(type) {
		case metrics.Counter:
			ms := metric.Snapshot()
			pts = append(pts, client.Point{
				Measurement: fmt.Sprintf("%s.count", name),
				Tags:        r.tags,
				Fields: map[string]interface{}{
					"value": ms.Count(),
				},
				Time: now,
			})
		case metrics.Gauge:
			ms := metric.Snapshot()
			pts = append(pts, client.Point{
				Measurement: fmt.Sprintf("%s.gauge", name),
				Tags:        r.tags,
				Fields: map[string]interface{}{
					"value": ms.Value(),
				},
				Time: now,
			})
		case metrics.GaugeFloat64:
			ms := metric.Snapshot()
			pts = append(pts, client.Point{
				Measurement: fmt.Sprintf("%s.gauge", name),
				Tags:        r.tags,
				Fields: map[string]interface{}{
					"value": ms.Value(),
				},
				Time: now,
			})
		case metrics.Histogram:
			ms := metric.Snapshot()
			ps := ms.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
			pts = append(pts, client.Point{
				Measurement: fmt.Sprintf("%s.histogram", name),
				Tags:        r.tags,
				Fields: map[string]interface{}{
					"count":    ms.Count(),
					"max":      ms.Max(),
					"mean":     ms.Mean(),
					"min":      ms.Min(),
					"stddev":   ms.StdDev(),
					"variance": ms.Variance(),
					"p50":      ps[0],
					"p75":      ps[1],
					"p95":      ps[2],
					"p99":      ps[3],
					"p999":     ps[4],
					"p9999":    ps[5],
				},
				Time: now,
			})
		case metrics.Meter:
			ms := metric.Snapshot()
			pts = append(pts, client.Point{
				Measurement: fmt.Sprintf("%s.meter", name),
				Tags:        r.tags,
				Fields: map[string]interface{}{
					"count": ms.Count(),
					"m1":    ms.Rate1(),
					"m5":    ms.Rate5(),
					"m15":   ms.Rate15(),
					"mean":  ms.RateMean(),
				},
				Time: now,
			})
		case metrics.Timer:
			ms := metric.Snapshot()
			ps := ms.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
			pts = append(pts, client.Point{
				Measurement: fmt.Sprintf("%s.timer", name),
				Tags:        r.tags,
				Fields: map[string]interface{}{
					"count":    ms.Count(),
					"max":      ms.Max(),
					"mean":     ms.Mean(),
					"min":      ms.Min(),
					"stddev":   ms.StdDev(),
					"variance": ms.Variance(),
					"p50":      ps[0],
					"p75":      ps[1],
					"p95":      ps[2],
					"p99":      ps[3],
					"p999":     ps[4],
					"p9999":    ps[5],
					"m1":       ms.Rate1(),
					"m5":       ms.Rate5(),
					"m15":      ms.Rate15(),
					"meanrate": ms.RateMean(),
				},
				Time: now,
			})
		}
	})

	bps := client.BatchPoints{
		Points:   pts,
		Database: r.database,
	}

	_, err := r.client.Write(bps)
	return err
}

// parseHttpHead find the http prifix
func parseHTTPHead(url string) string {
	if len(url) >= 7 &&
		url[0] == 'h' &&
		url[1] == 't' &&
		url[2] == 't' &&
		url[3] == 'p' &&
		url[4] == ':' &&
		url[5] == '/' &&
		url[6] == '/' {
		return url
	}
	return fmt.Sprint("http://", url)
}
