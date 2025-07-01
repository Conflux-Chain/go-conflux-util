package report

import (
	"time"

	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/pkg/errors"
	"github.com/rcrowley/go-metrics"
	"github.com/sirupsen/logrus"
)

type InfluxDBConfig struct {
	// host path
	Host string `default:"http://127.0.0.1:8086"`
	// database name
	DB string `default:"metrics_db"`
	// authenticated username
	Username string
	// authenticated password
	Password string

	// Timeout for influxdb writes, defaults to no timeout.
	Timeout time.Duration `default:"10s"`

	// namespace for metrics reporting
	Namespace string
	// tags for metrics reporting
	Tags map[string]string
}

type influxdbReporter struct {
	config InfluxDBConfig
	reg    metrics.Registry
	client client.Client
}

func (r *influxdbReporter) makeClient() (err error) {
	r.client, err = client.NewHTTPClient(client.HTTPConfig{
		Addr:     r.config.Host,
		Username: r.config.Username,
		Password: r.config.Password,
		Timeout:  r.config.Timeout,
	})

	return
}

// InfluxDB reports metrics to specified InfluxDB for the given metrics.Registry at each d interval.
func InfluxDB(config InfluxDBConfig, reg metrics.Registry, d time.Duration) {
	reporter := influxdbReporter{
		config: config,
		reg:    reg,
	}

	if err := reporter.makeClient(); err != nil {
		logrus.WithError(err).Fatal("Failed to create InfluxDB client")
		return
	}

	reporter.run(d)
}

// InfluxDBOnce reports metrics to an InfluxDB for the given metrics.Registry.
func InfluxDBOnce(config InfluxDBConfig, reg metrics.Registry) error {
	reporter := influxdbReporter{
		config: config,
		reg:    reg,
	}

	if err := reporter.makeClient(); err != nil {
		return errors.WithMessage(err, "Failed to create InfluxDB client")
	}

	defer reporter.client.Close()

	if err := reporter.send(0); err != nil {
		return errors.WithMessage(err, "Failed to send metrics to InfluxDB")
	}

	return nil
}

func (r *influxdbReporter) run(d time.Duration) {
	intervalTicker := time.NewTicker(d)
	defer intervalTicker.Stop()

	pingTicker := time.NewTicker(time.Second * 5)
	defer pingTicker.Stop()

	for {
		select {
		case <-intervalTicker.C:
			if err := r.send(0); err != nil {
				logrus.WithError(err).Warn("Failed to send metrics to InfluxDB")
			}
		case <-pingTicker.C:
			if _, _, err := r.client.Ping(0); err != nil {
				logrus.WithError(err).Warn("Failed to ping InfluxDB, trying to recreate client")

				if err = r.makeClient(); err != nil {
					logrus.WithError(err).Warn("Failed to recreate InfluxDB client")
				}
			}
		}
	}
}

// send sends the measurements. If provided tstamp is >0, it is used. Otherwise, a 'fresh' timestamp is used.
func (r *influxdbReporter) send(timestamp int64) error {
	bps, err := client.NewBatchPoints(
		client.BatchPointsConfig{
			Database: r.config.DB,
		})
	if err != nil {
		return err
	}

	r.reg.Each(func(name string, i interface{}) {
		var now time.Time
		if timestamp <= 0 {
			now = time.Now()
		} else {
			now = time.Unix(timestamp, 0)
		}

		measurement, fields := ReadMeter(r.config.Namespace, name, i)
		if fields == nil {
			return
		}

		if p, err := client.NewPoint(measurement, r.config.Tags, fields, now); err == nil {
			bps.AddPoint(p)
		}
	})

	return r.client.Write(bps)
}
