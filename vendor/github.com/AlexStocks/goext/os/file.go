// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// packaeg gxos encapsulates os related functions.
package gxos

import (
	"io"
	"os"
	"time"
)

import (
	jerrors "github.com/juju/errors"
)

func IsSameFile(filename1, filename2 string) bool {
	file1, err := os.Stat(filename1)
	if err != nil {
		return false
	}

	file2, err := os.Stat(filename2)
	if err != nil {
		return false
	}

	return os.SameFile(file1, file2)
}

func CopyFile(dst, src string) error {
	d, err := os.Create(dst)
	if err != nil {
		return jerrors.Trace(err)
	}
	defer d.Close()

	s, err := os.Open(src)
	if err != nil {
		return jerrors.Trace(err)
	}
	defer s.Close()

	_, err = io.Copy(d, s)
	return jerrors.Trace(err)
}

func GetFileModifyTime(file string) (time.Time, error) {
	fi, err := os.Stat(file)
	if err != nil {
		return time.Time{}, jerrors.Trace(err)
	}

	return fi.ModTime(), nil
}
