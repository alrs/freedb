package dbdump

import (
	"bufio"
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
	parseOffsetRxp, err = regexp.Compile("[0-9]+$")
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
	// 20 tracks should suffice in most cases
	offsets := make([]uint32, 0, 20)
	scanner := bufio.NewScanner(db)
	for scanner.Scan() {
		if findOffsetRxp.Match([]byte(scanner.Text())) {
			found, err := parseOffset(scanner.Text())
			if err != nil {
				return offsets, err
			}
			offsets = append(offsets, uint32(found))
		}
	}
	return offsets, nil
}
