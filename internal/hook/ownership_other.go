//go:build !unix

package hook

import "os"

// preserveOwnership is a no-op on non-Unix platforms.
func preserveOwnership(_ os.FileInfo, _ string) error {
	return nil
}
