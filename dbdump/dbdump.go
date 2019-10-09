// package freedb
// Copyright (C) 2019 Lars Lehtonen
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package dbdump

import (
	"bufio"
	"encoding/hex"
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
	findDiscLengthRxp,
	parseDiscLengthRxp,
	numRxp,
	filetypeRxp,
	// TODO
	extendedTitleRxp,
	playorderRxp,
	extendedRxp *regexp.Regexp
)

type pair [2]string

// Error to return when a dump is not in xmcd format
type XMCDErr struct {
}

// XMCDErr message.
func (e *XMCDErr) Error() string {
	return "not an xmcd format dump"
}

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

// ParseDump reads a freedb dump file, parses it, converts strings to UTF8,
// and returns the parsed data into a *freedb.Disc.
func ParseDump(dump io.Reader) *freedb.Disc {
	disc := freedb.Disc{}
	disc.Offsets = make([]uint32, 0, 20)
	disc.Tracks = make([]string, 0, 20)

	decoded, err := charset.NewReader(dump, "")
	if err != nil {
		disc.AppendErr(err)
		return &disc
	}

	scanner := bufio.NewScanner(decoded)
	scanner.Scan()
	// first line should identify the xmcd filetype
	if !filetypeRxp.Match([]byte(scanner.Text())) {
		var err *XMCDErr
		disc.AppendErr(err)
		return &disc
	}
	for scanner.Scan() {
		switch {
		case findOffsetRxp.Match([]byte(scanner.Text())):
			// collect offset
			found, err := parseOffset(scanner.Text())
			if err != nil {
				disc.AppendErr(fmt.Errorf("error parsing offset: %s", err))
			} else {
				disc.Offsets = append(disc.Offsets, uint32(found))
			}
		case findDiscLengthRxp.Match([]byte(scanner.Text())):
			// collect duration
			found := parseDiscLengthRxp.Find([]byte(scanner.Text()))
			intFound, err := strconv.Atoi(string(found))
			if err != nil {
				disc.AppendErr(err)
			}
			disc.Duration = uint16(intFound)
		case discIDRxp.Match([]byte(scanner.Text())):
			// collect disc ID
			kv, err := parsePair(scanner.Text())
			if err != nil {
				disc.AppendErr(err)
			}
			if len(kv.value()) < 8 {
				disc.AppendErr(fmt.Errorf("discID too short"))
			}
			disc.ID, err = hex.DecodeString(kv.value()[0:8])
			if err != nil {
				disc.AppendErr(fmt.Errorf("error decoding id to hex: %s", err))
			}
		case discTitleRxp.Match([]byte(scanner.Text())):
			// collect disc title
			kv, err := parsePair(scanner.Text())
			if err != nil {
				disc.AppendErr(err)
			}
			disc.AppendTitle(kv.value())
		case trackTitleRxp.Match([]byte(scanner.Text())):
			// collect track title
			kv, err := parsePair(scanner.Text())
			if err != nil {
				disc.AppendErr(err)
			}
			pos, err := extractPosNum(kv.key())
			if err != nil {
				disc.AppendErr(err)
			}
			err = disc.AppendTrack(kv.value(), pos)
			if err != nil {
				disc.AppendErr(err)
			}
		case discYearRxp.Match([]byte(scanner.Text())):
			// collect year
			found := numRxp.Find([]byte(scanner.Text()))
			if string(found) == "" {
				continue
			}
			year, err := strconv.Atoi(string(found))
			if err != nil {
				disc.AppendErr(err)
			}
			castYear := uint16(year)
			disc.Year = &castYear
		case discGenreRxp.Match([]byte(scanner.Text())):
			// collect genre
			kv, err := parsePair(scanner.Text())
			if err != nil {
				disc.AppendErr(err)
			}
			genre := kv.value()
			if genre != "" {
				disc.Genre = &genre
			}
		}
	}
	return &disc
}
