package model

import (
	"io/ioutil"
	"encoding/xml"
	"configuration"
)

// The struct of configuration file.
// Attribute NetworkConfiguration NetworkConfiguration - network-based settings.
// See NetworkConfiguration.
// Attribute CleaningConfiguration - settings that relate with periodical cleaning of old data entries.
type ConfigData struct {
	NetworkConfiguration 	NetworkConfiguration
	CleaningConfiguration 	CleaningConfiguration
	PredictionConfiguration PredictionConfiguration
}

// Network-based settings.
// Attribute AdapterName string - The PCAP path to selected network adapter
// (example: /Device/NPF_{A4C8ED88-6688-448F-8737-4451E903E16C}).
// Attribute MaximumFrameSize uint - Maximum possible size of the captured frame.
// Attribute ReadTimeout int - Reading timeout in milliseconds (reading from the network adapter buffer).
// Attribute DataBuffer uint - Maximum amount of time (milliseconds) during which the caching buffer is
// continuously filling before it is sent to the next processing (writing to the database is the last mile).
type NetworkConfiguration struct {
	AdapterName 		string
	MaximumFrameSize 	uint
	ReadTimeout 		int
	DataBuffer 			uint
}

// Prediction-based settings.
// Attribute CleaningInterval uint - attribute specifies how often should old data entries be removed (seconds).
// Attribute CleaningDepth uint - only data entries that are older than this treshhold are removed (seconds).
type CleaningConfiguration struct {
	CleaningInterval 	uint
	CleaningDepth 		uint
}

type PredictionConfiguration struct {
	SmoothingRange		uint
}

// Parsing of XML configuration file into the ConfigData struct.
// Returns ConfigData - The struct with all configuration settings. See ConfigData.
func ReadConfiguration() ConfigData {
	configuration.OpenXmlConfigurationFile()
	defer configuration.CloseConfigurationFile()
	xmlFileData, _ := ioutil.ReadAll(configuration.XmlFile)
	var configData ConfigData
	err := xml.Unmarshal(xmlFileData, &configData)
	if err != nil {
		configuration.Error.Fatal("Error occurred during unmarshaling of XML: ", err)
	}
	return configData
}