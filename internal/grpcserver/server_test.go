package grpcserver_test

import (
	"net"
	"os"
	"strconv"
	"syscall"
	"testing"
	"time"

	"vk_otbor/internal/config"
	"vk_otbor/internal/grpcserver"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func freePort(t *testing.T) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestRun_StartAndGracefulStop(t *testing.T) {
	port := freePort(t)

	cfg := &config.Config{
		GRPC: config.GRPC{
			Host: "127.0.0.1",
			Port: port,
		},
	}

	server := grpc.NewServer()
	log := logrus.New()

	errCh := make(chan error, 1)
	go func() {
		errCh <- grpcserver.Run(grpcserver.RunDeps{
			Cfg:    cfg,
			Server: server,
			Log:    log,
		})
	}()

	require.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp",
			net.JoinHostPort("127.0.0.1", strconv.Itoa(port)),
			100*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		return false
	}, time.Second, 50*time.Millisecond)

	process, _ := os.FindProcess(os.Getpid())
	require.NoError(t, process.Signal(syscall.SIGINT))
	require.NoError(t, <-errCh)
}

func TestRun_PortAlreadyInUse(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()

	port := l.Addr().(*net.TCPAddr).Port

	cfg := &config.Config{
		GRPC: config.GRPC{
			Host: "127.0.0.1",
			Port: port,
		},
	}

	server := grpc.NewServer()
	log := logrus.New()

	err = grpcserver.Run(grpcserver.RunDeps{
		Cfg:    cfg,
		Server: server,
		Log:    log,
	})

	require.Error(t, err)
}
