package dbdump

import (
	"io"
	"regexp"
	"strconv"
)

var findOffsetRxp, parseOffsetRxp *regexp.Regexp

func init() {
	var err error
	findOffsetRxp, err = regexp.Compile(`^#\s+[0-9]+$`)
	if err != nil {
		panic(err)
	}
	parseOffsetRxp, err = regexp.Compile("[0-9]")
	if err != nil {
		panic(err)
	}
}

func parseOffset(line string) (uint32, error) {
	foundChars := parseOffsetRxp.Find([]byte(line))
	found, err := strconv.Atoi(string(foundChars))
	if err != nil {
		return 0, err
	}
	return uint32(found), err
}

func collectOffsets(db io.Reader) ([]uint32, error) {
	var err error
	return []uint32{}, err
}
