// icloud-pull: download evicted (dataless) iCloud files on macOS.
//
// macOS storage optimization offloads file contents to iCloud while
// keeping metadata locally. This tool scans a directory for such
// evicted files and triggers their download by reading one byte from
// each, which causes APFS to transparently fetch the content.
package main

import (
	"os"

	"github.com/hyper0x/icloud-pull/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
