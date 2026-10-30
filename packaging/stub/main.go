// Command stub is a stand-in binary for installer tests.
// It prints a version and then waits until it is stopped.
package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// version is overwritten with -X main.version.
var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(version)
		return
	}
	if path := os.Getenv("CATSITTER_STUB_MARKER"); path != "" {
		_ = os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o644)
	}
	for {
		time.Sleep(time.Hour)
	}
}
