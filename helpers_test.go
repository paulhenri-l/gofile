package file

import (
	"fmt"
	"github.com/rs/xid"
	"os"
	"testing"
)

func fakeTmpPath(b testing.TB) string {
	// Will have to wait for this to be released
	// https://go-review.googlesource.com/c/go/+/251297
	cwd, _ := os.Getwd()
	dir := fmt.Sprintf("%s/tmp_%s", cwd, xid.New().String())
	os.MkdirAll(dir, os.ModePerm)

	b.Cleanup(func() {
		os.RemoveAll(dir)
	})

	return dir
}
