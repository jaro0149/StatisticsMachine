package model

import (
	"testing"
	"configuration"
	"io/ioutil"
	"os"
	"time"
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

// Unit tests preparation - initialisation of logging and database unit.
//
func setUp() {
	configuration.LoggingInit(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)
	databaseConnection = configuration.NewDatabaseConnection()
	databaseConnection.ConnectDatabaseFromTest(2)
	statMachine = NewStatisticalData()
}

// Cleaning after performing unit tests - closing of the database connection.
//
func tearDown() {
	cleanDatabases(nil)
	databaseConnection.CloseDatabase()
}

// Cleaning of the database - removing and recreating of all relations.
// Parameter t *testing.T - testing engine.
func cleanDatabases(t *testing.T) {
	err01 := configuration.DB.DropTableIfExists(&Data{}, &DataType{}, "data_to_types").Error
	if err01 != nil {
		t.Fatalf("Test failed while cleaning database: %s", err01)
	}
	err02 := configuration.DB.AutoMigrate(&DataType{}, &Data{}).Error
	if err02 != nil {
		t.Fatalf("Golang data model cannot be migrated to SQL: %s", err02)
	}
}

// Unit test - initialisation of the database relations or tables.
// Parameter t *testing.T - testing engine.
func TestTablesInit(t *testing.T) {
	t.Log("Initialisation of the database ...")
	statMachine.TablesInit()

	t.Log("Checking of created tables ...")
	dataCheck := configuration.DB.HasTable(&Data{})
	if !dataCheck {
		t.Errorf("Table 'data' hasn't been created.")
	}
	dataTypeCheck := configuration.DB.HasTable(&DataType{})
	if !dataTypeCheck {
		t.Errorf("Table 'data_types' hasn't been created.")
	}
	dataToTypesCheck := configuration.DB.HasTable("data_to_types")
	if !dataToTypesCheck {
		t.Errorf("Table 'data_to_types' hasn't been created.")
	}
}

// Unit test - modifying of already written data type.
// Parameter t *testing.T - testing engine.
func TestModifyDataType(t *testing.T) {
	t.Log("Cleaning of the database ...")
	cleanDatabases(t)

	t.Log("Writing of a new data type ...")
	name := "Type01"
	networkProtocol := 8000
	transportProtocol := 45
	port := 8080
	dataType := DataType{Name: name, Forecasting: false, NetworkProtocol: uint(networkProtocol),
		TransportProtocol: uint(transportProtocol), Port: uint(port)}
	writeDataType(&dataType, t)

	t.Log("Modifying of the valid data type ...")
	newName := "mod"
	newFormat := DataType{Name: newName, NetworkProtocol: 100, TransportProtocol: 20,
		Port: 22, Forecasting: true}
	err02 := statMachine.ModifyDataType(name, &newFormat)

	t.Log("Checking of the modified data type ...")
	if err02 != nil {
		t.Fatalf("The error was thrown while modifying of the data type: %s", err02)
	}
	tx := configuration.DB.Begin()
	var foundDataType DataType
	err03 := tx.Where(&DataType{Name: newName}).First(&foundDataType).Error
	if err03 != nil {
		tx.Rollback()
		t.Fatalf("An error occured during reading of the data type from database: %s", err03)
	}
	if foundDataType.Forecasting != newFormat.Forecasting {
		t.Errorf("Expected data type forecasting mod: %t; given forecasting mod: %t",
			newFormat.Forecasting, foundDataType.Forecasting)
	}
	if foundDataType.TransportProtocol != newFormat.TransportProtocol {
		t.Errorf("Expected data type transport protocol: %d; given transport protcol: %d",
			newFormat.Forecasting, foundDataType.Forecasting)
	}
	tx.Commit()

	t.Log("Writing of another data type ...")
	nextName := "TypeX"
	nextDataType := DataType{Name: nextName, NetworkProtocol: 10, TransportProtocol: 10, Port: 10}
	writeDataType(&nextDataType, t)

	t.Log("Writing of invalid modifications (failed unique constraint) ...")
	mod01 := DataType{Name: nextName}
	err05 := statMachine.ModifyDataType(newName, &mod01)
	if err05 == nil {
		t.Errorf("Expected error during writing of new data type (unique constrain failed), " +
			"but got nil error.")
	}
	mod02 := DataType{Name: "wtf", NetworkProtocol: uint(networkProtocol),
		TransportProtocol: uint(transportProtocol), Port: uint(port)}
	err06 := statMachine.ModifyDataType(nextName, &mod02)
	if err06 == nil {
		t.Errorf("Expected error during writing of new data type (unique constrain failed), " +
			"but got nil error.")
	}
}

// Unit test - writing of new data type.
// Parameter t *testing.T - testing engine.
func TestWriteNewDataType(t *testing.T) {
	t.Log("Cleaning of the database ...")
	cleanDatabases(t)

	t.Log("Writing of data types ...")
	dataTypes := []DataType{
		{Name: "Type01", Forecasting: false, NetworkProtocol: 10, TransportProtocol: 20, Port: 30},
		{Name: "Type02", NetworkProtocol: 44, TransportProtocol: 42, Port: 30},
		{Name: "Type03"},
		{ID: 10},
		{Name: "Type02", NetworkProtocol: 11, TransportProtocol: 42, Port: 30},
		{Name: "Type04", Forecasting: true, NetworkProtocol: 10, TransportProtocol: 20, Port: 30},
	}
	errorsListX := make([]error, len(dataTypes))
	for i, dataType := range dataTypes {
		errorsListX[i] = statMachine.WriteNewDataType(&dataType)
	}

	t.Log("Checking of errors after writing some new data types ...")
	trueErrors := []bool{false, false, false, true, true, true}
	for i, err := range errorsListX {
		if (err==nil) != (!trueErrors[i]) {
			t.Errorf("Expected error statement: %t; given error statement: %t",
				!trueErrors[i], err==nil)
		}
	}

	t.Log("Checking count of data types ...")
	expectedCount := 3
	tx := configuration.DB.Begin()
	var realCount int
	err01 := tx.Model(&DataType{}).Count(&realCount).Error
	if err01 != nil {
		tx.Rollback()
		t.Fatalf("An error occured during counting of data types: %s", err01)
	}
	if expectedCount != realCount {
		t.Errorf("Expected count of data types: %d; given count of data types: %d",
			expectedCount, realCount)
	}
	tx.Commit()
}

// Unit test - writing of new data entries.
// Parameter t *testing.T - testing engine.
func TestWriteNewDataEntries(t *testing.T) {
	t.Log("Cleaning of the database ...")
	cleanDatabases(t)

	t.Log("Writing of new data types into the database ...")
	dataTypes := make([]*DataType, 4)
	dataTypes[0] = &DataType{Name: "COM01", Forecasting: false, NetworkProtocol: 200, TransportProtocol: 45, Port: 8080}
	dataTypes[1] = &DataType{Name: "COM02", Forecasting: false, NetworkProtocol: 200, TransportProtocol: 45, Port: 0}
	dataTypes[2] = &DataType{Name: "COM03", Forecasting: false, NetworkProtocol: 200, TransportProtocol: 0, Port: 0}
	dataTypes[3] = &DataType{Name: "COM04", Forecasting: false, NetworkProtocol: 0, TransportProtocol: 0, Port: 0}
	writeNewDataTypes(&dataTypes, t)

	t.Log("Writing of new raw data into the database ...")
	rawData := []*RawData{
		{Bytes: 8, NetworkProtocol: 45, TransportProtocol: 11, SrcPort: 2, DstPort: 2},
		{Bytes: 15, NetworkProtocol: 200, TransportProtocol: 3, SrcPort: 15, DstPort: 15},
		{Bytes: 789, NetworkProtocol: 200, TransportProtocol: 45, SrcPort: 80, DstPort: 80},
		{Bytes: 454, NetworkProtocol: 200, TransportProtocol: 45, SrcPort: 8080, DstPort: 8080},
		{Bytes: 1, NetworkProtocol: 200, TransportProtocol: 60, SrcPort: 45, DstPort: 45},
		{Bytes: 2, NetworkProtocol: 450, TransportProtocol: 22, SrcPort: 8025, DstPort: 8025},
		{Bytes: 1200, NetworkProtocol: 200, TransportProtocol: 45, SrcPort: 80, DstPort: 80}}
	statMachine.WriteNewDataEntries(&rawData)

	t.Log("Searching for written data with filled IDs ...")
	completedData := getAllData(t)

	t.Log("Verification of written data ...")
	tx := configuration.DB.Begin()
	trueCounts := []int{1, 2, 3, 4, 2, 1, 3}
	for i := range trueCounts {
		var associatedTypes []DataType
		count := tx.Model((*completedData)[i]).
			Association("DataTypes").
			Find(&associatedTypes).
			Count()
		if count != trueCounts[i] {
			t.Errorf("Expected count of data types: %d, given count of data types: %d", trueCounts[i], count)
		}
	}
	tx.Commit()
}

// Unit test - searching for all data types.
// Parameter t *testing.T - testing engine.
func TestListDataTypes(t *testing.T) {
	t.Log("Cleaning of the database ...")
	cleanDatabases(t)

	t.Log("Writing of some data types ...")
	dataTypes := [](*DataType){
		&DataType{Name: "Type01", Forecasting: false, NetworkProtocol: 10, TransportProtocol: 20, Port: 30},
		&DataType{Name: "Type02", NetworkProtocol: 44, TransportProtocol: 42, Port: 30},
		&DataType{Name: "Type03"},
	}
	writeNewDataTypes(&dataTypes, t)

	t.Log("Reading of the data types list ...")
	realDataTypes := statMachine.ListDataTypes()

	t.Log("Comparing of awaited data types against real data types ...")
	for i := range dataTypes {
		if dataTypes[i].Name != ((*realDataTypes)[i]).Name {
			t.Errorf("Expected data type name: %s; given data type name: %s",
				dataTypes[i].Name, ((*realDataTypes)[i]).Name)
		}
	}
}

// Unit test - reading of the data type by specific name.
// Parameter t *testing.T - testing engine.
func TestGetDataType(t *testing.T) {
	t.Log("Cleaning of the database ...")
	cleanDatabases(t)

	t.Log("Writing of a new data type ...")
	name := "Type01"
	networkProtocol := 8000
	transportProtocol := 45
	port := 8080
	dataType := DataType{Name: name, Forecasting: false, NetworkProtocol: uint(networkProtocol),
		TransportProtocol: uint(transportProtocol), Port: uint(port)}
	tx := configuration.DB.Begin()
	err := tx.Create(&dataType).Error
	if err != nil {
		tx.Rollback()
		t.Fatalf("Test failed while creating of a new data type: %s", err)
	}
	tx.Commit()

	t.Log("Reading of the data type by name ...")
	tx = configuration.DB.Begin()
	fetchedDataType, err01 := statMachine.GetDataType(name)
	if err01 != nil {
		t.Errorf("An error occured during reading of the data type from datatabase: %s", err)
	}
	if fetchedDataType.ID == 0 {
		t.Errorf("Unexpected fetched data type ID: %d", fetchedDataType.ID)
	}
	if fetchedDataType.Port != uint(port) {
		t.Errorf("Expected data type port: %d; given data type port: %d", port, fetchedDataType.Port)
	}
	if fetchedDataType.NetworkProtocol != uint(networkProtocol) {
		t.Errorf("Expected data type network protocol: %d; given data type network protocol: %d",
			networkProtocol, fetchedDataType.NetworkProtocol)
	}
	tx.Commit()

	t.Log("Testing fetching of invalid data type (by name) ...")
	tx = configuration.DB.Begin()
	invalidName := "invalid"
	_, err02 := statMachine.GetDataType(invalidName)
	if err02 == nil {
		t.Errorf("An error is expected during reading of unknown data type from database but nil " +
			"error is thrown.")
	}
	tx.Commit()
}

// Unit test - removing of the data type from database.
// Parameter t *testing.T - testing engine.
func TestRemoveDataType(t *testing.T) {
	t.Log("Cleaning of the database ...")
	cleanDatabases(t)

	t.Log("Writing of a new data type ...")
	dataTypeName := "XXX"
	dataType := DataType{Name: dataTypeName}
	writeDataType(&dataType, t)

	t.Log("Writing of some data ...")
	data := [](*Data){
		&Data{Time: time.Now(), Bytes: 10},
		&Data{Time: time.Now(), Bytes: 25},
	}
	writeData(&data, t)

	t.Log("Writing of associations between data type and data ...")
	createAssociationDataTypeData(&dataType, &data, t)

	t.Log("Removing of the data type ...")
	err02 := statMachine.RemoveDataType(dataTypeName)
	if err02 != nil {
		t.Errorf("The removal of the data type failed: %s", err02)
	}

	t.Log("Checking of the data type removal ...")
	dataEntriesCount := len(*getAllData(t))
	if dataEntriesCount != 0 {
		t.Errorf("Expected data entries count: 0; got data entries: %d", dataEntriesCount)
	}
	dataTypesCount := len(*getAllDataTypes(t))
	if dataTypesCount != 0 {
		t.Errorf("Expected data types count: 0; got count: %d", dataTypesCount)
	}

	t.Log("Removing of invalid data type ...")
	err := statMachine.RemoveDataType("invalid")
	if err == nil {
		t.Errorf("Expected error during removing of unknown data type but got nil error.")
	}
}

// Unit test - listing of last data entries by entered time.
// Parameter t *testing.T - testing engine.
func TestListLastDataEntries(t *testing.T) {
	t.Log("Cleaning of the database ...")
	cleanDatabases(t)

	t.Log("Writing of a new data type ...")
	dataTypeName := "type01"
	dataType := DataType{Name: dataTypeName}
	writeDataType(&dataType, t)

	t.Log("Writing of some data #1 ...")
	data01 := [](*Data) {
		&Data{Bytes: 10, Time: time.Now()},
		&Data{Bytes: 10, Time: time.Now()},
	}
	writeData(&data01, t)

	t.Log("Creating of the timestamp before sleep ...")
	timestamp := time.Now()
	time.Sleep(1 * time.Second)
	dataBytes := 20

	t.Log("Writing of some data #2 ...")
	data02 := [](*Data) {
		&Data{Bytes: uint(dataBytes), Time: time.Now()},
		&Data{Bytes: uint(dataBytes), Time: time.Now()},
	}
	writeData(&data02, t)

	t.Log("Creating of the associations between the data type and data ...")
	createAssociationDataTypeData(&dataType, &data01, t)
	createAssociationDataTypeData(&dataType, &data02, t)

	t.Log("Fetching of last data entries ...")
	lastData, err01 := statMachine.ListLastDataEntries(dataTypeName, timestamp)
	if err01 != nil {
		t.Fatalf("Last data entries cannot be fetched from database: %s", err01)
	}
	if len(*lastData) != len(data02) {
		t.Errorf("Expected number of data entries: %d; got number of data entries: %d",
			len(data02), len(*lastData))
	} else {
		if (*lastData)[0].Bytes != uint(dataBytes) {
			t.Errorf("Expected number of data bytes: %d; got number of data bytes: %d",
				dataBytes, (*lastData)[0].Bytes)
		}
		if (*lastData)[1].Bytes != uint(dataBytes) {
			t.Errorf("Expected number of data bytes: %d; got number of data bytes: %d",
				dataBytes, (*lastData)[1].Bytes)
		}
	}

	t.Log("Reading with the invalid data type ...")
	_, err02 := statMachine.ListLastDataEntries("fake", timestamp)
	if err02 == nil {
		t.Errorf("An error was expected during reading of last data entries bounded to " +
			"invalid data type but nil error is thrown.")
	}
}

// Unit test - removing of old data entries.
// Parameter t *testing.T - testing engine.
func TestRemoveOldDataEntries(t *testing.T) {
	t.Log("Cleaning of the database ...")
	cleanDatabases(t)

	t.Log("Writing of a new data type ...")
	dataTypeName := "type01"
	dataType := DataType{Name: dataTypeName}
	writeDataType(&dataType, t)

	t.Log("Writing of some data #1 ...")
	data01 := [](*Data) {
		&Data{Bytes: 10, Time: time.Now()},
		&Data{Bytes: 10, Time: time.Now()},
	}
	writeData(&data01, t)

	t.Log("Creating of the timestamp before sleep ...")
	timestamp := time.Now()
	time.Sleep(1 * time.Second)
	dataBytes := 20

	t.Log("Writing of some data #2 ...")
	data02 := [](*Data) {
		&Data{Bytes: uint(dataBytes), Time: time.Now()},
		&Data{Bytes: uint(dataBytes), Time: time.Now()},
	}
	writeData(&data02, t)

	t.Log("Creating of the associations between the data type and data ...")
	createAssociationDataTypeData(&dataType, &data01, t)
	createAssociationDataTypeData(&dataType, &data02, t)

	t.Log("Removing of old data entries ...")
	statMachine.RemoveOldDataEntries(timestamp)

	t.Log("Checking of data entries ...")
	allData := getAllData(t)
	if len(*allData) != len(data02) {
		t.Errorf("Expected number of data entries: %d; got number of data entries: %d",
			len(data02), len(*allData))
	} else {
		if (*allData)[0].Bytes != uint(dataBytes) {
			t.Errorf("Expected number of data bytes: %d; got number of data bytes: %d",
				dataBytes, (*allData)[0].Bytes)
		}
		if (*allData)[1].Bytes != uint(dataBytes) {
			t.Errorf("Expected number of data bytes: %d; got number of data bytes: %d",
				dataBytes, (*allData)[1].Bytes)
		}
	}

	t.Log("Checking of associations ...")
	tx := configuration.DB.Begin()
	var associatedData [](*Data)
	err := tx.Model(&dataType).Association("Data").Find(&associatedData).Error
	if err != nil {
		tx.Rollback()
		t.Fatalf("Cannot find associated data with specific data type: %s", err)
	}
	for _, d := range associatedData {
		if d.Bytes != uint(dataBytes) {
			t.Errorf("Expected number of data bytes: %d; got number of data bytes: %d",
				dataBytes, d.Bytes)
		}
	}
	tx.Commit()
}

// Writing of new data types into the database.
// Parameter dataTypes *[](*DataType) - the slice with data types.
// Parameter t *testing.T - testing engine.
func writeNewDataTypes(dataTypes *[](*DataType), t *testing.T) {
	tx := configuration.DB.Begin()
	for _, dataType := range *dataTypes {
		err := configuration.DB.Create(dataType).Error
		if err != nil {
			tx.Rollback()
			t.Fatalf("Test failed while creating of new data types: %s", err)
		}
	}
	tx.Commit()
}

// Reading of all data from the database.
// Parameter t *testing.T - testing engine.
// Returning *[](*Data) - fetched data entries.
func getAllData(t *testing.T) *[](*Data) {
	tx := configuration.DB.Begin()
	var data []*Data
	err := tx.Order("id asc").Find(&data).Error
	if err != nil {
		tx.Rollback()
		t.Fatalf("Test failed while reading of written data entries: %s", err)
	}
	tx.Commit()
	return &data
}

// Listing of all data types.
// Parameter t *testing.T - testing engine.
// Returning *[](*DataType) - slice filled with all data types.
func getAllDataTypes(t *testing.T) *[](*DataType) {
	tx := configuration.DB.Begin()
	var dataTypes [](*DataType)
	err := tx.Find(&dataTypes).Error
	if err != nil {
		tx.Rollback()
		t.Fatalf("An error occured during fetching of data types: %s", err)
	}
	tx.Commit()
	return &dataTypes
}

// Writing of the single data type into the database.
// Parameter dataType *DataType - data type entry.
// Parameter t *testing.T - testing engine.
func writeDataType(dataType *DataType, t *testing.T) {
	tx := configuration.DB.Begin()
	err := tx.Create(dataType).Error
	if err != nil {
		tx.Rollback()
		t.Fatalf("Test failed while creating of a new data type: %s", err)
	}
	tx.Commit()
}

// Writing of some data into the database relation.
// Parameter data *[](*Data) - data that is going to be written into the database.
// Parameter t *testing.T - testing engine.
func writeData(data *[](*Data), t* testing.T) {
	tx := configuration.DB.Begin()
	for _,d := range *data {
		err := tx.Create(&d).Error
		if err != nil {
			tx.Rollback()
			t.Fatalf("Test failed while writing of new data: %s", err)
		}
	}
	tx.Commit()
}

// Creating of associations: one data type - a lot of data.
// Parameter dataType *DataType - the data type that is going to be associated with data.
// Parameter data *[](*Data) - sample data entries.
// Parameter t *testing.T - testing engine.
func createAssociationDataTypeData(dataType *DataType, data *[](*Data), t *testing.T) {
	tx := configuration.DB.Begin()
	err := tx.Model(dataType).Association("Data").Append(data).Error
	if err != nil {
		tx.Rollback()
		t.Errorf("An error occured during creating of associations between the data type " +
			"and data: %s", err)
	}
	tx.Commit()
}