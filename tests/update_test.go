package tests

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sancheza/tabwrangler/core/update"
)

func TestUpdateCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.9", "1.0.10", -1},
		{"1.0.16", "1.0.16", 0},
		{"v1.0.17", "1.0.16", 1},
		{"1.1", "1.0.9", 1},
		{"2.0.0", "1.99.99", 1},
		{"1.0.16-beta", "1.0.16", 0},
		{"1.0", "1.0.0", 0},
	}
	for _, c := range cases {
		if got := update.Compare(c.a, c.b); got != c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestUpdateAssetName(t *testing.T) {
	ok := map[[2]string]string{
		{"darwin", "arm64"}:  "tabwrangler-darwin-arm64",
		{"darwin", "amd64"}:  "tabwrangler-darwin-amd64",
		{"linux", "amd64"}:   "tabwrangler-linux-amd64",
		{"linux", "arm64"}:   "tabwrangler-linux-arm64",
		{"windows", "amd64"}: "tabwrangler-windows-amd64.zip",
	}
	for k, want := range ok {
		if got, err := update.AssetName(k[0], k[1]); err != nil || got != want {
			t.Errorf("AssetName(%v) = %q, %v; want %q", k, got, err, want)
		}
	}
	for _, k := range [][2]string{{"windows", "arm64"}, {"freebsd", "amd64"}, {"linux", "386"}} {
		if _, err := update.AssetName(k[0], k[1]); err == nil {
			t.Errorf("AssetName(%v) should fail", k)
		}
	}
}

func TestUpdateParseChecksums(t *testing.T) {
	got := update.ParseChecksums("ABCD  file-a\n1234 *file-b\n\nbad line here too\n")
	if got["file-a"] != "abcd" || got["file-b"] != "1234" || len(got) != 2 {
		t.Errorf("ParseChecksums = %v", got)
	}
}

// release serves a fake GitHub API and release files.
type release struct {
	tag    string
	files  map[string][]byte
	sums   string // contents of SHA256SUMS; empty means "compute from files"
	noSums bool
	hits   map[string]int
}

func (r *release) server(t *testing.T) *httptest.Server {
	r.hits = map[string]int{}
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/repos/o/r/releases/latest", func(w http.ResponseWriter, req *http.Request) {
		r.hits["api"]++
		var assets []string
		for name := range r.files {
			assets = append(assets, fmt.Sprintf(`{"name":%q,"browser_download_url":%q}`, name, srv.URL+"/dl/"+name))
		}
		if !r.noSums {
			assets = append(assets, fmt.Sprintf(`{"name":"SHA256SUMS","browser_download_url":%q}`, srv.URL+"/dl/SHA256SUMS"))
		}
		fmt.Fprintf(w, `{"tag_name":%q,"html_url":"https://example.test/rel","assets":[%s]}`, r.tag, strings.Join(assets, ","))
	})
	mux.HandleFunc("/dl/", func(w http.ResponseWriter, req *http.Request) {
		name := strings.TrimPrefix(req.URL.Path, "/dl/")
		r.hits[name]++
		if name == "SHA256SUMS" {
			if r.sums != "" {
				fmt.Fprint(w, r.sums)
				return
			}
			for n, data := range r.files {
				sum := sha256.Sum256(data)
				fmt.Fprintf(w, "%s  %s\n", hex.EncodeToString(sum[:]), n)
			}
			return
		}
		data, ok := r.files[name]
		if !ok {
			http.NotFound(w, req)
			return
		}
		_, _ = w.Write(data)
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func testUpdater(srv *httptest.Server, goos string, out *bytes.Buffer) *update.Updater {
	u := update.New(out)
	u.APIBase = srv.URL
	u.Repo = "o/r"
	u.Client = srv.Client()
	u.GOOS, u.GOARCH = goos, "amd64"
	u.RequireHTTPS = false
	return u
}

func TestUpdateInstallsNewerRelease(t *testing.T) {
	rel := &release{tag: "v1.0.17", files: map[string][]byte{"tabwrangler-linux-amd64": []byte("new program")}}
	srv := rel.server(t)
	exe := filepath.Join(t.TempDir(), "tabwrangler")
	if err := os.WriteFile(exe, []byte("old program"), 0755); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := testUpdater(srv, "linux", &out).Run(context.Background(), "1.0.16", exe, false); err != nil {
		t.Fatalf("Run failed: %v\n%s", err, out.String())
	}
	got, _ := os.ReadFile(exe)
	if string(got) != "new program" {
		t.Errorf("program = %q, want the new one", got)
	}
	if info, _ := os.Stat(exe); info.Mode().Perm() != 0755 {
		t.Errorf("mode = %v, want 0755", info.Mode().Perm())
	}
	if !strings.Contains(out.String(), "Checksum verified") || !strings.Contains(out.String(), "1.0.17") {
		t.Errorf("unexpected output:\n%s", out.String())
	}
	entries, _ := os.ReadDir(filepath.Dir(exe))
	if len(entries) != 1 {
		t.Errorf("expected only the program in its folder, found %d entries", len(entries))
	}
}

func TestUpdateUpToDateDownloadsNothing(t *testing.T) {
	rel := &release{tag: "v1.0.16", files: map[string][]byte{"tabwrangler-linux-amd64": []byte("same")}}
	srv := rel.server(t)
	exe := filepath.Join(t.TempDir(), "tabwrangler")
	_ = os.WriteFile(exe, []byte("current"), 0755)
	var out bytes.Buffer
	if err := testUpdater(srv, "linux", &out).Run(context.Background(), "1.0.16", exe, false); err != nil {
		t.Fatal(err)
	}
	if rel.hits["tabwrangler-linux-amd64"] != 0 || !strings.Contains(out.String(), "up to date") {
		t.Errorf("downloaded when up to date: hits=%v out=%s", rel.hits, out.String())
	}
	if got, _ := os.ReadFile(exe); string(got) != "current" {
		t.Error("program changed")
	}
}

func TestUpdateDryRunChangesNothing(t *testing.T) {
	rel := &release{tag: "v1.0.17", files: map[string][]byte{"tabwrangler-linux-amd64": []byte("new")}}
	srv := rel.server(t)
	exe := filepath.Join(t.TempDir(), "tabwrangler")
	_ = os.WriteFile(exe, []byte("old"), 0755)
	var out bytes.Buffer
	if err := testUpdater(srv, "linux", &out).Run(context.Background(), "1.0.16", exe, true); err != nil {
		t.Fatal(err)
	}
	if rel.hits["tabwrangler-linux-amd64"] != 0 || rel.hits["SHA256SUMS"] != 0 {
		t.Errorf("dry run downloaded files: %v", rel.hits)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old" {
		t.Error("dry run changed the program")
	}
	if !strings.Contains(out.String(), "[DRY-RUN]") {
		t.Errorf("dry run output missing marker:\n%s", out.String())
	}
}

func TestUpdateRejectsBadChecksum(t *testing.T) {
	rel := &release{tag: "v1.0.17", files: map[string][]byte{"tabwrangler-linux-amd64": []byte("tampered")},
		sums: strings.Repeat("0", 64) + "  tabwrangler-linux-amd64\n"}
	srv := rel.server(t)
	exe := filepath.Join(t.TempDir(), "tabwrangler")
	_ = os.WriteFile(exe, []byte("old"), 0755)
	err := testUpdater(srv, "linux", &bytes.Buffer{}).Run(context.Background(), "1.0.16", exe, false)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected a checksum error, got %v", err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old" {
		t.Error("program changed after a failed checksum")
	}
}

func TestUpdateRequiresChecksumFile(t *testing.T) {
	rel := &release{tag: "v1.0.17", files: map[string][]byte{"tabwrangler-linux-amd64": []byte("new")}, noSums: true}
	srv := rel.server(t)
	exe := filepath.Join(t.TempDir(), "tabwrangler")
	_ = os.WriteFile(exe, []byte("old"), 0755)
	err := testUpdater(srv, "linux", &bytes.Buffer{}).Run(context.Background(), "1.0.16", exe, false)
	if err == nil || !strings.Contains(err.Error(), "SHA256SUMS") {
		t.Fatalf("expected an error about the missing checksum file, got %v", err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old" {
		t.Error("program changed without a checksum file")
	}
}

func TestUpdateChecksumFileMustListTheAsset(t *testing.T) {
	rel := &release{tag: "v1.0.17", files: map[string][]byte{"tabwrangler-linux-amd64": []byte("new")},
		sums: strings.Repeat("a", 64) + "  some-other-file\n"}
	srv := rel.server(t)
	exe := filepath.Join(t.TempDir(), "tabwrangler")
	_ = os.WriteFile(exe, []byte("old"), 0755)
	if err := testUpdater(srv, "linux", &bytes.Buffer{}).Run(context.Background(), "1.0.16", exe, false); err == nil {
		t.Fatal("expected an error when SHA256SUMS does not list the file")
	}
}

func TestUpdateRefusesUntrustedAddresses(t *testing.T) {
	rel := &release{tag: "v1.0.17", files: map[string][]byte{"tabwrangler-linux-amd64": []byte("new")}}
	srv := rel.server(t)
	u := testUpdater(srv, "linux", &bytes.Buffer{})
	u.RequireHTTPS = true // the test server is http://127.0.0.1, which must be refused for downloads
	exe := filepath.Join(t.TempDir(), "tabwrangler")
	_ = os.WriteFile(exe, []byte("old"), 0755)
	err := u.Run(context.Background(), "1.0.16", exe, false)
	if err == nil || !strings.Contains(err.Error(), "refusing to download") {
		t.Fatalf("expected a refusal, got %v", err)
	}
}

func zipOf(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range entries {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = f.Write([]byte(content))
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestUpdateWindowsExtractsAndRunsSetup(t *testing.T) {
	archive := zipOf(t, map[string]string{"tabwrangler.exe": "windows program", "extension/manifest.json": "{}"})
	rel := &release{tag: "v1.0.17", files: map[string][]byte{"tabwrangler-windows-amd64.zip": archive}}
	srv := rel.server(t)
	u := testUpdater(srv, "windows", &bytes.Buffer{})
	var ranFrom, exeContent, manifest string
	u.RunSetup = func(exe string) error {
		ranFrom = exe
		data, _ := os.ReadFile(exe)
		exeContent = string(data)
		m, _ := os.ReadFile(filepath.Join(filepath.Dir(exe), "extension", "manifest.json"))
		manifest = string(m)
		return nil
	}
	if err := u.Run(context.Background(), "1.0.16", `C:\x\tabwrangler.exe`, false); err != nil {
		t.Fatal(err)
	}
	if filepath.Base(ranFrom) != "tabwrangler.exe" || exeContent != "windows program" || manifest != "{}" {
		t.Errorf("setup ran from %q with exe=%q manifest=%q", ranFrom, exeContent, manifest)
	}
}

func TestExtractZipRejectsPathEscape(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "evil.zip")
	if err := os.WriteFile(archive, zipOf(t, map[string]string{"../escaped.txt": "x"}), 0600); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "out")
	if err := update.ExtractZip(archive, dest); err == nil {
		t.Fatal("expected an error for a path that escapes the folder")
	}
	if _, err := os.Stat(filepath.Join(dir, "escaped.txt")); !os.IsNotExist(err) {
		t.Error("file was written outside the destination")
	}
}

func TestUpdateLatestErrors(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	u := testUpdater(srv, "linux", &bytes.Buffer{})
	if _, err := u.Latest(context.Background()); err == nil {
		t.Fatal("expected an error when GitHub returns 404")
	}
	u.APIBase = "http://127.0.0.1:1" // nothing listens here
	if _, err := u.Latest(context.Background()); err == nil {
		t.Fatal("expected an error when GitHub is unreachable")
	}
}
