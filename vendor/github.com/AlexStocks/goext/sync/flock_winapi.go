// Copyright 2015 Tim Heckman. All rights reserved.
// Copyright 2018 The Gofrs. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the FLOCK_LICENSE file.

// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.
//
// refers from github.com/gofrs/flock

// +build windows

package gxsync

import (
	"syscall"
	"unsafe"
)

var (
	kernel32, _         = syscall.LoadLibrary("kernel32.dll")
	procLockFileEx, _   = syscall.GetProcAddress(kernel32, "LockFileEx")
	procUnlockFileEx, _ = syscall.GetProcAddress(kernel32, "UnlockFileEx")
)

const (
	winLockfileFailImmediately = 0x00000001
	winLockfileExclusiveLock   = 0x00000002
	winLockfileSharedLock      = 0x00000000
)

// Use of 0x00000000 for the shared lock is a guess based on some the MS Windows
// `LockFileEX` docs, which document the `LOCKFILE_EXCLUSIVE_LOCK` flag as:
//
// > The function requests an exclusive lock. Otherwise, it requests a shared
// > lock.
//
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa365203(v=vs.85).aspx

func lockFileEx(handle syscall.Handle, flags uint32, reserved uint32, numberOfBytesToLockLow uint32, numberOfBytesToLockHigh uint32, offset *syscall.Overlapped) (bool, syscall.Errno) {
	r1, _, errNo := syscall.Syscall6(
		uintptr(procLockFileEx),
		6,
		uintptr(handle),
		uintptr(flags),
		uintptr(reserved),
		uintptr(numberOfBytesToLockLow),
		uintptr(numberOfBytesToLockHigh),
		uintptr(unsafe.Pointer(offset)))

	if r1 != 1 {
		if errNo == 0 {
			return false, syscall.EINVAL
		}

		return false, errNo
	}

	return true, 0
}

func unlockFileEx(handle syscall.Handle, reserved uint32, numberOfBytesToLockLow uint32, numberOfBytesToLockHigh uint32, offset *syscall.Overlapped) (bool, syscall.Errno) {
	r1, _, errNo := syscall.Syscall6(
		uintptr(procUnlockFileEx),
		5,
		uintptr(handle),
		uintptr(reserved),
		uintptr(numberOfBytesToLockLow),
		uintptr(numberOfBytesToLockHigh),
		uintptr(unsafe.Pointer(offset)),
		0)

	if r1 != 1 {
		if errNo == 0 {
			return false, syscall.EINVAL
		}

		return false, errNo
	}

	return true, 0
}
