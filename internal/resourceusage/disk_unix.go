//go:build darwin || freebsd || linux

package resourceusage

import "syscall"

// collectFilesystemUsage reads the filesystem containing path. Keep this in
// an OS-specific file because syscall.Statfs and Statfs_t are not available
// on every target in the release matrix.
func collectFilesystemUsage(path string) (total uint64, available uint64, err error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}

	blockSize := uint64(stat.Bsize)
	total = uint64(stat.Blocks) * blockSize
	available = uint64(stat.Bavail) * blockSize
	return total, available, nil
}
