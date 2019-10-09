package dbdump

import (
	"encoding/hex"
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

func TestParseDump(t *testing.T) {
	f, err := os.Open("7908090a")
	if err != nil {
		t.Fatal(err)
	}
	disc := ParseDump(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(spew.Sdump(disc))
}

func TestHex(t *testing.T) {
	id := "7908090a"
	f, err := os.Open(id)
	if err != nil {
		t.Fatal(err)
	}
	disc := ParseDump(f)
	tmpl := "expected:%s got:%s"
	enc := hex.EncodeToString(disc.ID)
	if id != enc {
		t.Fatalf(tmpl, id, enc)
	}
	t.Logf(tmpl, id, enc)
}
