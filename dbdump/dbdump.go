package dbdump

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
)

var (
	findOffsetRxp,
	parseOffsetRxp,
	findLengthRxp,
	parseLengthRxp,
	discIDRxp,
	discTitleRxp,
	discYearRxp,
	discGenreRxp,
	trackTitleRxp,
	extendedRxp,
	extendedTitleRxp,
	playorderRxp *regexp.Regexp
)

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
	discIDRxp, err = regexp.Compile(`^DISCID=`)
	if err != nil {
		panic(err)
	}
	discTitleRxp, err = regexp.Compile(`^DTITLE=`)
	if err != nil {
		panic(err)
	}
	discYearRxp, err = regexp.Compile(`^DYEAR=`)
	if err != nil {
		panic(err)
	}
	discGenreRxp, err = regexp.Compile(`^DGENRE=`)
	if err != nil {
		panic(err)
	}
	trackTitleRxp, err = regexp.Compile(`^TTITLE[0-9]+=`)
	if err != nil {
		panic(err)
	}
	extendedTitleRxp, err = regexp.Compile(`^EXTT[0-9]+=`)
	if err != nil {
		panic(err)
	}
	extendedRxp, err = regexp.Compile(`^EXTD=`)
	if err != nil {
		panic(err)
	}
	playorderRxp, err = regexp.Compile(`^PLAYORDER=`)
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
