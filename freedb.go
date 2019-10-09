// freedb.org
package freedb

// Disc represents the parsed output of a freeDB dump
type Disc struct {
	// ID is a non-unique algorithmically-generated hash identifying a compact
	// disc stored in hexidecimal.
	ID []uint8

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
