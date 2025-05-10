package usecase

import (
	"context"

	pb "vk_otbor/api/pubsub/v1"
	"vk_otbor/pkg/subpub"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type PubSubServer struct {
	pb.UnimplementedPubSubServer
	bus subpub.SubPub
}

func New(bus subpub.SubPub) *PubSubServer { return &PubSubServer{bus: bus} }

func (s *PubSubServer) Publish(ctx context.Context,
	req *pb.PublishRequest) (*emptypb.Empty, error) {

	if req.GetKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "key is empty")
	}
	if err := s.bus.Publish(req.GetKey(), req.GetData()); err != nil {
		return nil, status.Errorf(codes.Internal, "publish: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (s *PubSubServer) Subscribe(req *pb.SubscribeRequest,
	stream pb.PubSub_SubscribeServer) error {

	if req.GetKey() == "" {
		return status.Error(codes.InvalidArgument, "key is empty")
	}

	events := make(chan interface{}, 16)

	sub, err := s.bus.Subscribe(req.GetKey(), func(m interface{}) { events <- m })
	if err != nil {
		return status.Errorf(codes.Internal, "subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	select {
	case <-stream.Context().Done():
		return nil
	case e := <-events:
		data, _ := e.(string)
		if err := stream.Send(&pb.Event{Data: data}); err != nil {
			return status.Errorf(codes.Unknown, "send: %v", err)
		}
	}

	return nil
}
