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

package main

import (
	"archive/tar"
	"compress/bzip2"
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/davecgh/go-spew/spew"

	// blank import of pgx for database/sql driver
	_ "github.com/jackc/pgx/stdlib"

	"github.com/alrs/freedb"
)

var ignoreFiles = []string{"COPYING", "README"}

func openDB(u url.URL) (*sql.DB, error) {
	db, err := sql.Open("pgx", u.String())
	if err != nil {
		return db, fmt.Errorf("error connecting to database: %s", err)
	}
	return db, nil
}

func prepareStatement(tx *sql.Tx, template string) (*sql.Stmt, error) {
	insert, err := tx.Prepare(template)
	if err != nil {
		return insert, fmt.Errorf("error templating sql statement %s: %s",
			template, err)
	}
	return insert, nil
}

func ingestFromTarball(tx *sql.Tx, dumpPath string) error {
	inserts, err := prepareInserts(tx)
	if err != nil {
		return err
	}
	f, err := os.Open(dumpPath)
	if err != nil {
		return err
	}
	bz2 := bzip2.NewReader(f)
	tarball := tar.NewReader(bz2)
Loop:
	for {
		header, err := tarball.Next()
		switch {
		case err == io.EOF:
			break Loop
		case err != nil:
			return err
		default:
			if ignorable(header.FileInfo()) {
				continue
			}
			err := ingest(tarball, header.Name, inserts)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func extractShardName(fqp string) (string, error) {
	split := strings.Split(fqp, "/")
	if len(split) >= 2 {
		return split[len(split)-2], nil
	}
	return "", fmt.Errorf("no shard name in path string: %s", fqp)
}

func extractID(fqp string) ([]uint8, error) {
	_, fn := filepath.Split(fqp)
	return hex.DecodeString(fn)
}

func ignorable(info os.FileInfo) bool {
	if info.Mode().IsDir() {
		// ignore directories
		return true
	}
	for _, fn := range ignoreFiles {
		if info.Name() == fn {
			log.Printf("ignoring blacklist file: %s", info.Name())
			return true
		}
	}
	return false
}

func ingestFromFilesystem(tx *sql.Tx, dumpPath string) error {
	inserts, err := prepareInserts(tx)
	if err != nil {
		return err
	}

	parseFunc := func(fqp string, info os.FileInfo, err error) error {
		fi, err := os.Stat(fqp)
		if err != nil {
			return err
		}
		if ignorable(fi) {
			return nil
		}

		f, err := os.Open(fqp)
		if err != nil {
			return err
		}
		defer f.Close()

		return ingest(f, fqp, inserts)
	}

	return filepath.Walk(dumpPath, parseFunc)
}

func ingest(r io.Reader, fqp string, inserts map[string]*sql.Stmt) error {
	shardName, err := extractShardName(fqp)
	if err != nil {
		return err
	}
	shard, err := freedb.ShardPos(shardName)
	if err != nil {
		return err
	}
	shortID, err := extractID(fqp)
	if err != nil {
		return err
	}
	dump, err := freedb.ParseDump(r, shard)
	if err != nil {
		log.Printf("parsing '%s' resulted in error:%s, ignoring %s", fqp, err, spew.Sdump(dump))
		return nil
	}
	_, err = inserts["disc"].Exec(freedb.ComposeUID(shortID, shard), dump.Title)
	if err != nil {
		return fmt.Errorf("error inserting disc %s %s to db: %s",
			fqp, fqp, err)
	}
	for _, track := range dump.Tracks {
		_, err := inserts["track"].Exec(freedb.ComposeUID(shortID, shard), track)
		if err != nil {
			log.Fatalf("error inserting track %s from %s: %s", track, fqp, err)
		}
	}
	return nil
}

func prepareInserts(tx *sql.Tx) (map[string]*sql.Stmt, error) {
	var err error
	insertDisc := "INSERT INTO discs (id, title) VALUES ($1, $2);"
	insertTrack := "INSERT INTO tracks (disc_id, title) VALUES ($1, $2);"

	inserts := make(map[string]*sql.Stmt)
	inserts["disc"], err = prepareStatement(tx, insertDisc)
	if err != nil {
		return inserts, err
	}
	inserts["track"], err = prepareStatement(tx, insertTrack)
	return inserts, err
}

func main() {
	var user, password, host, dbName, dumpPath string
	var dbPort int

	flag.StringVar(&user, "user", "", "postgresql username")
	flag.StringVar(&password, "pass", "", "postgresql password")
	flag.StringVar(&host, "host", "localhost", "postgresql hostname")
	flag.StringVar(&dbName, "db", "freedb", "postgresql database name")
	flag.StringVar(&dumpPath, "dump", "", "path to database dump")
	flag.IntVar(&dbPort, "port", 5432, "postgresql port number")
	flag.Parse()

	pgURI := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   fmt.Sprintf("%s:%d", host, dbPort),
		Path:   dbName,
	}

	db, err := openDB(pgURI)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("error on Begin(): %s", err)
	}

	fi, err := os.Stat(dumpPath)
	if err != nil {
		log.Fatalf("error determining FileInfo on %s: %s", dumpPath, err)
	}

	switch mode := fi.Mode(); {
	case mode.IsDir():
		// dump has already been decompressed and untarred to filesystem
		err = ingestFromFilesystem(tx, dumpPath)
		if err != nil {
			tx.Rollback()
			log.Fatalf("error ingesting from filesystem: %s", err)
		}
	case mode.IsRegular():
		// operate directly on bzip2 tarball
		err = ingestFromTarball(tx, dumpPath)
		if err != nil {
			tx.Rollback()
			log.Fatalf("error ingesting from tarball: %s", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Fatalf("error commiting database transaction: %s", err)
	}

}
