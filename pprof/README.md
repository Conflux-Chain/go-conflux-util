# Runtime profiling

`MustInitFromViper` reads the `pprof` configuration and applies its defaults:

```yaml
pprof:
  enabled: false
  endpoint: "127.0.0.1:6060"
```

When enabled, the server exposes the standard `/debug/pprof/` endpoints on a
dedicated router. Routes registered by the application on `http.DefaultServeMux`
are not served on this diagnostic port. Request headers have a 5-second timeout,
and idle keep-alive connections have a 60-second timeout. Read and write timeouts
are left unset to allow long captures and profile downloads.

The default endpoint accepts local connections only. Remote or container access
requires an explicit endpoint and network access controls or a tunnel. The server
does not authenticate requests, so it should not be exposed to untrusted clients.

Importing this package also imports `net/http/pprof`. The standard library
registers diagnostic handlers on `http.DefaultServeMux` during initialization,
even when `enabled` is false. Disabling this server does not remove those global
handlers. Other HTTP services in the same process should use their own routers
instead of serving `http.DefaultServeMux`.

Call initialization once during process startup. `MustInit` uses the supplied
`Config` as-is; callers constructing it directly must set the endpoint themselves.
When disabled, it does not start a listener. When enabled, it starts serving in a
background goroutine for the lifetime of the process and provides no shutdown
entry point. Configuration loading and listening failures are logged with `Fatal`
and exit the process; unexpected serving failures are logged with `Error` and
stop only the diagnostic server.
