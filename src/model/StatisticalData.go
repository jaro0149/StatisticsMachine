package model

import (
	"time"
	"configuration"
	"strings"
	"fmt"
	"sync"
)

// Attribute DatabaseConnection *configuration.DatabaseConnection - database connection manager.
// Attribute mutex *sync.Mutex - synchronisation of access to data table (bug in sqlite3).
// See *configuration.DatabaseConnection.
type StatisticalData struct {
	DatabaseConnection 	*configuration.DatabaseConnection
	mutex				*sync.Mutex
}

// Data represents structure of information that is stored for matching incoming frames.
// Attribute ID uint - unique identification of data entry.
// Attribute Time time.Time - automatically generated time of insetting data entry into the relation.
// See time.Time.
// Attribute Bytes uint - number of captured bytes (whole frame).
// Attribute DataTypes *([]*DataType) - list of data types that describe this data entry (many-to-many).
// See DataType.
// Attribute Direction uint - RX (0) or TX (1) direction of flow.
type Data struct {
	ID 					uint			`gorm:"primary_key;AUTO_INCREMENT"`
	Time				time.Time		`gorm:"not null;default:CURRENT_TIMESTAMP"`
	Bytes				uint			`gorm:"not null"`
	DataTypes			*([]*DataType)	`gorm:"many2many:data_to_types"`
	Direction			uint			`gorm:"not null"`
}

// Description of the data type.
// Attribute ID uint - unique identification of data type.
// Attribute Name string - unique name of data type.
// Attribute Forecasting bool - Enabled or disabled forecasting feature.
// NetworkProtocol uint - EthernetType field from Ethernet2 frame (decimal value).
// TransportProtocol uint - Protocol field from IPv4 / IPv6 packet (decimal value).
// Port uint - TCP / UDP destination / source port number.
// Data *([]*Data) - List of data that is in relation with this data type (many-to-many). See Data.
type DataType struct {
	ID 					uint 			`gorm:"primary_key;AUTO_INCREMENT"`
	Name 				string			`gorm:"not null;unique;index:idx_name;size:255"`
	Forecasting 		bool			`gorm:"not null;default:'false'"`
	NetworkProtocol		uint			`gorm:"not null;unique_index:idx_unique_capture"`
	TransportProtocol	uint			`gorm:"not null;unique_index:idx_unique_capture"`
	Port				uint			`gorm:"not null;unique_index:idx_unique_capture"`
	Data				*([]*Data)		`gorm:"many2many:data_to_types"`
}

// This structure represents information that are used for inserting new data into Data relation.
// Attribute Bytes uint - number of captured bytes (whole frame).
// Attribute NetworkProtocol uint - EthernetType field from Ethernet2 frame (decimal value).
// Attribute TransportProtocol uint - Protocol field from IPv4 / IPv6 packet (decimal value).
// Attribute SrcPort uint - TCP / UDP source port number.
// Attribute DstPort uint - TCP / UDP destination port number.
// Attribute Direction uint - RX (0) or TX (1) direction of flow.
type RawData struct {
	Bytes				uint
	NetworkProtocol		uint
	TransportProtocol	uint
	SrcPort				uint
	DstPort				uint
	Time				time.Time
	Direction			uint
}

// Creating of StatisticalData instance.
// Parameter databaseConnection *configuration.DatabaseConnection - database connection manager.
// See *configuration.DatabaseConnection.
func NewStatisticalData(databaseConnection *configuration.DatabaseConnection) *StatisticalData {
	statisticalData := StatisticalData{
		DatabaseConnection: databaseConnection,
		mutex: &sync.Mutex{},
	}
	return &statisticalData
}

// Initialisation of database relations or tables if they haven't already been created.
//
func (StatisticalData *StatisticalData) TablesInit() {
	configuration.Info.Println("Initialisation of the database relations.")
	err := StatisticalData.DatabaseConnection.DB.AutoMigrate(&DataType{}, &Data{}).Error
	if err != nil {
		configuration.Error.Panic("Golang data model cannot be migrated to SQL: ", err)
	}
	configuration.Info.Println("Relations are initialised.")
}

// Writing of new data entries into the Data relation. Data is written only if there is at least one
// submitted data type that matches specified raw data (protocols).
// Parameter rawData *[](*RawData) - list of data that is going to be written into the database.
// See RawData
func (StatisticalData *StatisticalData) WriteNewDataEntries(rawData *[](*RawData)) {
	StatisticalData.mutex.Lock()
	if len(*rawData) != 0 {
		tx := StatisticalData.DatabaseConnection.DB.Begin()
		for _, data := range *rawData {
			// Searching for data types that match input data.
			var dataTypes [](*DataType)
			err01 := tx.Where(
				"network_protocol = ? OR " +
					"(network_protocol = ? AND " +
					"(transport_protocol = ? OR " +
					"(transport_protocol = ? AND " +
					"(port = ? OR port = ? OR port = ?))))",
					0, data.NetworkProtocol, 0, data.TransportProtocol, 0, data.SrcPort, data.DstPort).
				Find(&dataTypes).Error
			if err01 != nil {
				tx.Rollback()
				configuration.Error.Panic("Query into the data_types table failed: ", err01)
			}
			// There is at least one matching data type. Now it is needed to write new data entry.
			if len(dataTypes) != 0 {
				newData := Data{Bytes: data.Bytes, Time: data.Time, Direction: data.Direction}
				err02 := tx.Create(&newData).Error
				if err02 != nil {
					tx.Rollback()
					configuration.Error.Panic("A new data entry cannot be created: ", err02)
				}
				err03 := tx.Model(&newData).
					Association("DataTypes").
					Append(&dataTypes).Error
				if err03 != nil {
					tx.Rollback()
					configuration.Error.Panic("Associations between data and data types " +
						"cannot be established: ", err02)
				}
			}
		}
		tx.Commit()
		StatisticalData.mutex.Unlock()
	}
}

// Adding of new data type.
// Parameter dataType *DataType - information about data type that is going to be saved into the database
// (without id). Data type must be unique by name and group of three information: port, network, and
// transport protocol. See DataType.
// Returning *DataType - Data type with assigned ID.
// Returning error - The data type is not unique.
func (StatisticalData *StatisticalData) WriteNewDataType(dataType *DataType) (*DataType, error) {
	tx := StatisticalData.DatabaseConnection.DB.Begin()
	err01 := checkDataType(dataType)
	if err01 != nil {
		tx.Rollback()
		return nil, err01
	}
	err02 := tx.Create(dataType).Error
	if err02 != nil {
		tx.Rollback()
		if strings.HasPrefix(err02.Error(),"UNIQUE constraint failed") {
			compositeError := configuration.NewCompositeError()
			compositeError.AddError(1, fmt.Sprintf("Cannot insert a new data type into " +
				"the database, data type: %v: %s", *dataType, err02))
			return nil, compositeError.Evaluate()
		} else {
			configuration.Error.Panic("Cannot insert a new data type into the database, data type: ",
				*dataType, ": ", err02)
		}
	}
	tx.Commit()
	return dataType, nil
}

// Reading of information about data type by input ID.
// Parameter id uint - unique id of data type.
// Returning *DataType - read information about data type or nil if the error is not nil. See DataType.
// Returning error - Data type doesn't exist or nil if there is not error.
func (StatisticalData *StatisticalData) GetDataType(id uint) (*DataType, error) {
	if id != 0 {
		tx := StatisticalData.DatabaseConnection.DB.Begin()
		dataType := DataType{ID: id}
		err := tx.Where(&dataType).First(&dataType).Error
		if err != nil {
			tx.Rollback()
			if strings.HasSuffix(err.Error(),"record not found") {
				compositeError := configuration.NewCompositeError()
				compositeError.AddError(1, fmt.Sprintf("The data type with specified id cannot " +
					"be found, data type id: %d: %s", id, err))
				return nil, compositeError.Evaluate()
			} else {
				configuration.Error.Panic("Searching of the data type failed, data type id: ", id, ": ", err)
			}
		}
		tx.Commit()
		return &dataType, nil
	} else {
		compositeError := configuration.NewCompositeError()
		compositeError.AddError(1, fmt.Sprintf("The data type with specified id cannot " +
			"be found, data type id: 0"))
		return nil, compositeError.Evaluate()
	}
}

// Altering of data type settings.
// Parameter id uint - unique id of data type that is going to be modified.
// Parameter dataType *DataType - modified data type (id cannot be changed). See DataType.
// Returning error - the specified data type is not unique or data type with specified id cannot be found.
func (StatisticalData *StatisticalData) ModifyDataType(id uint, dataType *DataType) error {
	tx := StatisticalData.DatabaseConnection.DB.Begin()
	err01 := checkDataType(dataType)
	if err01 != nil {
		tx.Rollback()
		return err01
	}
	// Searching for old data type.
	oldDataType := DataType{ID: id}
	err02 := tx.Where(&oldDataType).First(&oldDataType).Error
	if err02 != nil {
		tx.Rollback()
		compositeError := configuration.NewCompositeError()
		compositeError.AddError(1, fmt.Sprintf("Old data type cannot be identified, data type " +
			"id: %d: %s", id, err02))
		return compositeError.Evaluate()
	}
	// Writing of data type modifications.
	dataType.ID = oldDataType.ID
	err03 := tx.Save(dataType).Error
	if err03 != nil {
		tx.Rollback()
		if strings.HasPrefix(err03.Error(),"UNIQUE constraint failed") {
			compositeError := configuration.NewCompositeError()
			compositeError.AddError(1, fmt.Sprintf("Cannot update the data type, data type " +
				"id: %d: %s", id, err03))
			return compositeError.Evaluate()
		} else {
			configuration.Error.Panic("Cannot update the data type, data type id: ", id, ": ", err03)
		}
	}
	tx.Commit()
	return nil
}

// Removal of the data type; afterwards removal of orphaned data.
// Parameter id uint - id of the data type that is going to be removed.
// Returning *DataType - removed data type. See DataType.
// Returning error - data type with given name cannot be found.
func (StatisticalData *StatisticalData) RemoveDataType(id uint) (*DataType, error) {
	if id != 0 {
		tx := StatisticalData.DatabaseConnection.DB.Begin()
		dataType := DataType{ID: id}
		err01 := tx.Where(&dataType).First(&dataType).Error
		if err01 != nil {
			compositeError := configuration.NewCompositeError()
			compositeError.AddError(1, fmt.Sprintf("The data type with given id doesn't exist: %d: %s",
				id, err01))
			tx.Rollback()
			return nil, compositeError.Evaluate()
		} else {
			// Searching for related data.
			var data [](*Data)
			err02 := tx.Model(&dataType).Association("Data").Find(&data).Error
			if err01 != nil {
				tx.Rollback()
				configuration.Error.Panic("Data associated with the data type cannot be matched, data type id: ",
					id, ": ", err02)
			}
			// Removing of associations.
			err03 := tx.Model(&dataType).Association("Data").Delete(&data).Error
			if err03 != nil {
				tx.Rollback()
				configuration.Error.Panic("An association between data and types cannot be removed, data type id: ",
					id, ": ", err03)
			}
			// Removing of orphaned associated data.
			for _, dataEntry := range data {
				count := tx.Model(&dataEntry).Association("DataTypes").Count()
				if count == 0 {
					err04 := tx.Delete(&dataEntry).Error
					if err04 != nil {
						tx.Rollback()
						configuration.Error.Panic("One of the data entries cannot be removed from database, " +
							"data type id: ", id, ": ", err04)
					}
				}
			}
			// Removing of the data type.
			err05 := tx.Delete(&dataType).Error
			if err05 != nil {
				tx.Rollback()
				configuration.Error.Panic("Cannot delete an existing data type, data type id: ", id, ": ", err05)
			}
		}
		tx.Commit()
		return &dataType, nil
	} else {
		compositeError := configuration.NewCompositeError()
		compositeError.AddError(1, fmt.Sprintf("The data type with given id doesn't exist: %d", id))
		return nil, compositeError.Evaluate()
	}
}

// Listing of all saved data types.
// Returning *[](*DataType) - list of all data types with their description. See DataType.
func (StatisticalData *StatisticalData) ListDataTypes() *[](*DataType) {
	tx := StatisticalData.DatabaseConnection.DB.Begin()
	var dataTypes [](*DataType)
	err := tx.Find(&dataTypes).Error
	if err != nil {
		tx.Rollback()
		configuration.Error.Panic("Written data types cannot be listed: ", err)
	}
	tx.Commit()
	return &dataTypes
}

// Searching for the most recent data entries of specific type.
// Parameter name string - name of the data type.
// Parameter limit time.Time - only data entries older than limit are returned. See time.Time.
// Parameter direction uint - only RX (0) or TX (1) data entries are returned.
// Returning *[](*Data) - data entries (references). See Data.
// Returning error - Non-nil error is returned if the data type with selected name doesn't exist.
func (StatisticalData *StatisticalData) ListLastDataEntries(name string, limit time.Time, direction uint) (
	*[](*Data), error) {
	tx := StatisticalData.DatabaseConnection.DB.Begin()
	var finalData [](*Data)
	dataType := DataType{Name: name}
	tx.Where(&dataType).First(&dataType)
	if dataType.ID == 0 {
		compositeError := configuration.NewCompositeError()
		compositeError.AddError(1, fmt.Sprintf("The data type with given name doesn't exist: %s", name))
		tx.Rollback()
		return nil, compositeError.Evaluate()
	} else {
		err := tx.Model(&dataType).
			Order("time asc").
			Where("time > ? AND direction == ?", limit, direction).
			Association("Data").
			Find(&finalData).Error
		if err != nil {
			tx.Rollback()
			configuration.Error.Panic("Historical data cannot be fetched from the database: ", err)
		}
	}
	tx.Commit()
	return &finalData, nil
}

// Removing of old data entries and associations with data types.
// Parameter limit time.Time - only data entries that are as old or older than limit are removed.
func (StatisticalData *StatisticalData) RemoveOldDataEntries(limit time.Time) {
	tx := StatisticalData.DatabaseConnection.DB.Begin()
	// Searching for old data.
	var oldData [](*Data)
	err01 := tx.Where("time <= ?", limit).Find(&oldData).Error
	if err01 != nil {
		tx.Rollback()
		configuration.Error.Panic("Old data cannot be fetched from database: ", err01)
	}
	// Removing of old associations and data.
	for _, data := range oldData {
		var dataTypes [](*DataType)
		err02 := tx.Model(&data).Association("DataTypes").Find(&dataTypes).Error
		if err02 != nil {
			tx.Rollback()
			configuration.Error.Panic("Old data associations cannot be fetched from database: ", err02)
		}
		err03 := tx.Model(&data).Association("DataTypes").Delete(&dataTypes).Error
		if err03 != nil {
			tx.Rollback()
			configuration.Error.Panic("Old data associations cannot be removed from database: ", err03)
		}
		err04 := tx.Delete(&data).Error
		if err04 != nil {
			tx.Rollback()
			configuration.Error.Panic("Old data cannot be removed from database: ", err04)
		}
	}
	tx.Commit()
}

// Checking of the data type specification (fields format).
// Parameter dataType *DataType - inspected data type. See DataType.
// Returning error - indication of wrong format (one or more fields).
func checkDataType(dataType *DataType) error {
	compositeError := configuration.NewCompositeError()
	if len(dataType.Name) == 0 || len(dataType.Name) > 255 {
		compositeError.AddError(1, fmt.Sprintf("data type name: %s: length of the name must be longer " +
			"than 0 and shorter than 256 characters", dataType.Name))
	}
	if dataType.Port > 65535 {
		compositeError.AddError(1, fmt.Sprintf("data type port: %d: maximum value of the port identification" +
			" is 65535", dataType.Port))
	}
	if dataType.TransportProtocol > 255 {
		compositeError.AddError(1, fmt.Sprintf("data type transport protocol: %d: maximum value of the " +
			"transport protocol identification is 255", dataType.TransportProtocol))
	}
	if dataType.NetworkProtocol > 65535 {
		compositeError.AddError(1, fmt.Sprintf("data type network protocol: %d: maximum value of the " +
			"network protocol identification is 65535", dataType.NetworkProtocol))
	}
	finalError := compositeError.Evaluate()
	return finalError
}