package update

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const stagePrefix = "catsitter-update-"

// Download limits. Release archives are a few MB.
const (
	maxArchive   = 256 << 20
	maxChecksums = 1 << 20
)

// Progress reports bytes downloaded so far. total is -1 when unknown.
type Progress func(done, total int64)

// Stage downloads checksums.txt and the archive for goos/goarch into a new
// directory laid out like a release mirror, Dir/download/<tag>/<file>, and
// verifies the archive. The installer can then install from Dir offline.
func (f Feed) Stage(ctx context.Context, rel Release, goos, goarch string, progress Progress) (string, error) {
	if rel.Tag != "v"+rel.Version.String() {
		return "", fmt.Errorf("release tag %q does not match version %s", rel.Tag, rel.Version)
	}
	if _, ok := ParseVersion(rel.Tag); !ok {
		return "", fmt.Errorf("release tag %q is not a version", rel.Tag)
	}
	name, err := ArchiveName(rel.Version, goos, goarch)
	if err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp("", stagePrefix)
	if err != nil {
		return "", err
	}
	ok := false
	defer func() {
		if !ok {
			os.RemoveAll(dir)
		}
	}()
	files := filepath.Join(dir, "download", rel.Tag)
	if r, err := filepath.Rel(dir, files); err != nil || r != filepath.Join("download", rel.Tag) {
		return "", fmt.Errorf("release tag %q leaves the download folder", rel.Tag)
	}
	if err := os.MkdirAll(files, 0o755); err != nil {
		return "", err
	}
	sumsPath := filepath.Join(files, "checksums.txt")
	if err := f.download(ctx, f.DownloadURL(rel.Tag, "checksums.txt"), sumsPath, maxChecksums, nil, nil); err != nil {
		return "", fmt.Errorf("download checksums.txt: %w", err)
	}
	sumsFile, err := os.Open(sumsPath)
	if err != nil {
		return "", err
	}
	sums, err := ParseChecksums(sumsFile)
	sumsFile.Close()
	if err != nil {
		return "", err
	}
	hasher, err := sums.Expect(name)
	if err != nil {
		return "", err
	}
	if err := f.download(ctx, f.DownloadURL(rel.Tag, name), filepath.Join(files, name), maxArchive, hasher, progress); err != nil {
		return "", fmt.Errorf("download %s: %w", name, err)
	}
	if err := hasher.Verify(); err != nil {
		return "", err
	}
	ok = true
	return dir, nil
}

// download saves url to path, refusing more than limit bytes.
func (f Feed) download(ctx context.Context, url, path string, limit int64, also io.Writer, progress Progress) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if err := httpsOnly(req.URL); err != nil {
		return err
	}
	if f.UserAgent != "" {
		req.Header.Set("User-Agent", f.UserAgent)
	}
	resp, err := f.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", resp.Status)
	}
	if resp.ContentLength > limit {
		return fmt.Errorf("file is %d bytes, more than the %d allowed", resp.ContentLength, limit)
	}
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	var w io.Writer = out
	if also != nil {
		w = io.MultiWriter(out, also)
	}
	if progress != nil {
		w = &counter{w: w, total: resp.ContentLength, report: progress}
	}
	n, err := io.Copy(w, io.LimitReader(resp.Body, limit+1))
	if err == nil && n > limit {
		err = fmt.Errorf("file is more than the %d bytes allowed", limit)
	}
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	return err
}

type counter struct {
	w      io.Writer
	done   int64
	total  int64
	report Progress
}

func (c *counter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.done += int64(n)
	c.report(c.done, c.total)
	return n, err
}
