//go:build unix

package hook

import (
	"os"
	"syscall"
)

func preserveOwnership(info os.FileInfo, dst string) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}
	return os.Chown(dst, int(stat.Uid), int(stat.Gid))
}
