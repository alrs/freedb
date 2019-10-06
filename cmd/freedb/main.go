package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/alrs/freedb/dbdump"
	"github.com/davecgh/go-spew/spew"
)

func main() {
	walkFunc := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		spew.Dump(dbdump.ParseDump(f))
		return nil
	}

	err := filepath.Walk("/home/lars/freedb", walkFunc)
	if err != nil {
		log.Fatal(err)
	}
}
