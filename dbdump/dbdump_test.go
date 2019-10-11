package dbdump

import (
	"encoding/hex"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/alrs/freedb"

	"github.com/davecgh/go-spew/spew"
)

func TestParseOffset(t *testing.T) {
	line := `#       139870`
	offset, err := parseOffset(line)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(offset)
}

func TestParsePair(t *testing.T) {
	testPair := "key=value"
	kv, err := parsePair(testPair)
	if err != nil {
		t.Fatal(err)
	}
	logTmpl := "expected:%s got:%s"
	expected := pair{"key", "value"}
	if !reflect.DeepEqual(kv, expected) {
		t.Fatalf(logTmpl, expected, kv)
	}
	t.Logf(logTmpl, expected, kv)

	expectedKey := "key"
	if kv.key() != expectedKey {
		t.Fatalf(logTmpl, expectedKey, kv.key())
	}
	t.Logf(logTmpl, expectedKey, kv.key())

	expectedValue := "value"
	if kv.value() != expectedValue {
		t.Fatalf(logTmpl, expectedValue, kv.value())
	}
	t.Logf(logTmpl, expectedValue, kv.value())
}

func TestExtractPosNumber(t *testing.T) {
	test := "SOMETHING3"
	expected := 3
	got, err := extractPosNum(test)
	if err != nil {
		t.Fatal(err)
	}
	logTmpl := "expected:%d got:%d"
	if expected != got {
		t.Fatalf(logTmpl, expected, got)
	}
	t.Logf(logTmpl, expected, got)
}

func TestHex(t *testing.T) {
	id := "decafbad"
	f, err := os.Open(path.Join("test", id))
	if err != nil {
		t.Fatal(err)
	}
	shard, err := freedb.ShardPos("soundtrack")
	if err != nil {
		t.Fatal(err)
	}
	disc := ParseDump(f, shard)
	tmpl := "expected:%s got:%s"
	enc := hex.EncodeToString(disc.IDs[0])
	if id != enc {
		t.Fatalf(tmpl, id, enc)
	}
	t.Logf(tmpl, id, enc)
}

func TestParseDump(t *testing.T) {
	testDump := "decafbad"
	f, err := os.Open(path.Join("test", testDump))
	if err != nil {
		t.Fatal(err)
	}
	shard, err := freedb.ShardPos("soundtrack")
	if err != nil {
		t.Fatal(err)
	}
	disc := ParseDump(f, shard)
	if err != nil {
		t.Fatal(err)
	}
	exID, err := hex.DecodeString(testDump)
	if err != nil {
		t.Fatal(err)
	}
	exGenre := "Math Rock"
	exYear := uint16(2005)
	if err != nil {
		t.Fatal(err)
	}
	expected := &freedb.Disc{
		IDs:      [][]uint8{exID},
		Shard:    10,
		Genre:    &exGenre,
		Year:     &exYear,
		Title:    "The Nameless Faceless Many / dot dot E.P.",
		Offsets:  []uint32{100, 200, 60000},
		Duration: uint16(2000),
		Tracks: []string{
			"Introduction",
			"So Long That It Wraps Around",
		},
	}
	tmpl := "expected:%s got:%s"
	if !reflect.DeepEqual(expected, disc) {
		t.Fatalf(tmpl, spew.Sdump(expected), spew.Sdump(disc))
	}
	t.Log(spew.Sdump(disc))
}
