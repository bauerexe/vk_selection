// Package subpub provides a simple FIFO ‑pub/sub implementation.
package subpub

import (
	"context"
	"errors"
	"sync"
)

// MessageHandler processes messages delivered to a subscriber.
type MessageHandler func(msg interface{})

// Subscription represents a single subject subscription.
type Subscription interface {
	// Unsubscribe removes interest in the current subject for this subscription.
	Unsubscribe()
}

// SubPub is the publish/subscribe interface.
type SubPub interface {
	// Subscribe registers an asynchronous queue subscriber on the given subject.
	Subscribe(subject string, cb MessageHandler) (Subscription, error)
	// Publish sends msg to all subscribers of subject.
	Publish(subject string, msg interface{}) error
	// Close gracefully shuts down the pub/sub system.
	Close(ctx context.Context) error
}

func NewSubPub() SubPub {
	return &subPubImpl{
		subscribers: make(map[string][]*subscriber),
	}
}

type subscriber struct {
	handler   MessageHandler
	ch        chan interface{}
	closeOnce sync.Once
	wg        sync.WaitGroup
}

func newSubscriber(h MessageHandler) *subscriber {
	s := &subscriber{
		handler: h,
		ch:      make(chan interface{}),
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for msg := range s.ch {
			s.handler(msg)
		}
	}()
	return s
}

func (s *subscriber) stopSubscriber() {
	s.closeOnce.Do(func() {
		close(s.ch)
	})
	s.wg.Wait()
}

type subPubImpl struct {
	mu          sync.RWMutex
	subscribers map[string][]*subscriber
	closed      bool
}

func (sp *subPubImpl) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	if subject == "" {
		return nil, errors.New("subject cannot be empty")
	}
	if cb == nil {
		return nil, errors.New("callback cannot be nil")
	}

	sub := newSubscriber(cb)

	sp.mu.Lock()
	defer sp.mu.Unlock()
	if sp.closed {
		sub.stopSubscriber()
		return nil, errors.New("SubPub already closed")
	}
	sp.subscribers[subject] = append(sp.subscribers[subject], sub)

	return &subscriptionImpl{
		subPub:     sp,
		subject:    subject,
		subscriber: sub,
	}, nil
}

func (sp *subPubImpl) Publish(subject string, msg interface{}) error {
	if subject == "" {
		return errors.New("subject cannot be empty")
	}

	sp.mu.RLock()
	if sp.closed {
		sp.mu.RUnlock()
		return errors.New("SubPub already closed")
	}
	subs := sp.subscribers[subject]
	sp.mu.RUnlock()

	for _, sub := range subs {
		sub.ch <- msg
	}
	return nil
}

func (sp *subPubImpl) Close(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	sp.mu.Lock()
	if sp.closed {
		sp.mu.Unlock()
		return nil
	}
	sp.closed = true

	done := make(chan struct{}, 1)
	go func() {
		for _, subs := range sp.subscribers {
			for _, sub := range subs {
				sub.stopSubscriber()
			}
		}
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		sp.subscribers = nil
		sp.mu.Unlock()
		return nil
	}
}

type subscriptionImpl struct {
	subject    string
	subPub     *subPubImpl
	subscriber *subscriber
}

func (s *subscriptionImpl) Unsubscribe() {
	sp := s.subPub

	sp.mu.Lock()
	if sp.closed {
		sp.mu.Unlock()
		return
	}

	list := sp.subscribers[s.subject]
	newList := make([]*subscriber, 0, len(list)-1)
	for _, sub := range list {
		if sub != s.subscriber {
			newList = append(newList, sub)
		}
	}
	if len(newList) == 0 {
		delete(sp.subscribers, s.subject)
	} else {
		sp.subscribers[s.subject] = newList
	}
	sp.mu.Unlock()

	s.subscriber.stopSubscriber()
}
