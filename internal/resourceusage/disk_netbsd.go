//go:build netbsd

package resourceusage

import "golang.org/x/sys/unix"

// NetBSD exposes filesystem statistics through statvfs rather than the
// statfs API used by the other Unix targets in the release matrix.
func collectFilesystemUsage(path string) (total uint64, available uint64, err error) {
	var stat unix.Statvfs_t
	if err := unix.Statvfs(path, &stat); err != nil {
		return 0, 0, err
	}

	fragmentSize := uint64(stat.Frsize)
	total = stat.Blocks * fragmentSize
	available = stat.Bavail * fragmentSize
	return total, available, nil
}
