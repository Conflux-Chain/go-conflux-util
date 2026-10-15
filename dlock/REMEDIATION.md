# dlock 审计关闭与验收登记

基线：35f6f75b05f37c4b3aadcd5e9f15cab89a53a719。来源为 Astra High、
Sol High、同事清单 C1—C12、Astra Extra High（历史 AUDIT.md）。
U01—U22 是合并后的既有根因 ID，不是本次重新发现的问题。
下表等级是修复优先级；“实现验证”指本地安全断言通过，不替代生产升级验收。

## Commit 索引

| 序号 | Commit | 内容 |
|---|---|---|
| 1 | f7823fa | P1-gate：集成入口与 MySQL CI |
| 2 | 4741fcc | P1：代际句柄、字节 schema、显式初始化/迁移 |
| 3 | df97cb4 | P1：UTC 微秒、保守截止时间 |
| 4 | 6e636b5 | P2：输入、无预读竞争、错误分类 |
| 5 | 65fe3ad | P2：构造错误、资源所有权、日志隔离 |
| 6 | a6c41d7 | P3：有界清理、测试隔离、容量实测 |
| 7 | 0c3a01e | P3：可编译示例、契约与离线升级说明 |
| 8 | test(dlock): [P1-gate] finalize audit regression coverage | 本登记、组合/故障安全断言、历史材料及防御性入口检查 |

SQL 原子写路径和基础输入校验随第 2 个 API commit 一起引入，以使中间版本保持安全可运行；
第 3、4 个 commit 完成时间预算、失败处理和相应专项回归。

第 8 个 commit 无法在自身文件中引用自身 SHA，使用唯一主题定位。
PR 保留 8 个 commits；请勿 squash。

## 同事 12 项映射

| ID | 优先级 | 根因/关闭方式 | Commit | 验收测试/证据 | 状态 |
|---|---|---|---|---|---|
| C1 | P1 | 字符比较折叠身份 → VARBINARY key/BINARY token | 2,8 | TestMysqlGenerationAndBinaryIdentity、TestTokenBytesCannotBeSubstituted | 实现验证 |
| C2 | P1 | nonce 跨代复用 → 随机 token、终态句柄 | 2,3,8 | TestLeaseGeneration、TestSharedLeaseConcurrentRenewReleaseAndRecovery、TestRandomFailureBeforeSQL | 实现验证 |
| C3 | P2 契约 | 旧业务无法由租约撤销 → 明确 fencing/暂停边界 | 5,7,8 | README、ExampleLockManager_Acquire、旧代恢复测试 | 设计边界明确；未实现严格业务互斥 |
| C4 | P2 | 非正 TTL → 所有入口校验 | 3,4 | TestValidationBeforeBackend、TestDirectBackendValidation | 实现验证 |
| C5 | P1 | 秒级截断/边界 → UTC 微秒、<=、安全预算 | 3,8 | TestFractionalLeases、TestControlledMicrosecondBoundary、TestTimezoneOffsetsAndClientFoundRows | 实现验证 |
| C6 | P2 | INSERT 竞争误报数据库故障 → 仅规范预期重复键 | 4,8 | TestConcurrentAcquire、TestOnlyContentionNormalized | 实现验证 |
| C7 | P2 | 通用工厂 SQLite 错配 → MySQL-only 返回错误 | 5,8 | TestViperFactoryRejectsFallbackAndDecodeError、TestFactoryConfiguredMySQL、TestZeroValuesDoNotPanic | 实现验证 |
| C8 | P2 | 自建池生命周期不明 → 失败关闭、幂等 Close、共享池不关 | 5,8 | TestCloseOwnershipAndConcurrency、TestOwnedPoolFailureCloses、TestFixedConnectionRejectedAndOwnedPoolClosed | 实现验证 |
| C9 | P1 gate | 默认跳过/CI 无数据库 → 显式严格集成入口/双版本 CI | 1,8 | dlock.yml、普通/race/integration/vet；缺环境入口失败 | 本地验证；远端 CI 仍为合并门禁 |
| C10 | P3 | 固定测试身份/清理不足 → 随机 key、t.Cleanup、隔离库 | 2,6,8 | uniqueKey、isolatedBackend、清理注册 | 实现验证 |
| C11 | P3 | 示例签名/错误处理错误 → 编译示例检查各步骤 | 7 | ExampleLockManager_Acquire | 编译验证；运行需配置专用 MySQL |
| C12 | P3 | 预读/动态 key 膨胀 → 条件写、有界清理、容量数据 | 4,6,8 | TestBoundedCleanup、TestCleanupRechecksRenewedRow、PERFORMANCE.md | 实现验证；未承诺固定吞吐 |

## 其余合并根因映射

| ID | 优先级 | 问题及 Commit | 验收证据 | 关闭方式 |
|---|---|---|---|---|
| U01 | P1 | collation，2,8 | 身份/伪造 token 测试 | 实现验证 |
| U02 | P1 | 精度/边界，3,8 | 短 TTL 与受控恰好到期 | 实现验证 |
| U03 | P1 | 等待耗尽租约，3,8 | TestWaitConsumesAcquireBudget、TestRenewWaitAndCancelledAcquisition、TestManagerBoundsCustomBackend | 实现验证 |
| U04 | P2 | 首次竞争错误分类，4,8 | 首次及过期 16 方竞争、错误码分类 | 实现验证 |
| U05 | P2 | 非严格模式 key 截断，4,8 | TestNonStrictByteLengthBoundary（191/192 字节） | 实现验证 |
| U06 | P2 | 内部池泄漏，5,8 | owned pool 失败/成功关闭 | 实现验证 |
| U07 | P2 | nonce SQL 日志，5,8 | TestGORMIsolationAndExternalPool、TestLeaseFormattingRedactsToken | 实现验证；Backend 自行保护 Grant |
| U08 | P2 | SQLite 工厂，5,8 | Viper/方言拒绝测试 | 实现验证 |
| U09 | P2 | 构造/迁移失败 Fatal，5,8 | 解码/取消初始化、并发 Initialize | 实现验证，错误返回调用者 |
| U10 | P1 | 会话时区，3,8 | UTC/+08/-05/-04、clientFoundRows | 实现验证；固定 DST 季节偏移，不依赖可选时区表 |
| U11 | P1 | 迁移撤除唯一约束，2,8 | TestSchemaDefinitionsAndNoMutation、TestConcurrentInitializeAndMigrationRecovery、TestInitializationIsRequiredForEveryBackend | 实现验证；生产停机升级仍需操作员验收 |
| U12 | P1 | 继承 GORM clauses，2,5,8 | DryRun/OnConflict/Table/Where/logger 隔离 | 实现验证 |
| U13 | P1 | 代际 ABA，2,8 | 旧句柄释放/续租与恢复新代 | 实现验证 |
| U14 | P2 契约 | 缺 fencing，5,7,8 | README、示例、恢复组合 | 设计边界明确；无单调 fencing 序列 |
| U15 | P2 | 非法 TTL，3,4 | Manager/Backend、微秒整数约束 | 实现验证 |
| U16 | P1 gate | 测试/CI 缺口，1,8 | 双版本真实 MySQL/race/vet | 本地验证；远端 CI 合并门禁 |
| U17 | P3 | 测试身份，2,6,8 | 随机命名/注册式清理 | 实现验证 |
| U18 | P3 | README 签名，7 | 编译 Example | 编译验证 |
| U19 | P3 | 示例忽略失败，7 | 编译 Example、错误/失锁流程 | 契约及示例修正 |
| U20 | P3 | 预读开销，4,6 | 无 SELECT 的固定写路径、实测基准 | 实现验证，不保证生产吞吐 |
| U21 | P3 | 动态 key 增长，6 | 清理接口、容量估算 | 实现验证；需操作员调度 |
| U22 | P3 | 缺有界过期清理，6,8 | 上限/有效租约保护/并发续租交错 | 实现验证 |

## 本地验收记录（2026-10-15）

- Go 1.23.0，darwin/arm64。MySQL 8.0.32 独立审计实例中新建
  dlock_safety_v2；MySQL 8.4.11 临时容器的专用 dlock_test。
- 两版本均执行 `go test -race -tags integration -count=1 -timeout=180s ./dlock`。
  安全断言为 PASS，历史反例没有进入默认安全测试。
- `go test -race ./dlock`、`go test ./...`、`go vet ./...` 通过。
- 缺 TEST_MYSQL_HOST/DB 的显式 integration 入口必须失败，不能 SKIP。
- 全仓库额外 `go test -race ./...` 检出了 blockchain/sync 的既有测试竞争；
  原基线 35f6f75 的独立导出副本执行 `go test -race ./blockchain/sync`
  同样在 TestSyncFinalizedDB/TestSyncLatestDB 失败。本 PR 不改动该包。
- 本机 race 链接器报告 LC_DYSYMTAB 警告，但 dlock 二进制成功执行并通过。
- 性能原始数字和环境见 PERFORMANCE.md。历史 AUDIT.md 和 testdata 原样保留，
  它们只证明旧基线存在问题，不是修复验证入口。

## 合并与部署验收

必须等两个 MySQL CI matrix job 和通用 Go job 通过；仓库维护者应将其配置为
required checks。此 PR 不修改分支保护或生产数据库。需要人工完成并记录：
停止/排除旧业务恢复、备份、离线迁移、Initialize 校验、新版本独占启动和回退演练。

未承诺数据库任意时钟跳变、复制历史丢失、跨数据库 failover 或 split-brain 安全。
提交确认丢失使用传输层故障注入验证 fail-closed；未进行真实生产断网/崩溃实验。
这些是明确边界，不能标注为已实现 fencing 或严格业务互斥。
