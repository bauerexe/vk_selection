package subpub

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func wait[T any](t *testing.T, ch <-chan T, d time.Duration) (T, bool) {
	t.Helper()
	select {
	case v := <-ch:
		return v, true
	case <-time.After(d):
		var zero T
		return zero, false
	}
}

func TestOrderFIFO(t *testing.T) {
	sp := NewSubPub()
	defer func() {
		err := sp.Close(context.Background())
		assert.NoError(t, err)
	}()

	const n = 100
	got := make([]int, 0, n)
	done := make(chan struct{})

	_, err := sp.Subscribe("fifo", func(msg interface{}) {
		got = append(got, msg.(int))
		if len(got) == n {
			close(done)
		}
	})
	assert.NoError(t, err)

	for i := 0; i < n; i++ {
		err := sp.Publish("fifo", i)
		assert.NoError(t, err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for messages")
	}

	for i, v := range got {
		assert.Equal(t, i, v, "message order mismatch at index %d", i)
	}
}

func TestSlowSubscriberDoesNotBlock(t *testing.T) {
	sp := NewSubPub()
	defer func() {
		err := sp.Close(context.Background())
		assert.NoError(t, err)
	}()

	fastCh := make(chan struct{}, 1)

	_, err := sp.Subscribe("slow", func(msg interface{}) {
		time.Sleep(500 * time.Millisecond)
	})
	assert.NoError(t, err)

	_, err = sp.Subscribe("slow", func(msg interface{}) {
		fastCh <- struct{}{}
	})
	assert.NoError(t, err)

	_, err = sp.Subscribe("slow", func(msg interface{}) {
		time.Sleep(500 * time.Millisecond)
	})
	assert.NoError(t, err)

	err = sp.Publish("slow", struct{}{})
	assert.NoError(t, err)

	_, ok := wait(t, fastCh, 100*time.Millisecond)
	assert.True(t, ok, "fast subscriber was blocked by slow one")
}

func TestUnsubscribe(t *testing.T) {
	sp := NewSubPub()
	defer func() {
		err := sp.Close(context.Background())
		assert.NoError(t, err)
	}()

	rcv := make(chan struct{}, 1)

	sub, err := sp.Subscribe("unsub", func(msg interface{}) {
		rcv <- struct{}{}
	})
	assert.NoError(t, err)

	err = sp.Publish("unsub", struct{}{})
	assert.NoError(t, err)
	_, ok := wait(t, rcv, time.Second)
	assert.True(t, ok, "didn't receive before unsubscribe")

	sub.Unsubscribe()

	err = sp.Publish("unsub", struct{}{})
	assert.NoError(t, err)
	_, ok = wait(t, rcv, 200*time.Millisecond)
	assert.False(t, ok, "received after unsubscribe")
}

func TestCloseWithContext(t *testing.T) {
	sp := NewSubPub()

	var wg sync.WaitGroup
	wg.Add(1)
	num := int32(1)

	_, err := sp.Subscribe("ctx", func(msg interface{}) {
		time.Sleep(300 * time.Millisecond)
		atomic.AddInt32(&num, 1)
		wg.Done()
	})
	assert.NoError(t, err)

	err = sp.Publish("ctx", struct{}{})
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = sp.Close(ctx)
	assert.Error(t, err, "expected context timeout error")

	wg.Wait()
	assert.Equal(t, int32(2), num, "subscriber handler did not complete its work")
}

func TestMultipleSubscribersReceiveAllMessages(t *testing.T) {
	sp := NewSubPub()
	defer func() {
		err := sp.Close(context.Background())
		assert.NoError(t, err)
	}()

	const n = 5
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		_, err := sp.Subscribe("all", func(msg interface{}) {
			wg.Done()
		})
		assert.NoError(t, err)
	}

	err := sp.Publish("all", struct{}{})
	assert.NoError(t, err)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("not all subscribers received the message")
	}
}

func TestSubscribeAfterPublishGetsNothing(t *testing.T) {
	sp := NewSubPub()
	defer func() {
		err := sp.Close(context.Background())
		assert.NoError(t, err)
	}()

	err := sp.Publish("late", "should not be received")
	assert.NoError(t, err)

	rcv := make(chan struct{}, 1)

	_, err = sp.Subscribe("late", func(msg interface{}) {
		rcv <- struct{}{}
	})
	assert.NoError(t, err)

	select {
	case <-rcv:
		t.Fatal("subscriber should not receive past messages")
	case <-time.After(200 * time.Millisecond):
	}
}
