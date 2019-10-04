package dbdump

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
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
	findDiscLengthRxp,
	parseDiscLengthRxp,
	playorderRxp,
	numRxp *regexp.Regexp
)

type pair [2]string

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
	findDiscLengthRxp, err = regexp.Compile(`^#\sDisc\slength\:\s[0-9]+\sseconds$`)
	if err != nil {
		panic(err)
	}
	parseDiscLengthRxp, err = regexp.Compile(`[0-9]+`)
	if err != nil {
		panic(err)
	}
	numRxp, err = regexp.Compile(`[0-9]+`)
	if err != nil {
		panic(err)
	}
}

func (p *pair) key() string {
	return p[0]
}

func (p *pair) value() string {
	return p[1]
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

func collectDiscLength(db io.Reader) (uint16, error) {
	scanner := bufio.NewScanner(db)
	for scanner.Scan() {
		if findDiscLengthRxp.Match([]byte(scanner.Text())) {
			found := parseDiscLengthRxp.Find([]byte(scanner.Text()))
			intFound, err := strconv.Atoi(string(found))
			if err != nil {
				return 0, err
			}
			return uint16(intFound), nil
		}
	}
	return 0, errors.New("no disc length found")
}

func parsePair(line string) (pair, error) {
	splitPair := strings.Split(line, "=")
	if len(splitPair) != 2 {
		return pair{}, fmt.Errorf("%s is not a key-value pair", line)
	}
	var kv pair
	copy(kv[:], splitPair[:2])
	return kv, nil
}

func collectTracks(db io.Reader) ([]string, error) {
	var highPos int
	trackPairs := make([]pair, 0, 20)
	scanner := bufio.NewScanner(db)
	for scanner.Scan() {
		if trackTitleRxp.Match([]byte(scanner.Text())) {
			kv, err := parsePair(scanner.Text())
			if err != nil {
				return []string{}, err
			}
			trackPairs = append(trackPairs, kv)
			pos, err := extractPosNum(kv.key())
			if err != nil {
				return []string{}, err
			}
			if pos > highPos {
				highPos = pos
			}
		}
	}
	tracks := make([]string, highPos+1)
	for _, tp := range trackPairs {
		pos, err := extractPosNum(tp.key())
		if err != nil {
			return tracks, err
		}
		tracks[pos] = tracks[pos] + tp.value()
	}
	return tracks, nil
}

func extractPosNum(key string) (int, error) {
	// key values can span multiple lines that share
	// identical keys
	posFound := numRxp.Find([]byte(key))
	if len(posFound) == 0 {
		return 0, fmt.Errorf("value %s has no position number.", key)
	}
	return strconv.Atoi(string(posFound))
}

func collectDiscYear(db io.Reader) (uint16, error) {
	scanner := bufio.NewScanner(db)
	for scanner.Scan() {
		if discYearRxp.Match([]byte(scanner.Text())) {
			kv, err := parsePair(scanner.Text())
			if err != nil {
				return 0, err
			}
			val, err := strconv.Atoi(kv.value())
			if err != nil {
				return 0, err
			}
			return uint16(val), err
		}
	}
	return 0, nil
}
