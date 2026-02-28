# Health
Provides utilities for error tolerant health monitoring to avoid massive duplicated alerts.

Generally, system shall not report failure if auto resolved in a short time. However, system should report failure if not solved in a short time, and periodically remind failure if unrecovered for a long time.

- [Counter](./counter.go): manage health status based on the number of continuous failures.
- [TimedCounter](./timed_counter.go): manage health status based on duration since the first failure.

For example:

```go
package main

import (
	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/sirupsen/logrus"
)

func main() {
	counter := health.NewTimedCounter()

	err := doBusinessLogic()

	recovered, unhealthy, unrecovered, elapsed := counter.OnError(err)
	if recovered {
		logrus.WithField("elapsed", elapsed).Warn("System is healthy now")
	} else if unhealthy {
		logrus.WithError(err).WithField("elapsed", elapsed).Warn("Something wrong happened")
	} else if unrecovered {
		logrus.WithError(err).WithField("elapsed", elapsed).Error("System is unhealthy in a long time")
	}
}
```
