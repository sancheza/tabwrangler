// Package update checks GitHub for a newer TabWrangler release and installs it.
//
// The only network traffic TabWrangler makes is initiated here, and only when the user runs
// `tabwrangler --update`. Downloads come from the project's GitHub releases over HTTPS and are
// verified against the release's SHA256SUMS file before anything is replaced.
package update

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	// Repo is the GitHub repository that publishes releases.
	Repo = "sancheza/tabwrangler"
	// APIBase is the GitHub API address.
	APIBase = "https://api.github.com"
	// ChecksumsAsset is the release file that lists the SHA-256 of every other asset.
	ChecksumsAsset = "SHA256SUMS"

	maxDownloadBytes = 200 << 20
	trustedHost      = "github.com"
)

// Release is the part of a GitHub release the updater needs.
type Release struct {
	Tag    string
	URL    string            // release page
	Assets map[string]string // asset name -> download URL
}

// Updater downloads and installs releases. Create one with New.
type Updater struct {
	Client       *http.Client
	APIBase      string
	Repo         string
	GOOS         string
	GOARCH       string
	RequireHTTPS bool // refuse download addresses that are not https://github.com/...
	Out          io.Writer
	// RunSetup runs `tabwrangler setup` from a freshly extracted program (Windows only).
	RunSetup func(exe string) error
}

// New returns an Updater for the running platform.
func New(out io.Writer) *Updater {
	return &Updater{
		Client:       &http.Client{Timeout: 5 * time.Minute},
		APIBase:      APIBase,
		Repo:         Repo,
		GOOS:         runtime.GOOS,
		GOARCH:       runtime.GOARCH,
		RequireHTTPS: true,
		Out:          out,
		RunSetup: func(exe string) error {
			cmd := exec.Command(exe, "setup")
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			return cmd.Run()
		},
	}
}

// AssetName returns the release file for a platform.
func AssetName(goos, goarch string) (string, error) {
	switch {
	case goos == "windows" && goarch == "amd64":
		return "tabwrangler-windows-amd64.zip", nil
	case (goos == "darwin" || goos == "linux") && (goarch == "amd64" || goarch == "arm64"):
		return fmt.Sprintf("tabwrangler-%s-%s", goos, goarch), nil
	}
	return "", fmt.Errorf("no release is published for %s/%s", goos, goarch)
}

// Compare returns -1, 0, or 1 as version a is older than, equal to, or newer than b.
// Versions are dot-separated numbers with an optional leading "v"; text after "-" is ignored.
func Compare(a, b string) int {
	pa, pb := parts(a), parts(b)
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y int
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

func parts(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	var out []int
	for _, p := range strings.Split(v, ".") {
		n, _ := strconv.Atoi(p)
		out = append(out, n)
	}
	return out
}

// Latest fetches the newest published release.
func (u *Updater) Latest(ctx context.Context) (*Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/repos/%s/releases/latest", u.APIBase, u.Repo), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "tabwrangler-updater")
	resp, err := u.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach GitHub: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub returned %s while looking for the latest release", resp.Status)
	}
	var payload struct {
		Tag    string `json:"tag_name"`
		URL    string `json:"html_url"`
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("could not read the release information: %w", err)
	}
	rel := &Release{Tag: payload.Tag, URL: payload.URL, Assets: map[string]string{}}
	for _, a := range payload.Assets {
		rel.Assets[a.Name] = a.URL
	}
	if rel.Tag == "" {
		return nil, errors.New("the latest release has no version tag")
	}
	return rel, nil
}

func (u *Updater) checkURL(raw string) error {
	if !u.RequireHTTPS {
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() != trustedHost {
		return fmt.Errorf("refusing to download from %q: releases are fetched only from https://%s", raw, trustedHost)
	}
	return nil
}

func (u *Updater) download(ctx context.Context, raw, dest string) error {
	if err := u.checkURL(raw); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "tabwrangler-updater")
	client := *u.Client
	client.CheckRedirect = func(r *http.Request, via []*http.Request) error {
		if u.RequireHTTPS && r.URL.Scheme != "https" {
			return fmt.Errorf("refusing a redirect to %s", r.URL.Redacted())
		}
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	n, err := io.Copy(f, io.LimitReader(resp.Body, maxDownloadBytes+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	if n > maxDownloadBytes {
		return fmt.Errorf("download is larger than %d MB; refusing it", maxDownloadBytes>>20)
	}
	return nil
}

// ParseChecksums reads a SHA256SUMS file into a map of file name to lower-case hex digest.
func ParseChecksums(data string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(data, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 {
			out[strings.TrimPrefix(fields[1], "*")] = strings.ToLower(fields[0])
		}
	}
	return out
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (u *Updater) printf(format string, args ...any) {
	if u.Out != nil {
		fmt.Fprintf(u.Out, format, args...)
	}
}

// Run installs the latest release over exePath when it is newer than current.
// With dryRun it reports what it would do and changes nothing.
func (u *Updater) Run(ctx context.Context, current, exePath string, dryRun bool) error {
	asset, err := AssetName(u.GOOS, u.GOARCH)
	if err != nil {
		return err
	}
	rel, err := u.Latest(ctx)
	if err != nil {
		return err
	}
	latest := strings.TrimPrefix(rel.Tag, "v")
	if Compare(latest, current) <= 0 {
		u.printf("TabWrangler %s is up to date (latest release: %s).\n", current, latest)
		return nil
	}
	u.printf("New version available: %s (installed: %s).\n", latest, current)
	assetURL, ok := rel.Assets[asset]
	if !ok {
		return fmt.Errorf("release %s has no file named %s", rel.Tag, asset)
	}
	sumsURL, ok := rel.Assets[ChecksumsAsset]
	if !ok {
		return fmt.Errorf("release %s has no %s file, so the download cannot be verified; install it manually from %s", rel.Tag, ChecksumsAsset, rel.URL)
	}
	if dryRun {
		u.printf("[DRY-RUN] Would download %s (%s) and verify it against %s.\n", asset, assetURL, ChecksumsAsset)
		u.printf("[DRY-RUN] Would replace %s.\n", exePath)
		return nil
	}

	work, err := os.MkdirTemp("", "tabwrangler-update-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)

	u.printf("Downloading %s...\n", asset)
	sumsPath := filepath.Join(work, ChecksumsAsset)
	if err := u.download(ctx, sumsURL, sumsPath); err != nil {
		return err
	}
	sums, err := os.ReadFile(sumsPath)
	if err != nil {
		return err
	}
	want, ok := ParseChecksums(string(sums))[asset]
	if !ok {
		return fmt.Errorf("%s does not list %s; refusing to install it", ChecksumsAsset, asset)
	}
	assetPath := filepath.Join(work, asset)
	if err := u.download(ctx, assetURL, assetPath); err != nil {
		return err
	}
	got, err := fileSHA256(assetPath)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("checksum mismatch for %s (expected %s, got %s); nothing was installed", asset, want, got)
	}
	u.printf("Checksum verified.\n")

	if u.GOOS == "windows" {
		return u.installWindows(assetPath, work, latest)
	}
	return installUnix(assetPath, exePath, latest, u)
}

// installUnix replaces the program at exePath with the downloaded file. The new file is written
// next to the old one and renamed over it, which is safe while the old program is running.
func installUnix(src, exePath, latest string, u *Updater) error {
	target, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		target = exePath
	}
	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, ".tabwrangler-update-*")
	if err != nil {
		return fmt.Errorf("cannot write to %s: %w (run the update with permission to write there, for example with sudo)", dir, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	in, err := os.Open(src)
	if err != nil {
		_ = tmp.Close()
		return err
	}
	defer in.Close()
	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0755); err != nil {
		return err
	}
	if err := os.Rename(tmpName, target); err != nil {
		return fmt.Errorf("could not replace %s: %w", target, err)
	}
	u.printf("[OK] Updated %s to %s.\n", target, latest)
	u.printf("If the release notes mention companion extension changes, reload the extension in your browser.\n")
	return nil
}

// installWindows extracts the release zip and runs the new program's setup, which replaces the
// installed program and the companion extension and registers them with the browsers.
func (u *Updater) installWindows(zipPath, work, latest string) error {
	dest := filepath.Join(work, "release")
	if err := ExtractZip(zipPath, dest); err != nil {
		return err
	}
	exe := filepath.Join(dest, "tabwrangler.exe")
	if _, err := os.Stat(exe); err != nil {
		return fmt.Errorf("the release archive does not contain tabwrangler.exe: %w", err)
	}
	u.printf("Installing %s...\n", latest)
	if err := u.RunSetup(exe); err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}
	u.printf("[OK] Updated to %s. Reload the TabWrangler extension at chrome://extensions.\n", latest)
	return nil
}

// ExtractZip unpacks a zip archive into dest, refusing entries that would land outside it.
func ExtractZip(zipPath, dest string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("could not open the release archive: %w", err)
	}
	defer r.Close()
	root, err := filepath.Abs(dest)
	if err != nil {
		return err
	}
	var total int64
	for _, f := range r.File {
		target := filepath.Join(root, filepath.FromSlash(f.Name))
		if target != root && !strings.HasPrefix(target, root+string(os.PathSeparator)) {
			return fmt.Errorf("the release archive contains an unsafe path: %q", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}
		total += int64(f.UncompressedSize64)
		if total > maxDownloadBytes {
			return errors.New("the release archive is larger than expected")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err := writeZipEntry(f, target); err != nil {
			return err
		}
	}
	return nil
}

func writeZipEntry(f *zip.File, target string) error {
	in, err := f.Open()
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, io.LimitReader(in, maxDownloadBytes)); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
