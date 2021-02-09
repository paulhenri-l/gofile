package gofile

import (
	"fmt"
	"time"

	"github.com/rs/xid"
)

func newRandFileName(dirPath, prefix string) string {
	guid := xid.New()

	return fmt.Sprintf(
		"%s/%s%s_%s",
		dirPath,
		prefix,
		time.Now().UTC().Format("20060102150405"),
		guid.String(),
	)
}
