// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.
//
// refers to github.com/jonhoo/drwmutex
package gxsync

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

func map_cpus() (cpus map[uint64]int) {
	cpus = make(map[uint64]int)

	cpuinfo, err := ioutil.ReadFile("/proc/cpuinfo")
	if err != nil {
		return
	}

	var pnum int
	var apic uint64
	lines := strings.Split(string(cpuinfo), "\n")
	for i, line := range lines {
		if len(line) == 0 && i != 0 {
			cpus[apic] = pnum
			pnum = 0
			apic = 0
			continue
		}

		fields := strings.Fields(line)

		switch fields[0] {
		case "processor":
			pnum, err = strconv.Atoi(fields[2])
		case "apicid":
			apic, err = strconv.ParseUint(fields[2], 10, 64)
		}

		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return
		}
	}
	return
}
