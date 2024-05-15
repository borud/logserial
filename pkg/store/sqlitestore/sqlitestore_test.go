package sqlitestore

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/borud/logserial/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestList(t *testing.T) {
	db, err := New(":memory:")
	require.NoError(t, err)

	ts := uint64(time.Now().UnixMilli()) - 10

	for i := uint64(0); i < 10; i++ {
		err := db.Log(model.Message{
			TS:     ts + i,
			Device: "foo",
			Msg:    fmt.Sprintf("bar number %d", i),
		})
		require.NoError(t, err)
	}

	// straight query
	msgChan := make(chan model.Message, 100)
	go func() {
		err = db.List(context.Background(), msgChan, time.Time{}, time.Now())
		require.NoError(t, err)
	}()

	for msg := range msgChan {
		fmt.Printf("msg: %v\n", msg)
	}

	require.NoError(t, db.Close())
}

func TestListDevice(t *testing.T) {
	db, err := New(":memory:")
	require.NoError(t, err)

	ts := uint64(time.Now().UnixMilli()) - 10

	for i := uint64(0); i < 10; i++ {
		err := db.Log(model.Message{
			TS:     ts + i,
			Device: "foo",
			Msg:    fmt.Sprintf("bar number %d", i),
		})
		require.NoError(t, err)
	}

	// straight query
	msgChan := make(chan model.Message, 100)
	go func() {
		err = db.List(context.Background(), msgChan, time.Time{}, time.Now(), "foo")
		require.NoError(t, err)
	}()

	i := 0
	for range msgChan {
		i++
	}
	require.Equal(t, i, 10)

	// nonexist device
	msgChan = make(chan model.Message, 100)
	go func() {
		err = db.List(context.Background(), msgChan, time.Time{}, time.Now(), "nonexist")
		require.NoError(t, err)
	}()

	i = 0
	for range msgChan {
		i++
	}
	require.Equal(t, i, 0)
	require.NoError(t, db.Close())
}

func TestListCancelled(t *testing.T) {
	db, err := New(":memory:")
	require.NoError(t, err)

	ts := uint64(time.Now().UnixMilli()) - 10

	for i := uint64(0); i < 10; i++ {
		err := db.Log(model.Message{
			TS:     ts + i,
			Device: "foo",
			Msg:    fmt.Sprintf("bar number %d", i),
		})
		require.NoError(t, err)
	}

	// straight query
	msgChan := make(chan model.Message, 100)

	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err = db.List(ctx, msgChan, time.Time{}, time.Now())
		require.ErrorIs(t, ErrListCancelled, err)
	}()

	for range msgChan {
		require.FailNow(t, "cancelled list should not return anything")
	}

	require.NoError(t, db.Close())
}
