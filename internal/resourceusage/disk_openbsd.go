//go:build openbsd

package resourceusage

import "golang.org/x/sys/unix"

func collectFilesystemUsage(path string) (total uint64, available uint64, err error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}

	blockSize := uint64(stat.F_bsize)
	total = stat.F_blocks * blockSize
	available = uint64(stat.F_bavail) * blockSize
	return total, available, nil
}
