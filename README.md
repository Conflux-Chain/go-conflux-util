# Golang Development Utilities

Utilities for golang developments on Conflux blockchain, especially for backend service.

|Module|Description|
|------|-------|
|[Alert](./alert/README.md)|Send notification messages to DingTalk, Telegram, SMTP email or PagerDuty.|
|[API](./api/README.md)|REST API utilities based on [gin](https://github.com/gin-gonic/gin).|
|[Cmd](./cmd)|Utilities for CLI tools.|
|[Config](./config/README.md)|Initialize all modules.|
|[DLock](./dlock/README.md)|Utilities for distributed lock.|
|[Health](./health/README.md)|Utilities for health management.|
|[HTTP](./http)|Provides common used middlewares，e.g. remote address, API key and rate limit.|
|[Log](./log/README.md)|Based on [logrus](https://github.com/sirupsen/logrus) and integrated with [Alert](./alert/README.md).|
|[Metrics](./metrics/README.md)|To monitor system runtime.|
|[Parallel](./parallel)|Utilities for parallel execution.|
|[Pprof](./pprof)|To enable pprof server based on configuration.|
|[Rate Limit](./rate)|Utilities to limit request rate, along with HTTP handler middlewares|
|[Store](./store/README.md)|Provides utilities to initialize database.|
|[Viper](./viper/README.md)| Initialize the original [viper](https://github.com/spf13/viper) in common and fix some issues.|
