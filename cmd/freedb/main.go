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
	"path"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"

	// blank import of pgx for database/sql driver
	_ "github.com/jackc/pgx/stdlib"

	"github.com/alrs/freedb/dbdump"
)

const insertDiscStmt = "INSERT INTO discs (freedb_id, title) VALUES ($1, $2) RETURNING id;"
const insertTrackStmt = "INSERT INTO tracks (disc_id, title) VALUES ($1, $2);"

var ignoreFiles = []string{"COPYING", "README"}

func openDB(u url.URL) (*sql.DB, error) {
	db, err := sql.Open("pgx", u.String())
	if err != nil {
		return db, fmt.Errorf("error connecting to database: %s")
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

func ingestFromFilesystem(tx *sql.Tx, dumpPath string) error {
	insertDisc, err := prepareStatement(tx, insertDiscStmt)
	if err != nil {
		return err
	}
	insertTrack, err := prepareStatement(tx, insertTrackStmt)
	if err != nil {
		return err
	}
	parseFile := func(fqp string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if info.Size() < 1 {
			log.Printf("ignoring tiny file: %s", fqp)
			return nil
		}
		for _, fn := range ignoreFiles {
			_, file := path.Split(fqp)
			if file == fn {
				log.Printf("ignoring blacklisted file: %s", fqp)
				return nil
			}
		}

		f, err := os.Open(fqp)
		if err != nil {
			return err
		}
		defer f.Close()

		dump := dbdump.ParseDump(f)
		if dump.ID == nil {
			log.Printf("ignoring nil entry: %s", fqp)
			return nil
		}

		row := insertDisc.QueryRow(dump.ID, dump.Title)
		var id int
		err = row.Scan(&id)
		if err != nil {
			log.Fatalf("error inserting disc %s %s to db: %s",
				hex.EncodeToString(dump.ID), dump.Title, err)
		}

		for _, track := range dump.Tracks {
			_, err := insertTrack.Exec(id, track)
			if err != nil {
				log.Fatalf("error inserting track %s from %s: %s", track, dump.ID, err)
			}
		}
		return nil
	}

	return filepath.Walk(dumpPath, parseFile)
}

func ingest(r io.Reader, fi os.FileInfo, inserts map[string]*sql.Stmt) error {
	if fi.Mode().IsDir() {
		// don't attempt to parse a directory
		return nil
	}
	for _, fn := range ignoreFiles {
		if fi.Name() == fn {
			log.Printf("ignoring blacklist file: %s", fi.Name())
			return nil
		}
	}
	dump := dbdump.ParseDump(r)
	if dump.ID == nil {
		log.Printf("ignoring nil entry %s: %s", fi.Name(), spew.Sdump(dump))
		return nil
	}

	row := inserts["disc"].QueryRow(dump.ID, dump.Title)
	var id int
	err := row.Scan(&id)
	if err != nil {
		return fmt.Errorf("error inserting disc %s %s to db: %s",
			fi.Name(), dump.Title, err)
	}

	for _, track := range dump.Tracks {
		_, err := inserts["track"].Exec(id, track)
		if err != nil {
			log.Fatalf("error inserting track %s from %s: %s", track, dump.ID, err)
		}
	}
	return nil
}

func prepareInserts(tx *sql.Tx) (map[string]*sql.Stmt, error) {
	var err error
	inserts := make(map[string]*sql.Stmt)
	inserts["disc"], err = prepareStatement(tx, insertDiscStmt)
	if err != nil {
		return inserts, err
	}
	inserts["track"], err = prepareStatement(tx, insertTrackStmt)
	return inserts, err
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
			err := ingest(tarball, header.FileInfo(), inserts)
			if err != nil {
				return err
			}
		}
	}
	return nil
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
		log.Fatal("error determining FileInfo on %s: %s", dumpPath, err)
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
