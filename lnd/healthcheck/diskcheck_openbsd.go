package healthcheck

import (
	"github.com/pkt-cash/pktd/btcutil/er"
	"golang.org/x/sys/unix"
)

// AvailableDiskSpaceRatio returns ratio of available disk space to total
// capacity for openbsd.
func AvailableDiskSpaceRatio(path string) (float64, er.R) {
	s := unix.Statfs_t{}
	err := unix.Statfs(path, &s)
	if err != nil {
		return 0, err
	}

	// Calculate our free blocks/total blocks to get our total ratio of
	// free blocks.
	return float64(s.F_bfree) / float64(s.F_blocks), nil
}

// AvailableDiskSpace returns the available disk space in bytes of the given
// file system for openbsd.
func AvailableDiskSpace(path string) (uint64, er.R) {
	s := unix.Statfs_t{}
	err := unix.Statfs(path, &s)
	if err != nil {
		return 0, err
	}

	return uint64(s.F_bavail) * uint64(s.F_bsize), nil
}
