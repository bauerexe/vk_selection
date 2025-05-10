package app

import (
	pb "vk_otbor/api/pubsub/v1"
	"vk_otbor/internal/config"
	"vk_otbor/internal/grpcserver"
	"vk_otbor/internal/usecase"
	"vk_otbor/pkg/subpub"

	grpclogrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := logrus.New()
	level, err := logrus.ParseLevel(cfg.Log.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	logEntry := logrus.NewEntry(logger)

	grpclogrus.ReplaceGrpcLogger(logEntry)

	bus := subpub.NewSubPub()

	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpclogrus.UnaryServerInterceptor(logEntry)),
		grpc.ChainStreamInterceptor(
			grpclogrus.StreamServerInterceptor(logEntry)),
	)
	pb.RegisterPubSubServer(s, usecase.New(bus))

	return grpcserver.Run(grpcserver.RunDeps{
		Cfg:    cfg,
		Server: s,
		Log:    logger,
	})
}
