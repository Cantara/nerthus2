package main

import (
	"database/sql"
	_ "github.com/microsoft/go-mssqldb"
	"time"
)

func MSSQLVersion(conn *sql.DB) (version string, err error) {
	row := conn.QueryRow(`SELECT SERVERPROPERTY('productversion')`)
	err = row.Scan(&version)
	if err != nil {
		return
	}
	return
}
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
