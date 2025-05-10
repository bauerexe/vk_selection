package app_test

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"vk_otbor/internal/app"

	"github.com/stretchr/testify/require"
)

func freePort(t *testing.T) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func writeConfig(t *testing.T, dir string, port int) {
	y := []byte(`grpc:
  host: "127.0.0.1"
  port: ` + strconv.Itoa(port) + `
log:
  level: "info"
`)
	require.NoError(t,
		os.WriteFile(filepath.Join(dir, "config.yaml"), y, 0o644))
}

func TestRun_OK_GracefulStop(t *testing.T) {
	port := freePort(t)
	tmp := t.TempDir()
	writeConfig(t, tmp, port)

	wd, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(wd)

	errCh := make(chan error, 1)
	go func() { errCh <- app.Run() }()

	require.Eventually(t, func() bool {
		c, err := net.DialTimeout("tcp",
			"127.0.0.1:"+strconv.Itoa(port),
			100*time.Millisecond)
		if err == nil {
			_ = c.Close()
			return true
		}
		return false
	}, 2*time.Second, 50*time.Millisecond)

	p, _ := os.FindProcess(os.Getpid())
	require.NoError(t, p.Signal(syscall.SIGINT))

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after SIGINT")
	}
}

func TestRun_ConfigMissing(t *testing.T) {
	tmp := t.TempDir()
	wd, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(wd)

	err := app.Run()
	require.Error(t, err)
}
