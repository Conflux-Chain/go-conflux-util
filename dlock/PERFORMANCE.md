# Capacity and measurements

Local measurement, 2026-10-15: Apple M3 Pro (darwin/arm64), Go 1.23.0,
MySQL Community 8.0.32 (local Unix socket, isolated audit instance), one client,
200 iterations per benchmark, no race instrumentation:

| Scenario | ns/op | SQL statements/attempt |
|---|---:|---:|
| Held hot key, contention failure | 107672 | 2 |
| New dynamic key, successful acquire | 197630 | 2 |

Command: integration environment as in README, then
`go test -tags integration -run '^$' -bench BenchmarkMySQL -benchtime=200x -count=1 ./dlock`.

These are latency samples, not production throughput guarantees. They do not
measure replication, contention with business traffic, or network failure.
Takeover of an expired row uses one conditional UPDATE; new/held keys use an
UPDATE and INSERT. Renew and release each use one statement. There is no pre-read.
At N attempts/sec, budget up to 2N statements/sec before renewal and cleanup.
Use bounded randomized caller backoff on ErrLockAcquisitionFailed.

Expired rows remain until reacquired or explicitly cleaned. At K abandoned new
keys/sec, approximately K*T records can accumulate over T seconds without
cleanup; measure actual row/index overhead and provision disk accordingly.
Schedule CleanupExpired with small batches (1..10000), monitor rows affected,
latency and free disk, and stop each sweep after an explicit batch/time budget.
No background goroutine is installed by this package.
