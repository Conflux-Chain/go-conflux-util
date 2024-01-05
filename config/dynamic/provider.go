package dynamic

import (
	"context"
	"crypto/md5"
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const defaultMonitorInterval = time.Second * 15

// KvPair holds both a key and value pair.
type KvPair struct {
	Key   string
	Value string
}

// KvConfig database K/V model.
type KvConfig struct {
	ID        uint32    // Primary key
	Name      string    `gorm:"unique;size:128;not null"` // Config key
	Value     string    `gorm:"type:text"`                // Config value
	CreatedAt time.Time // Creation time
	UpdatedAt time.Time // Update time
}

func (KvConfig) TableName() string {
	return "kv_configs"
}

// KvConfigProvider retrieves configurations from database as a key/value store.
type KvConfigProvider struct {
	mu              sync.Mutex
	db              *gorm.DB
	tabler          schema.Tabler //  define the database table name
	callbacks       map[string]OnConfigChangedCallback
	monitorInterval time.Duration
}

func NewDefaultKvConfigProvider(db *gorm.DB) *KvConfigProvider {
	return NewKvConfigProvider(db, &KvConfig{}, defaultMonitorInterval)
}

func NewKvConfigProvider(
	db *gorm.DB, tabler schema.Tabler, monitorInterval time.Duration) *KvConfigProvider {
	return &KvConfigProvider{
		db:              db,
		tabler:          tabler,
		callbacks:       make(map[string]OnConfigChangedCallback),
		monitorInterval: monitorInterval,
	}
}

func (p *KvConfigProvider) Get(key string) (string, error) {
	var kvc KvConfig
	err := p.db.Model(p.tabler).Where("name = ?", key).First(&kvc).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}

	if err != nil {
		return "", err
	}

	return kvc.Value, nil
}

func (p *KvConfigProvider) Set(key string, value string) error {
	res := p.db.Model(p.tabler).Where("name = ?", key).Update("value", value)
	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 { // try insert instead
		res = p.db.Model(p.tabler).Create(&KvConfig{
			Name: key, Value: value,
		})
	}

	return res.Error
}

func (p *KvConfigProvider) Watch(ctx context.Context, key string) {
	t := time.NewTimer(p.monitorInterval)
	defer t.Stop()

	var oldChecksum [16]byte
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			data, err := p.Get(key)
			if err != nil {
				logrus.WithField("key", key).
					WithError(err).Error("Failed to read MySQL KV store")
				continue
			}

			var checkSum [16]byte
			if len(data) > 0 {
				checkSum = md5.Sum([]byte(data))
			}

			if oldChecksum != checkSum {
				p.handleCallback(&KvPair{Key: key, Value: data})
			}

			oldChecksum = checkSum
		}
	}
}

type OnConfigChangedCallback func(*KvPair)

func (p *KvConfigProvider) RegisterCallback(name string, cb OnConfigChangedCallback, async ...bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(async) > 0 && async[0] {
		cb = func(p *KvPair) { go cb(p) }
	}

	p.callbacks[name] = cb
}

func (p *KvConfigProvider) DeregisterCallback(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.callbacks, name)
}

func (p *KvConfigProvider) handleCallback(kvp *KvPair) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, cb := range p.callbacks {
		cb(kvp)
	}
}
