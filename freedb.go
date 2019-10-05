package freedb

type Disc struct {
	ID          string
	Title       string
	Genre       string
	Year        *uint16
	Offsets     []uint32
	Duration    uint16
	Tracks      []string
	ParseErrors []error
}
