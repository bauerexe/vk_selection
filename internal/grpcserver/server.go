package grpcserver

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vk_otbor/internal/config"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type RunDeps struct {
	Cfg    *config.Config
	Server *grpc.Server
	Log    *logrus.Logger
}

func Run(d RunDeps) error {
	addr := fmt.Sprintf("%s:%d", d.Cfg.GRPC.Host, d.Cfg.GRPC.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		d.Log.WithField("addr", addr).Info("gRPC listen")
		return d.Server.Serve(lis)
	})

	g.Go(func() error {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-quit:
			d.Log.Info("shutting down...")
			stopped := make(chan struct{})
			go func() {
				d.Server.GracefulStop()
				close(stopped)
			}()
			select {
			case <-time.After(2 * time.Second):
				d.Log.Warn("graceful timeout — forcing stop")
				d.Server.Stop()
				return nil
			case <-stopped:
				return nil
			}
		}
	})

	return g.Wait()
}
