package main

import (
	"database/sql"
	"fmt"
	"os"
)

func initDB() (*sql.DB, error) {
	var dberr error

	dbuser := os.Getenv("dbuser")
	dbpassword := os.Getenv("dbpassword")
	dbname := os.Getenv("dbname")
	dbhost := os.Getenv("dbhost")

	dbURI := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%s", dbhost, dbuser, dbname, dbpassword)

	db, dberr := sql.Open("postgres", dbURI)

	return db, dberr
}

func closeDB(db *sql.DB) {
	db.Close()
}
