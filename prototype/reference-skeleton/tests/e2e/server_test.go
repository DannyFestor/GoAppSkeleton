//go:build e2e

package e2e_test

import (
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

const (
	startupTimeout = 10 * time.Second
	staticDir      = "../../public"
)

func TestServerServesAndShutsDownOnSIGTERM(t *testing.T) {
	addr := freeAddr(t)
	server := startServer(t, buildBinary(t), addr)

	waitUntilHealthy(t, addr)
	expectStatus(t, "http://"+addr+"/static/css/main.css", http.StatusOK)

	if err := server.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if err := server.Wait(); err != nil {
		t.Errorf("server exited with %v, want a clean exit", err)
	}
}

func buildBinary(t *testing.T) string {
	t.Helper()

	binary := filepath.Join(t.TempDir(), "web")
	build := exec.Command("go", "build", "-o", binary, "example.com/myapp/cmd/web")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building: %v\n%s", err, out)
	}

	return binary
}

func freeAddr(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	return listener.Addr().String()
}

func startServer(t *testing.T, binary, addr string) *exec.Cmd {
	t.Helper()

	absStaticDir, err := filepath.Abs(staticDir)
	if err != nil {
		t.Fatal(err)
	}

	server := exec.Command(binary)
	server.Env = []string{"ADDR=" + addr, "STATIC_DIR=" + absStaticDir}
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Process.Kill() })

	return server
}

func waitUntilHealthy(t *testing.T, addr string) {
	t.Helper()

	deadline := time.Now().Add(startupTimeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + addr + "/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("server at %s not healthy within %s", addr, startupTimeout)
}

func expectStatus(t *testing.T, url string, want int) {
	t.Helper()

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != want {
		t.Errorf("GET %s = %d, want %d", url, resp.StatusCode, want)
	}
}
