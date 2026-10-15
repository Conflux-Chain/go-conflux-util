package dlock

import "context"

// CleanupBackend is optional for custom backends.
type CleanupBackend interface {
	CleanupExpired(context.Context, int) (int64, error)
}

const MaxCleanupBatch = 10000

// CleanupExpired removes at most limit currently expired records, without
// starting background work. The operator chooses the schedule and load budget.
func (m *LockManager) CleanupExpired(ctx context.Context, limit int) (int64, error) {
	if m == nil || ctx == nil || limit < 1 || limit > MaxCleanupBatch {
		return 0, ErrInvalidInput
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		return 0, ErrClosed
	}
	be, ok := m.backend.(CleanupBackend)
	if !ok {
		return 0, ErrUnsupported
	}
	return be.CleanupExpired(ctx, limit)
}
func (m *MySQLBackend) CleanupExpired(ctx context.Context, limit int) (int64, error) {
	if m == nil || m.db == nil || ctx == nil || limit < 1 || limit > MaxCleanupBatch {
		return 0, ErrInvalidInput
	}
	if !m.ready.Load() {
		return 0, ErrSchema
	}
	r, err := m.db.ExecContext(ctx, "DELETE FROM dlocks WHERE expired_at<=UTC_TIMESTAMP(6) ORDER BY expired_at LIMIT ?", limit)
	if err != nil {
		return 0, err
	}
	return r.RowsAffected()
}
