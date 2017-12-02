package model

import (
	"testing"
	"os"
	"io/ioutil"
	"configuration"
)

// Statistical operations.
var statMachine *StatisticalData

// Database connection.
var databaseConnection *configuration.DatabaseConnection

// Scheduling of setup, unit tests and tear-down functions. See testing.M
// Parameter m *testing.M - unit tests machine.
func TestMain(m *testing.M) {
	setUp()
	retCode := m.Run()
	tearDown()
	os.Exit(retCode)
}

// Unit tests preparation - initialisation of logging and database unit and tested objects.
//
func setUp() {
	configuration.LoggingInit(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)
	databaseConnection = configuration.NewDatabaseConnection()
	databaseConnection.ConnectDatabaseFromTest(2)
	statMachine = NewStatisticalData(databaseConnection)
}

// Cleaning after performing unit tests - closing of the database connection.
//
func tearDown() {
	cleanDatabases(nil)
	databaseConnection.CloseDatabase()
}