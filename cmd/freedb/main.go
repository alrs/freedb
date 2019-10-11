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
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"

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
	err = ingestFromFilesystem(tx, dumpPath)
	if err != nil {
		tx.Rollback()
		log.Fatalf("error ingesting from filesystem: %s", err)
	}

	err = tx.Commit()
	if err != nil {
		log.Fatalf("error commiting database transaction: %s", err)
	}

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
		if info.Size() < 10 {
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
