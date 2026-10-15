-- OFFLINE ONLY: stop ALL old participants AND their business tasks first.
-- Take a database backup. Do not run if dlocks_v1_backup already exists.
-- RENAME is atomic; an interrupted run can be resumed by inspecting table names.
RENAME TABLE dlocks TO dlocks_v1_backup;
CREATE TABLE dlocks (
  `key` VARBINARY(191) NOT NULL,
  token BINARY(32) NOT NULL,
  expired_at DATETIME(6) NOT NULL,
  UNIQUE KEY uidx_key (`key`),
  KEY idx_expired_at (expired_at)
) ENGINE=InnoDB;
-- Never copy old leases into v2. Keep backup for investigation.
-- Before startup invoke MySQLBackend.Initialize to validate the new schema.
-- Rollback: stop ALL v2 participants and tasks, retain v2 table, rename backup
-- back to dlocks, then TRUNCATE dlocks before starting old software.
