package bench

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	KiB = 1024
	MiB = KiB * 1024
	GiB = MiB * 1024
	TiB = GiB * 1024
)

var sizeRE = regexp.MustCompile(`(?i)^([0-9]+)([kmg]?b)?$`)

// ParseSize parses a size with B, MB, GB unit and returns its value in bytes.
func ParseSize(s string) (uint64, error) {
	m := sizeRE.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("invalid size %q", s)
	}
	v, _ := strconv.ParseUint(m[1], 10, 64)
	switch strings.ToLower(m[2]) {
	case "kb":
		v *= KiB
	case "mb":
		v *= MiB
	case "gb":
		v *= GiB
	case "tb":
		v *= TiB
	}
	return v, nil
}
