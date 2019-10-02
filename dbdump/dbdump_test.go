package dbdump

import (
	"os"
	"reflect"
	"testing"

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

func TestCollectOffsets(t *testing.T) {
	f, err := os.Open("7908090a")
	if err != nil {
		t.Fatal(err)
	}
	offsets, err := collectOffsets(f)
	if err != nil {
		t.Fatal(err)
	}
	expected := []uint32{
		150,
		12860,
		23460,
		37067,
		54250,
		70892,
		90637,
		107507,
		124742,
		139870,
	}

	logTmpl := "expected:%v got:%v"
	if !reflect.DeepEqual(expected, offsets) {
		t.Fatalf(logTmpl, expected, offsets)
	}
	t.Logf(logTmpl, expected, offsets)
}

func TestCollectDiscLength(t *testing.T) {
	f, err := os.Open("7908090a")
	if err != nil {
		t.Fatal(err)
	}
	length, err := collectDiscLength(f)
	if err != nil {
		t.Fatal(err)
	}
	expected := uint16(2059)
	logTmpl := "expected:%d got:%d"
	if expected != length {
		t.Fatalf(logTmpl, expected, length)
	}
	t.Logf(logTmpl, expected, length)
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

func TestCollectTracks(t *testing.T) {
	f, err := os.Open("7908090a")
	if err != nil {
		t.Fatal(err)
	}
	tracks, err := collectTracks(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(spew.Sdump(tracks))
}
