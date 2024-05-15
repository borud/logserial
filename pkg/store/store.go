// Package store contains the storage interface definition.
package store

import (
	"context"
	"time"

	"github.com/borud/logserial/pkg/model"
)

// Store is the storage interface.
type Store interface {
	Log(msg model.Message) error
	List(ctx context.Context, msgChan chan model.Message, since, until time.Time, device ...string) error
	Close() error
}
