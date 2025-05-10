package usecase_test

import (
	"context"
	"sync"
	"testing"
	"time"

	pb "vk_otbor/api/pubsub/v1"
	"vk_otbor/internal/usecase"
	"vk_otbor/pkg/subpub"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type testServerStream struct{ ctx context.Context }

func (f *testServerStream) SetHeader(metadata.MD) error  { return nil }
func (f *testServerStream) SendHeader(metadata.MD) error { return nil }
func (f *testServerStream) SetTrailer(metadata.MD)       {}
func (f *testServerStream) Context() context.Context     { return f.ctx }
func (f *testServerStream) SendMsg(interface{}) error    { return nil }
func (f *testServerStream) RecvMsg(interface{}) error    { return nil }

type testStream struct {
	testServerStream
	mu     sync.Mutex
	events []*pb.Event
}

func newTestStream(ctx context.Context) *testStream {
	return &testStream{testServerStream{ctx: ctx}, sync.Mutex{}, nil}
}

func (s *testStream) Send(e *pb.Event) error {
	s.mu.Lock()
	s.events = append(s.events, e)
	s.mu.Unlock()
	return nil
}

func TestPublish(t *testing.T) {
	bus := subpub.NewSubPub()
	srv := usecase.New(bus)

	t.Run("empty key -> InvalidArgument", func(t *testing.T) {
		_, err := srv.Publish(context.Background(), &pb.PublishRequest{Key: "", Data: "x"})
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("happy path publishes into bus", func(t *testing.T) {
		msgCh := make(chan interface{}, 1)
		sub, _ := bus.Subscribe("orders", func(m interface{}) { msgCh <- m })
		defer sub.Unsubscribe()

		_, err := srv.Publish(context.Background(),
			&pb.PublishRequest{Key: "orders", Data: "order-42"})
		require.NoError(t, err)

		select {
		case v := <-msgCh:
			assert.Equal(t, "order-42", v.(string))
		case <-time.After(time.Second):
			t.Fatal("message not delivered to subscriber")
		}
	})
}

func TestSubscribe(t *testing.T) {
	bus := subpub.NewSubPub()
	srv := usecase.New(bus)

	t.Run("empty key -> InvalidArgument", func(t *testing.T) {
		stream := newTestStream(context.Background())
		err := srv.Subscribe(&pb.SubscribeRequest{Key: ""}, stream)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("server sends events to stream", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		stream := newTestStream(ctx)

		done := make(chan error, 1)
		go func() {
			done <- srv.Subscribe(&pb.SubscribeRequest{Key: "news"}, stream)
		}()
		time.Sleep(10 * time.Millisecond)
		_, err := srv.Publish(context.Background(),
			&pb.PublishRequest{Key: "news", Data: "hello"})
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			stream.mu.Lock()
			defer stream.mu.Unlock()
			return len(stream.events) == 1
		}, time.Second, 10*time.Millisecond)

		stream.mu.Lock()
		assert.Equal(t, "hello", stream.events[0].GetData())
		stream.mu.Unlock()

		cancel()
		assert.NoError(t, <-done)
	})
}
