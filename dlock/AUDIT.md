# dlock 深度代码审计（2026-10-15）

审计基线：`35f6f75b05f37c4b3aadcd5e9f15cab89a53a719`。范围包括 `dlock.go`、`mysql_backend.go`、现有测试和 README，以及直接相关的 `store` 建库、连接和迁移逻辑、GORM/MySQL 驱动实现及 CI 配置。

结论：发现 8 项可复现的问题，其中 4 项按触发后的影响评为 P1、4 项评为 P2。优先处理 nonce 比较、获取完成时的租约有效性及时间计算。另列 4 项设计或测试方面的不足。P1 表示满足所述条件时可能破坏互斥，应优先修复；不表示任意部署都会被远程利用。

验证使用 Go 1.23.0、MySQL Community 8.0.32、GORM 1.30.0、gorm mysql driver 1.6.0，以及独立临时数据目录和 Unix socket。没有连接业务数据库，没有修改业务实现。测试库使用仓库建库路径指定的 `utf8mb4_general_ci`。

**验证材料**

- [复现测试](testdata/audit_20261015/repro_test.go)：10 个测试包含 8 项问题、Version 行为验证以及正常路径对照。测试的 PASS 表示当前问题成功复现，不表示实现安全。
- [实际测试输出](testdata/audit_20261015/results.txt)：使用带 `-race` 的二进制运行，全部通过，没有 Go data race 报告。
- `go vet ./dlock` 通过。
- 原有 `go test -v ./dlock` 的两个用例在未配置 `TEST_MYSQL_*` 时全部 SKIP，不能据此判断 MySQL 行为正确。

**F1 · P1：nonce 按非二进制排序规则比较，不同凭据可以操作同一把锁。**

位置：`mysql_backend.go:20–22,61,82`；相关默认配置为 `store/config_mysql.go:61–63`。

`Key` 和 `Nonce` 都是普通字符串字段，没有显式字符集或比较规则。仓库自动建库指定 `utf8mb4_general_ci`；新建的锁表继承该规则。`nonce = ?` 因而不等于逐字节相等。问题既影响续租/接管条件，也影响释放条件。

复现：用 `OwnerA` 获取 1 分钟租约，随后分别用 `ownera`、`OwnerA `（尾随空格）及 `ÓwnerA` 调用释放，均返回成功；重新创建锁后，用上述不同 nonce 获取未过期锁也全部成功。

影响：逻辑上不同的所有者可以被数据库当成同一所有者，使两个调用方都得到成功，或让非原始 nonce 删除有效锁。Key 同样受影响：不同大小写等键值可能意外共享锁空间。利用前提是客户端能调用锁操作并提交相应字符串；这里没有把该内部包假定为公开的鉴权接口。

建议：为 key 和 nonce 定义明确的字节语义和长度限制，优先使用适当长度的 VARBINARY 或二进制 token。迁移已有表及唯一索引，并测试大小写、重音、尾随空格。仅改为某些 `_bin` collation 仍可能保留 PAD SPACE 规则，不足以保证字节相等。[MySQL 官方比较规则](https://dev.mysql.com/doc/refman/8.0/en/charset-binary-collations.html)。

**F2 · P1：SQL 阻塞时间消耗完租约后，获取仍返回成功。**

位置：`mysql_backend.go:50,60–68,70–76`；`dlock.go:57–67`。

`NOW()` 表示语句开始执行的时间，而不是行锁等待结束或事务提交的时间。更新操作若在 InnoDB 行锁上等待，等待完成后仍写入旧语句时间加 lease。实现只检查 RowsAffected/Error，没有检查返回时是否还有可用租期；API 也没有返回剩余有效期。[MySQL 官方 NOW 语义](https://dev.mysql.com/doc/refman/8.0/en/date-and-time-functions.html#function_now)。

复现：先准备过期锁，用另一事务 `SELECT ... FOR UPDATE` 持有该行。A 请求 1 秒租约，确认 performance_schema 已出现行锁等待后再等 2.2 秒，然后提交阻塞事务。A 在约 2.214 秒后返回成功，数据库中的租约已经过期约 1.441 秒；B 随即获取同一把锁也成功。

影响：A 根据成功结果进入临界区时已经没有有效租约。即使调用方只执行很短的任务，也可能与 B 同时执行。提交延迟和响应延迟也会消耗租期，本用例直接验证的是数据库行锁等待。

建议：在获取前记录单调时钟时间，返回前扣除整个操作耗时及明确的安全余量；有效期耗尽时不得报告持锁成功。返回包含有效截止时间的锁句柄，调用方也需要遵守截止时间。数据库时间改为微秒精度不能单独修复这一问题；还应设置合理操作超时，并处理提交结果不确定的情况。若失败后做清理，必须按本轮 token/代次删除，避免清掉后续租约。需要严格防止过期持有者写入时，还需下游 fencing。

**F3 · P1：秒级 NOW 与小数秒 lease 混用，可能提前允许接管。**

位置：`mysql_backend.go:50,61`。

API 接收任意 `time.Duration`，但基准 `NOW()` 丢弃小数秒，过期判断也使用秒级 `NOW()`。两处截断与严格 `<` 组合后，实际接管时间与请求的租期不一致。

确定性复现使用同一个真实 MySQL 会话的 `SET timestamp` 控制语句时间，不修改代码或机器系统时间：在时间 `T+0.950s` 获取 100ms 租约，实际存入 `expired_at=T+0.100s`；在 `T+1.000s`，B 满足 `expired_at < NOW()` 并获取成功。只过了 50ms，A 请求的 100ms 尚未结束。测试日志确认 B 返回 `true, nil`。

必须区分：对整数秒 lease，在没有等待的普通情况下，严格 `<` 往往使接管晚一个秒边界，不能笼统声称所有整数秒租约都会提前失效。此处确定的互斥风险是小数秒 lease；整数秒也存在最多接近 1 秒的额外接管延迟。

建议：统一使用高精度 UTC 时间表达式及匹配的列精度，明确支持的最小租期和舍入规则；采用整数微秒参数，避免浮点秒转换。若不支持小数秒，必须显式拒绝，而不是默默接受。过期条件应按定义使用一致精度及边界，并覆盖刚好到期的测试。时间函数精度参见 [MySQL 官方文档](https://dev.mysql.com/doc/refman/8.0/en/date-and-time-functions.html#function_now)。

**F4 · P1（配置相关）：不同时区的 MySQL 会话会破坏互斥。**

位置：`mysql_backend.go:26,50,61`。

当前迁移生成 `expired_at datetime(3)`，字段没有时区信息；`NOW()` 使用当前数据库会话的时区。后端允许传入任意 `*gorm.DB`，没有统一或检查会话时区。一个会话写入的本地时间，另一个会话直接按自己的本地时间比较。

复现：两个连接池分别设置 `time_zone='+00:00'` 和 `time_zone='+08:00'`，连接同一个库。A 获取 1 分钟租约，B 无须等待就能获取同一键；B 的当前本地时间看起来比 A 的到期时间晚近 8 小时。

影响：时区较快的客户端能接管尚未到期的锁；反方向还可能造成长时间无法接管。触发条件是 MySQL 的 SESSION time_zone 不一致，不是 Go 进程所在机器的时区不同；`store.MysqlConfig.Location` 的 DSN `loc` 也不等于设置 MySQL 会话时区。

建议：存储和比较统一采用 UTC 时间表达式，或者强制所有连接池的会话时区一致并验证。该默认路径在所有连接继承相同服务器时区时通常不会触发，但当前公共构造器没有约束这一前提。[MySQL 官方时间函数时区语义](https://dev.mysql.com/doc/refman/8.0/en/date-and-time-functions.html)。

**F5 · P2：零和负租期被当成成功获取。**

位置：`dlock.go:35–36,57–58`；`mysql_backend.go:48–50`。

构造器、Manager 和公开的 MySQLBackend 均未检查 lease。`0` 和 `-1s` 会直接写入当前或过去的到期时间，并返回成功。测试中 A 使用 `-1s` 获取成功后，B 立即用正常 1 分钟租期获取也成功。

影响：遗漏配置、错误的单位转换或负数计算结果会悄悄把互斥变成无保护执行，而调用方只看到成功。与 F3 不同，这一问题不涉及有效正租期的精度，而是非法输入被接受。

建议：在公开入口拒绝 `lease <= 0`，并检查最小精度、最大租期。Backend 可被直接调用，因此不能只在 NewLockIntent 校验；目前其字段还允许构造后修改。

**F6 · P2：并发首次获取把正常竞争报告为数据库故障。**

位置：`mysql_backend.go:54,69–76`；`dlock.go:58–64`。

先 SELECT，再决定 INSERT。两个客户端同时观察到键不存在时都会 INSERT；唯一索引正确地保证只有一个成功，但失败方收到 Error 1062，而不是 `(false,nil)`。Manager 因此返回带数据库错误的包装错误，`errors.Is(err, ErrLockAcquisitionFailed)` 为 false。

复现：用 GORM query callback 在两次真实 SELECT 完成之后设置同步屏障，再同时放行真实 INSERT。一个获取成功，另一个得到 `Duplicate entry ... for key 'dlocks.uidx_key'`。

影响：把正常锁争用错误分类为基础设施异常，破坏调用方基于 ErrLockAcquisitionFailed 的重试、退避和告警逻辑。本问题本身不会让两个不同 nonce 同时成功；唯一索引仍然有效。

建议：采用经过验证的原子 upsert/条件获取设计，或者仅将预期唯一键冲突规范化为“未获取”，必要时重新尝试。不要吞掉所有数据库错误；需要兼容 GORM TranslateError 的配置差异。

**F7 · P2（使用条件相关）：nonce 跨获取轮次复用时，旧 Release 可以删除新锁。**

位置：`mysql_backend.go:59–65,82`；`dlock.go:29–36,72–73`。

释放仅匹配 `(key,nonce)`，没有本次获取的代次。nonce 由调用方提供，同 nonce 再次获取被明确允许；现有测试也使用固定的 `node0`、`node1`。若 nonce 表示节点身份而不是唯一的一轮租约，接口无法区分新旧请求。

复现：node0 第一轮获取后释放；node0 第二轮使用相同 nonce 获取；第一轮请求残留或重试的 Release 到达并成功删除第二轮锁；node1 随后可获取，而 node0 的第二轮仍可能在工作。

影响：在释放重试、延迟回调、重启后复用节点标识等情形下出现 ABA 型误释放。每轮都使用新的高熵 nonce 且不跨轮复用时，这条路径被阻断；不能说任意旧 nonce 都能删除其他新 nonce 的锁，正常对照已确认后者会被拒绝。

建议：区分“获取新租约”与“续租当前租约”，由库生成每轮唯一 token 并返回不可混淆的句柄；释放和续租匹配本轮 token/代次。若保留现 API，应显式规定 nonce 每轮唯一及重入/续租语义，修正文档示例，并对错误用法提供防护。

**F8 · P2：Viper 工厂可以成功构造不工作的 MySQL 后端。**

位置：`dlock.go:49–52`；相关 `store/config.go:69–83`。

`store.Config` 支持 MySQL、SQLite 和默认内存 SQLite。工厂无条件把打开的数据库交给 `NewMySQLBackend`，即使选择的是 SQLite。建表和构造可以成功，随后第一次 Acquire 才因 `NOW() + INTERVAL ? SECOND` 报语法错误。

复现：只设置 `store.sqlite.path`，调用 `NewLockManagerFromViper()` 成功；Acquire 返回 `failed to write lock entry: near "?": syntax error`。未配置数据库时也会进入 store 的 SQLite fallback 路径。

影响：程序启动看似成功，但分布式锁永远不可用，错误延后到业务运行阶段。这里是 fail-closed 的可用性/配置校验问题，不是 SQLite 误报持锁成功。

建议：工厂强制要求 MySQL 配置并检查实际 Dialector；提供可返回错误的构造器。若支持其他数据库，应选择真正适配其语法和跨进程一致性的 Backend，而不是沿用 MySQL 实现。

**设计边界与较低优先级问题**

1. **没有 fencing；Version 不能充当现成的 fencing token。** Version 没有通过获取 API 返回，释放删除记录后重新 INSERT 又从 0 开始。测试实际观察到 `1 -> 0`。因此即使修好前述问题，也不能阻止“进程暂停超过租期，恢复后继续写资源”的旧持有者。用于 leader election 或不可重复副作用时，需要持久单调代次及下游资源对旧代次的拒绝，或有等效事务保护。代码目前没有承诺 Version 全局单调，故将其列为设计限制，而非独立的已违反接口承诺的漏洞。
2. **测试对租期和并发基本没有覆盖。** 原有两项测试只覆盖串行获取、相同 nonce 再获取、错误 nonce 释放等；未配置数据库直接跳过。`.github/workflows/go.yml` 没有配置 MySQL 服务或测试变量。应添加独立 MySQL CI job，并覆盖本报告的场景；不能用一般的 `go test` 成功替代实际集成测试。
3. **README 的公开示例无法按现 API 编译，而且未检查获取结果。** `README.md:22` 的 NewLockIntent 缺少 nonce 参数，24 行忽略 Acquire 错误。修正参数之外，还应展示失败分支、每轮 token、租期和续租责任。README 的“ensures atomicity”不宜被理解为任意业务临界区的原子执行承诺。
4. **输入、存储与运维边界未说明。** nil LockIntent/backend 会 panic；空 nonce 被接受，多调用方采用默认空值时会共享所有者；默认 key 为 varchar(191)，超长键依赖数据库报错；过期但从不复用的键不会自动删除。store 的 GORM 日志也未设置 ParameterizedQueries，慢 SQL/错误 SQL 可能包含 nonce。这些应按调用方威胁模型做防护或文档约束，尚无本仓库调用链证据支持将它们全部升级为独立高危漏洞。

**已核对的正确行为和结论范围**

- SQL 中的 key、nonce 和 lease 使用参数绑定；没有在 dlock 实现中发现可直接成立的 SQL 注入路径。
- UPDATE 的到期/nonce 判断与写入在同一条 SQL 中执行；不能仅因前面存在 SELECT 就认定条件更新失去原子性。16 个不同 nonce 并发竞争一个已过期键的对照测试只有一个成功。
- 不同 nonce 的普通获取被拒绝；错误 nonce 的释放和不同 nonce 的旧持有者释放被拒绝；已取消 context 被传递并返回取消错误。
- `-race` 检查的是 Go 内存访问，不会发现这些数据库协议层面的竞争。本文的并发结论来自真实 MySQL 的执行结果。
- 没有验证跨 MySQL 版本、复制切换或网络分区；不对这些部署条件给出已经实测的保证。源码审计未发现不等于证明不存在其他缺陷。

**建议修复顺序**

先修 F1/F2/F3/F4 的所有权和时间不变量，再补 F5/F6/F8 的输入、错误和配置契约；同步确定 F7 的 token 生命周期。若目标是可靠的 leader election/带副作用任务，fencing 应进入 API 设计，而不只是给现有 Version 字段加注释。将复现用例改为修复后的正向安全断言并接入真实 MySQL CI。

**复现方式**

在独立的可丢弃 MySQL 实例中创建 `audit_dlock` 库，使用 `utf8mb4_general_ci`，允许测试账户访问 `performance_schema.data_lock_waits`。当前测试 helper 使用本地 Unix socket、临时实例的空密码 root，并会创建/更新锁表，只应连接专用审计实例。

从仓库根目录执行（将 socket 路径改为该独立实例的实际路径）：

```sh
DLOCK_AUDIT_SOCKET=/absolute/path/to/isolated/mysql.sock go test -race -v -count=1 -timeout=45s ./dlock/testdata/audit_20261015
```

testdata 目录不会被普通 `go test ./...` 递归执行。测试用例断言的是漏洞仍然存在；修复后相应用例失败是预期，需要重写为安全契约断言。时间边界测试使用固定 MySQL 会话 timestamp；并发插入测试通过 callback 强制竞争顺序；行锁等待、collation、时区和数据库报错均由真实 MySQL 处理。
