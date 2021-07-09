// +build !windows

package node

import "syscall"

// SetUlimits sets limits (within OS-specified limits):
// * nofile - max number of open files - for badger LSM tree
func (l *NodeCommand) SetUlimits() error {
	return syscall.Setrlimit(syscall.RLIMIT_NOFILE, &syscall.Rlimit{
		Max: l.conf.UlimitNOFile,
		Cur: l.conf.UlimitNOFile,
	})
}
