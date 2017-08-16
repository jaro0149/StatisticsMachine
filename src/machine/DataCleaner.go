package machine

import (
	"time"
	"model"
	"configuration"
)

// Functions starts cleaning of the data entries from database.
// Parameter cleaningConfiguration *model.CleaningConfiguration - cleaning depth and interval. See
// model.CleaningConfiguration.
func StartCleaning(cleaningConfiguration *model.CleaningConfiguration) {
	configuration.Info.Println("Starting of the data entries cleaning process.")
	go periodicTask(cleaningConfiguration)
}

// Function executes infinite loop under which old data entries are periodically removed to the configured depth.
// Parameter cleaningConfiguration *model.CleaningConfiguration - cleaning depth and interval. See
// model.CleaningConfiguration.
func periodicTask(cleaningConfiguration *model.CleaningConfiguration) {
	ticker := time.NewTicker(time.Duration(cleaningConfiguration.CleaningInterval) * time.Second)
	for {
		select {
		case <- ticker.C:
			now := time.Now()
			limit := now.Add(- time.Duration(cleaningConfiguration.CleaningDepth) * time.Second)
			model.RemoveOldDataEntries(limit)
		}
	}
	configuration.Info.Println("Data entries cleaning finished.")
}