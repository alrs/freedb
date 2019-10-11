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

// Package freedb can parse freedb.org dumps and insert into PostgreSQL.
package freedb

import (
	"fmt"
	"strings"
)

// These shard names are meaningless and vestigal, but are needed to
// disambiguate the freedb.org DISCID
var Shards = []string{
	"blues",
	"classical",
	"country",
	"data",
	"folk",
	"jazz",
	"misc",
	"newage",
	"reggae",
	"rock",
	"soundtrack",
}

// Disc represents the parsed output of a freeDB dump
type Disc struct {
	// IDs is a list of non-unique algorithmically-generated hashes identifying
	// a compact disc stored in hexadecimal.
	IDs [][]uint8

	// Shard is an 8 bit unsigned int that represents the position of the shard
	// in the Shards slice. These shards look like genre subdirectories, but
	// that meaning is vestigal. The subdirectories are used as generic shard
	// buckets to work around ID collisions.
	Shard uint8

	// Title is the combined artist name and release name of a compact disc.
	Title string

	// Genre is an optional field that represents the genre of music found
	// on a compact disc.
	Genre *string

	// Year is an optional field denoting the release year of a compact disc.
	Year *uint16

	// Offsets are the positions on a CD where tracks begin.
	Offsets []uint32

	// Duration of the compact disc, in seconds.
	Duration uint16

	// Tracks is a slice of strings representing the track titles on a compact
	// disc.
	Tracks []string

	// ParseErrors are all of the errors accumulated during parsing of a dump
	// file.
	ParseErrors []error
}

// AppendErr appends a parsing error to a Disc and returns the number of
// collected errors on the Disc object.
func (d *Disc) AppendErr(err error) int {
	d.ParseErrors = append(d.ParseErrors, err)
	return len(d.ParseErrors)
}

// AppendTitle appends to the string that represents the compact disc title,
// ensuring that the string is valid UTF8.
func (d *Disc) AppendTitle(s string) {
	d.Title = d.Title + strings.ToValidUTF8(s, "")
}

// AppendTrack appends to the track title in the given slice position,
// ensuring that the string is valid UTF8.
func (d *Disc) AppendTrack(s string, pos int) error {
	switch {
	case len(d.Tracks) == pos:
		d.Tracks = append(d.Tracks, strings.ToValidUTF8(s, ""))
	case len(d.Tracks) == pos+1:
		d.Tracks[pos] = d.Tracks[pos] + strings.ToValidUTF8(s, "")
	default:
		return fmt.Errorf("attempted to append position %d to %d length slice",
			pos, len(d.Tracks))
	}
	return nil
}

// ShardErr is the response to an unknown shard name.
type ShardErr struct {
	shard string
}

// Error provides an error string for the ShardErr type.
func (e *ShardErr) Error() string {
	return fmt.Sprintf("unknown shard name: %s", e.shard)
}

// ShardPos returns the shard position of a named shard as well as a ShardErr
// if the named shard does not exist.
func ShardPos(name string) (uint8, error) {
	for i, s := range Shards {
		if name == s {
			return uint8(i), nil
		}
	}
	return 0, &ShardErr{name}
}

// ComposeUID concatenates the non-unique DISCID with the unique shard number
// to create an actually unique ID.
func ComposeUID(discID []uint8, shard uint8) []uint8 {
	return append(discID, uint8(shard))
}
