package source

import (
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go-micro.dev/v4/config/source"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const (
	defaultWatchChanSize   = 100
	defaultMonitorInterval = 15 * time.Second
)

// MySQL key-value config table
type KvConfig struct {
	// Primary key
	ID uint32
	// Config key
	Name string `gorm:"unique;size:128;not null"`
	// Config value
	Value string `gorm:"type:text"`
	// Automatically managed by GORM for creation time
	CreatedAt time.Time
	// Automatically managed by GORM for update time
	UpdatedAt time.Time
}

func (c KvConfig) TableName() string {
	return "kv_configs"
}

// MySQL is a data source from which to load configs.
type Mysql struct {
	source.Options

	mu       sync.RWMutex
	watchers map[string]*watcher
	db       *gorm.DB
	// `tblkey` is used to locate the config value from the key-value table.
	tblkey string
	// `tabler` is used to define the database table name, make sure the schema
	// of this table conforms to `KvConfig` model.
	tabler schema.Tabler
}

func NewSource(
	db *gorm.DB, tblkey string, tabler schema.Tabler, opts ...source.Option) source.Source {
	if tabler == nil {
		tabler = &KvConfig{}
	}

	mysql := &Mysql{
		db:       db,
		tblkey:   tblkey,
		tabler:   tabler,
		Options:  source.NewOptions(opts...),
		watchers: make(map[string]*watcher),
	}
	go mysql.monitor()
	return mysql
}

// monitor periodically monitors new changes and notify all watchers.
func (s *Mysql) monitor() {
	t := time.NewTicker(defaultMonitorInterval)
	defer t.Stop()

	var oldChecksum string
	for {
		select {
		case <-s.Context.Done():
			return
		case <-t.C:
			if s.numWatchers() == 0 {
				// skip if no watchers
				return
			}

			cs, err := s.Read()
			if err != nil {
				logrus.WithError(err).Error("Failed to read MySQL for dynamic configurations")
				continue
			}

			var checkSum string
			if cs != nil {
				checkSum = cs.Checksum
			}

			if oldChecksum != checkSum {
				// notify observed changes
				s.notify(cs)
			}
			oldChecksum = checkSum
		}
	}
}

func (s *Mysql) notify(cs *source.ChangeSet) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, w := range s.watchers {
		if len(w.Updates) >= defaultWatchChanSize {
			// Drop if channel is already full
			logrus.WithField("watcher", w.Id).
				Info("Drop data source changeset due to channel is full")
			continue
		}

		w.Updates <- cs
	}
}

func (s *Mysql) diswatch(w *watcher) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.watchers, w.Id)
}

func (s *Mysql) numWatchers() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.watchers)
}

// implements `source.Source` interface

func (s *Mysql) Read() (*source.ChangeSet, error) {
	var conf KvConfig

	err := s.db.Model(s.tabler).Where("name = ?", s.tblkey).First(&conf).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	cs := &source.ChangeSet{
		Format:    s.Encoder.String(),
		Source:    s.String(),
		Data:      []byte(conf.Value),
		Timestamp: conf.UpdatedAt,
	}
	cs.Checksum = cs.Sum()

	return cs, nil
}

func (s *Mysql) Write(cs *source.ChangeSet) error {
	res := s.db.Model(s.tabler).Where("name = ?", s.tblkey).Update("value", cs.Data)
	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 { // try insert instead
		res = s.db.Model(s.tabler).Create(&KvConfig{
			Name:  s.tblkey,
			Value: string(cs.Data),
		})
	}

	return res.Error
}

func (s *Mysql) String() string {
	return "mysql"
}

func (s *Mysql) Watch() (source.Watcher, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	w := newWatcher(s, make(chan *source.ChangeSet, defaultWatchChanSize))
	s.watchers[w.Id] = w
	return w, nil
}
