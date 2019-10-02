package dbdump

import (
	"os"
	"reflect"
	"testing"
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
