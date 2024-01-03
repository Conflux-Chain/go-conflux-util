package source

import (
	"errors"

	"github.com/google/uuid"
	"go-micro.dev/v4/config/source"
)

type watcher struct {
	Id      string
	Source  *Mysql
	Updates chan *source.ChangeSet
	closeCh chan struct{}
}

func newWatcher(src *Mysql, updates chan *source.ChangeSet) *watcher {
	return &watcher{
		Id:      uuid.New().String(),
		Source:  src,
		Updates: updates,
		closeCh: make(chan struct{}),
	}
}

func (w *watcher) Next() (*source.ChangeSet, error) {
	select {
	case <-w.closeCh:
		return nil, errors.New("watcher closed")
	case cs := <-w.Updates:
		return cs, nil
	}
}

func (w *watcher) Stop() error {
	close(w.closeCh)
	w.Source.diswatch(w)
	return nil
}
