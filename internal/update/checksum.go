package update

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"regexp"
	"strings"
)

var sumLine = regexp.MustCompile(`^([0-9A-Fa-f]{64})\s+\*?(\S+)$`)

// Checksums maps a file name to its lowercase SHA-256 from checksums.txt.
type Checksums map[string]string

// ParseChecksums reads sha256sum output. Lines it does not understand are skipped.
func ParseChecksums(r io.Reader) (Checksums, error) {
	sums := Checksums{}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		m := sumLine.FindStringSubmatch(strings.TrimSpace(sc.Text()))
		if m == nil {
			continue
		}
		sums[m[2]] = strings.ToLower(m[1])
	}
	return sums, sc.Err()
}

// Hasher hashes bytes as they are written and checks them against a release checksum.
type Hasher struct {
	name string
	want string
	h    hash.Hash
}

// Expect returns a Hasher for name, or an error when checksums.txt has no entry.
func (c Checksums) Expect(name string) (*Hasher, error) {
	want, ok := c[name]
	if !ok {
		return nil, fmt.Errorf("checksums.txt has no entry for %s", name)
	}
	return &Hasher{name: name, want: want, h: sha256.New()}, nil
}

func (h *Hasher) Write(p []byte) (int, error) { return h.h.Write(p) }

// Verify reports a mismatch between the bytes written and the expected sum.
func (h *Hasher) Verify() error {
	got := hex.EncodeToString(h.h.Sum(nil))
	if got != h.want {
		return fmt.Errorf("checksum mismatch for %s", h.name)
	}
	return nil
}
