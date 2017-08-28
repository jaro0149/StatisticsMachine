package model

import (
	"io/ioutil"
	"encoding/xml"
	"configuration"
)

type ConfigurationManager struct {}

// The struct of configuration file.
// Attribute NetworkConfiguration NetworkConfiguration - network-based settings.
// See NetworkConfiguration.
// Attribute CleaningConfiguration - settings that relate with periodical cleaning of old data entries.
// Attribute PredictionConfiguration - setting that relate with smoothing and forecasting model.
// Attribute RestConfiguration - settings that relate with routing.
type ConfigData struct {
	NetworkConfiguration 	NetworkConfiguration
	CleaningConfiguration 	CleaningConfiguration
	PredictionConfiguration PredictionConfiguration
	RestConfiguration		RestConfiguration
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

// Cleaning-based settings.
// Attribute CleaningInterval uint - attribute specifies how often should old data entries be removed (seconds).
// Attribute CleaningDepth uint - only data entries that are older than this treshhold are removed (seconds).
type CleaningConfiguration struct {
	CleaningInterval 	uint
	CleaningDepth 		uint
}

// Prediction-based settings.
// Attribute SmoothingRange uint - time range (milliseconds) that is smoothed to one point in time.
// Attribute SmoothingThreads uint - Initial number of threads that serve data smoothing. This count is subsequently
// decreased if threads cannot be fitted with data slice.
type PredictionConfiguration struct {
	SmoothingRange		uint
	SmoothingThreads	uint
}

// REST configuration.
// Attribute LocalhostPort uint - listening TCP port (HTTP communication).
type RestConfiguration struct {
	LocalhostPort		uint
}

func NewConfigurationManager() *ConfigurationManager {
	return &ConfigurationManager{}
}

// Parsing of XML configuration file into the ConfigData struct.
// Returns ConfigData - The struct with all configuration settings. See ConfigData.
func (ConfigurationManager *ConfigurationManager) ReadConfiguration() ConfigData {
	xmlInstance := configuration.NewConfigFileAccessor()
	xmlInstance.OpenXmlConfigurationFile()
	defer xmlInstance.CloseConfigurationFile()
	xmlFileData, _ := ioutil.ReadAll(configuration.XmlFile)
	var configData ConfigData
	err := xml.Unmarshal(xmlFileData, &configData)
	if err != nil {
		configuration.Error.Fatal("Error occurred during unmarshaling of XML: ", err)
	}
	return configData
}