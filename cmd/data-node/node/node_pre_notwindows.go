// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

//go:build !windows
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
