package dbdump

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/alrs/freedb"
	"golang.org/x/net/html/charset"
)

var (
	findOffsetRxp,
	parseOffsetRxp,
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
	numRxp,
	filetypeRxp *regexp.Regexp
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
	filetypeRxp, err = regexp.Compile(`^#\sxmcd`)
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

func parsePair(line string) (pair, error) {
	splitPair := strings.SplitN(line, "=", 2)
	if len(splitPair) < 2 {
		return pair{}, fmt.Errorf("%s is not a key-value pair", line)
	}
	var kv pair
	copy(kv[:], splitPair[:2])
	return kv, nil
}

func extractPosNum(key string) (int, error) {
	posFound := numRxp.Find([]byte(key))
	if len(posFound) == 0 {
		return 0, fmt.Errorf("value %s has no position number", key)
	}
	return strconv.Atoi(string(posFound))
}

// ParseDump reads a freedb dump file, parses it, and returns the parsed
// data into a *freedb.Disc.
func ParseDump(dump io.Reader) *freedb.Disc {
	disc := freedb.Disc{}
	disc.Offsets = make([]uint32, 0, 20)
	disc.ParseErrors = make([]error, 0)

	decoded, err := charset.NewReader(dump, "")
	if err != nil {
		disc.ParseErrors = append(disc.ParseErrors, err)
		return &disc
	}
	scanner := bufio.NewScanner(decoded)
	scanner.Scan()
	// first line should identify the xmcd filetype
	if !filetypeRxp.Match([]byte(scanner.Text())) {
		disc.ParseErrors = append(disc.ParseErrors, fmt.Errorf("not an xmcd dump file"))
		return &disc
	}
	for scanner.Scan() {
		if findOffsetRxp.Match([]byte(scanner.Text())) {
			// collect offset
			found, err := parseOffset(scanner.Text())
			if err != nil {
				disc.ParseErrors = append(disc.ParseErrors, fmt.Errorf("error parsing offset: %s", err))
			}
			disc.Offsets = append(disc.Offsets, uint32(found))
		} else if findDiscLengthRxp.Match([]byte(scanner.Text())) {
			// collect duration
			found := parseDiscLengthRxp.Find([]byte(scanner.Text()))
			intFound, err := strconv.Atoi(string(found))
			if err != nil {
				disc.ParseErrors = append(disc.ParseErrors, err)
			}
			disc.Duration = uint16(intFound)
		} else if discIDRxp.Match([]byte(scanner.Text())) {
			// collect disc ID
			kv, err := parsePair(scanner.Text())
			if err != nil {
				disc.ParseErrors = append(disc.ParseErrors, err)
			}
			disc.ID = kv.value()
		} else if discTitleRxp.Match([]byte(scanner.Text())) {
			// collect disc title
			kv, err := parsePair(scanner.Text())
			if err != nil {
				disc.ParseErrors = append(disc.ParseErrors, err)
			}
			disc.Title = disc.Title + kv.value()
		} else if trackTitleRxp.Match([]byte(scanner.Text())) {
			// collect track title
			kv, err := parsePair(scanner.Text())
			if err != nil {
				disc.ParseErrors = append(disc.ParseErrors, err)
			}
			pos, err := extractPosNum(kv.key())
			if err != nil {
				disc.ParseErrors = append(disc.ParseErrors, err)
			}
			if len(disc.Tracks) < (pos + 1) {
				disc.Tracks = append(disc.Tracks, kv.value())
			} else {
				disc.Tracks[pos] = disc.Tracks[pos] + kv.value()
			}
		} else if discYearRxp.Match([]byte(scanner.Text())) {
			// collect year
			found := numRxp.Find([]byte(scanner.Text()))
			if string(found) == "" {
				continue
			}
			year, err := strconv.Atoi(string(found))
			if err != nil {
				disc.ParseErrors = append(disc.ParseErrors, err)
			}
			castYear := uint16(year)
			disc.Year = &castYear
		} else if discGenreRxp.Match([]byte(scanner.Text())) {
			// collect genre
			kv, err := parsePair(scanner.Text())
			if err != nil {
				disc.ParseErrors = append(disc.ParseErrors, err)
			}
			disc.Genre = kv.value()
		}
	}
	return &disc
}
