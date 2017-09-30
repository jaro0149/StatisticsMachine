package machine

import (
	"time"
	"model"
	"configuration"
)

// Attribute cleaningConfiguration *model.CleaningConfiguration - cleaning depth and interval. See
// model.CleaningConfiguration.
// Attribute statisticalData *model.StatisticalData - instance that control access to SQL database.
// See model.StatisticalData.
type DataCleaner struct {
	cleaningConfiguration 	*model.CleaningConfiguration
	statisticalData 		*model.StatisticalData
}

// Creating instance of the DataCleaner.
// Parameter cleaningConfiguration *model.CleaningConfiguration - cleaning depth and interval. See
// model.CleaningConfiguration.
// Parameter statisticalData *model.StatisticalData - instance that control access to SQL database.
// See model.StatisticalData.
// Returning *DataCleaner - DataCleaner object.
func NewDataCleaner(cleaningConf *model.CleaningConfiguration, statisticalData *model.StatisticalData) *DataCleaner {
	dataCleaner := DataCleaner{
		cleaningConfiguration: cleaningConf,
		statisticalData: statisticalData,
	}
	return &dataCleaner
}

// Functions starts cleaning of the data entries from database.
func (DataCleaner *DataCleaner) StartCleaning() {
	configuration.Info.Println("Starting of the data entries cleaning process.")
	go periodicTask(DataCleaner.cleaningConfiguration, DataCleaner.statisticalData)
}

// Function executes infinite loop under which old data entries are periodically removed to the configured depth.
// Parameter cleaningConfiguration *model.CleaningConfiguration - cleaning depth and interval. See
// model.CleaningConfiguration.
func periodicTask(cleaningConfiguration *model.CleaningConfiguration, statisticalData *model.StatisticalData) {
	ticker := time.NewTicker(time.Duration(cleaningConfiguration.CleaningInterval) * time.Millisecond)
	for {
		select {
		case <- ticker.C:
			now := time.Now()
			limit := now.Add(- time.Duration(cleaningConfiguration.CleaningDepth) * time.Millisecond)
			statisticalData.RemoveOldDataEntries(limit)
		}
	}
	configuration.Info.Println("Data entries cleaning finished.")
}