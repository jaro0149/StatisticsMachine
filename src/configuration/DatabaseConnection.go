package configuration

import (
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"bytes"
)

// Attribute DB *gorm.DB - Database connection reference. See gorm.DB.
type DatabaseConnection struct {
	DB *gorm.DB
}

// Stale path to the database file (SQLite3 machine) - the same directory as application.
var databasePath = "database.db"

// Creating of instance that controls database connection.
// Returning *DatabaseConnection - database connection controller.
func NewDatabaseConnection() *DatabaseConnection {
	return &DatabaseConnection{}
}

// Opening of the database connection - initialising of DB variable.
func (DatabaseConnection *DatabaseConnection) ConnectDatabase() {
	tempDb, err := gorm.Open("sqlite3", databasePath)
	if err != nil {
		Error.Panic("Database connection cannot be created: ", err)
	}
	tempDb.LogMode(false)
	DatabaseConnection.DB = tempDb
	Info.Println("Databse connection is created.")
}

// Opening of the database connection from another path than from the main directory of application -
// it is useful when unit tests are executed so the unit test can find original database file that is
// located in one of the upper level directories.
// Parameter upperDirectoryLevels int - number of upper level directories that lead to path of the
// database file (from the path of unit test).
func (DatabaseConnection *DatabaseConnection) ConnectDatabaseFromTest(upperDirectoryLevels int) {
	Info.Println("Opening of the database connection.")
	var resultingStringBuffer bytes.Buffer
	for i:=0; i<upperDirectoryLevels; i++ {
		resultingStringBuffer.WriteString("../")
	}
	resultingStringBuffer.WriteString(databasePath)
	tempDb, err := gorm.Open("sqlite3", resultingStringBuffer.String())
	if err != nil {
		Error.Panic("Database connection cannot be created: ", err)
	}
	tempDb.LogMode(true)
	DatabaseConnection.DB = tempDb
	Info.Println("Database connection is created.")
}

// Closing of the database connection - DB is set to nil.
func (DatabaseConnection *DatabaseConnection) CloseDatabase() {
	Info.Println("Closing of the database connection.")
	err := DatabaseConnection.DB.Close()
	if err != nil {
		Error.Panic("Database connection cannot be closed: ", err)
	}
	DatabaseConnection.DB = nil
	Info.Println("Database connection is closed.")
}