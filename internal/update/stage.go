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

// Progress reports bytes downloaded so far. total is -1 when unknown.
type Progress func(done, total int64)

// Stage downloads checksums.txt and the archive for goos/goarch into a new
// directory laid out like a release mirror, Dir/download/<tag>/<file>, and
// verifies the archive. The installer can then install from Dir offline.
func (f Feed) Stage(ctx context.Context, rel Release, goos, goarch string, progress Progress) (string, error) {
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
	if err := os.MkdirAll(files, 0o755); err != nil {
		return "", err
	}
	sumsPath := filepath.Join(files, "checksums.txt")
	if err := f.download(ctx, f.DownloadURL(rel.Tag, "checksums.txt"), sumsPath, nil, nil); err != nil {
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
	if err := f.download(ctx, f.DownloadURL(rel.Tag, name), filepath.Join(files, name), hasher, progress); err != nil {
		return "", fmt.Errorf("download %s: %w", name, err)
	}
	if err := hasher.Verify(); err != nil {
		return "", err
	}
	ok = true
	return dir, nil
}

func (f Feed) download(ctx context.Context, url, path string, also io.Writer, progress Progress) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
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
	_, err = io.Copy(w, resp.Body)
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
