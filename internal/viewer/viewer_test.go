package viewer_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raqolbi/qolauncher/internal/config"
	"github.com/raqolbi/qolauncher/internal/viewer"
)

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "2026-07-01.log"), []byte("hello\nworld\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return &config.Config{
		LogDir:        dir,
		LogPort:       8081,
		LogUsername:   "admin",
		LogPassword:   "secret",
		ViewerEnabled: true,
	}
}

func testServer(t *testing.T, cfg *config.Config) *httptest.Server {
	t.Helper()
	s := viewer.New(cfg)
	return httptest.NewServer(s.Handler())
}

func TestHealthNoAuth(t *testing.T) {
	cfg := testConfig(t)
	ts := testServer(t, cfg)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" || body["viewer"] != "enabled" {
		t.Fatalf("body = %#v", body)
	}
}

func TestLogsRequiresAuth(t *testing.T) {
	cfg := testConfig(t)
	ts := testServer(t, cfg)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/logs?format=json")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	if !strings.Contains(resp.Header.Get("WWW-Authenticate"), "QoLauncher Log Viewer") {
		t.Fatal("missing WWW-Authenticate")
	}
}

func TestLogsListJSON(t *testing.T) {
	cfg := testConfig(t)
	ts := testServer(t, cfg)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/logs?format=json", nil)
	req.SetBasicAuth("admin", "secret")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	var out struct {
		Logs []viewer.LogEntry `json:"logs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Logs) != 1 || out.Logs[0].Name != "2026-07-01.log" {
		t.Fatalf("logs = %#v", out.Logs)
	}
}

func TestLogViewRaw(t *testing.T) {
	cfg := testConfig(t)
	ts := testServer(t, cfg)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/logs/2026-07-01.log?format=raw", nil)
	req.SetBasicAuth("admin", "secret")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "hello") {
		t.Fatalf("body = %q", body)
	}
}

func TestLogDownload(t *testing.T) {
	cfg := testConfig(t)
	ts := testServer(t, cfg)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/logs/2026-07-01.log/download", nil)
	req.SetBasicAuth("admin", "secret")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if !strings.Contains(resp.Header.Get("Content-Disposition"), "2026-07-01.log") {
		t.Fatal("missing attachment header")
	}
}

func TestInvalidFilename(t *testing.T) {
	cfg := testConfig(t)
	ts := testServer(t, cfg)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/logs/not-a-valid-name.log?format=raw", nil)
	req.SetBasicAuth("admin", "secret")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestRootRedirect(t *testing.T) {
	cfg := testConfig(t)
	ts := testServer(t, cfg)
	defer ts.Close()

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/", nil)
	req.SetBasicAuth("admin", "secret")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want 302", resp.StatusCode)
	}
	if resp.Header.Get("Location") != "/logs" {
		t.Fatalf("location = %q", resp.Header.Get("Location"))
	}
}

func TestValidateFilename(t *testing.T) {
	if err := viewer.ValidateFilename("2026-07-01.log"); err != nil {
		t.Fatal(err)
	}
	if err := viewer.ValidateFilename("../etc/passwd"); err == nil {
		t.Fatal("expected error")
	}
}
