package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var parseVersionRe = regexp.MustCompile(`^[Vv](\d+(\.\d+)*)$`)

func parseVersion(versionStr string) ([]int, error) {
	if versionContainsIllegalCharacters(versionStr) {
		return nil, fmt.Errorf("illegal version string: %s , parse fail ", versionStr)
	}
	matches := parseVersionRe.FindStringSubmatch(versionStr)
	if matches == nil {
		return nil, fmt.Errorf("invalid version string format")
	}

	versionParts := strings.Split(matches[1], ".")
	version := make([]int, len(versionParts))

	for i, part := range versionParts {
		number, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid version number part: %s", part)
		}
		version[i] = number
	}

	return version, nil
}

const (
	VersionEqual   = 0
	VersionLess    = -1
	VersionGreater = 1
)

func CompareVersions(src, target string) (int, error) {
	version1, err := parseVersion(src)
	if err != nil {
		return 0, err
	}
	version2, err := parseVersion(target)
	if err != nil {
		return 0, err
	}

	for i := 0; i < len(version1) || i < len(version2); i++ {
		v1 := 0
		if i < len(version1) {
			v1 = version1[i]
		}
		v2 := 0
		if i < len(version2) {
			v2 = version2[i]
		}
		if v1 > v2 {
			return VersionGreater, nil
		} else if v1 < v2 {
			return VersionLess, nil
		}
	}

	if len(version1) > len(version2) {
		return VersionGreater, nil
	} else if len(version1) < len(version2) {
		return VersionLess, nil
	} else {
		return VersionEqual, nil
	}
}

var versionContainsIllegalCharactersRe = regexp.MustCompile(`^[Vv0-9.]+$`)

func versionContainsIllegalCharacters(s string) bool {
	return !versionContainsIllegalCharactersRe.MatchString(s)
}
