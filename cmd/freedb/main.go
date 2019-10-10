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
	"path/filepath"

	// blank import of pgx for database/sql driver
	_ "github.com/jackc/pgx/stdlib"

	"github.com/alrs/freedb/dbdump"
)

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

	insertDisc, err := prepareStatement(tx,
		"INSERT INTO discs (freedb_id, title) VALUES ($1, $2) RETURNING id;")

	if err != nil {
		log.Fatal(err)
	}

	insertTrack, err := prepareStatement(tx,
		"INSERT INTO tracks (disc_id, title) VALUES ($1, $2);")
	if err != nil {
		log.Fatal(err)
	}

	parseFile := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || info.Size() < 10 {
			log.Printf("ignoring: %s", path)
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		dump := dbdump.ParseDump(f)
		if dump.ID == nil {
			log.Printf("ignoring: %s", path)
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

	err = filepath.Walk(dumpPath, parseFile)
	if err != nil {
		log.Fatal(err)
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
}
