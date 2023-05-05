package main

import (
	"database/sql"
	_ "github.com/denisenkom/go-mssqldb"
	"time"
)

func MSSQLStatus(conn *sql.DB) (out baseStatus, err error) {
	row := conn.QueryRow(`
		SELECT
		  count(*)
		FROM sys.databases
		WHERE state != 0;`)
	var count int
	err = row.Scan(&count)
	if err != nil {
		return
	}
	status := "DOWN"
	if count == 0 {
		status = "UP"
	}
	out = baseStatus{
		Status:  status,
		Name:    "mssql",
		Version: version,
		IP:      GetOutboundIP(),
		Now:     time.Now(),
	}

	/*
		SELECT
		  count(*)
		FROM sys.databases
		WHERE state != 0;
	*/
	return
}
