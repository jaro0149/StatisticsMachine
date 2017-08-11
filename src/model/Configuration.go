package model

import (
	"io/ioutil"
	"encoding/xml"
	"configuration"
)

// The struct of configuration file.
// Attribute NetworkConfiguration NetworkConfiguration - Network-based configuration.
// See NetworkConfiguration.
type ConfigData struct {
	NetworkConfiguration *NetworkConfiguration
}

// Network-based settings.
// Attribute SourceMacAddress string - The source MAC address determines primary network interface.
type NetworkConfiguration struct {
	SourceMacAddress string
}

// Parsing of XML configuration file into the ConfigData struct.
// Returns ConfigData - The struct with all configuration settings. See ConfigData.
func UnmarshalXmlFile() ConfigData {
	configuration.OpenXmlConfigurationFile()
	defer configuration.CloseConfigurationFile()
	xmlFileData, _ := ioutil.ReadAll(configuration.XmlFile)
	var configData ConfigData
	err := xml.Unmarshal(xmlFileData, &configData)
	if err != nil {
		configuration.Error.Fatal("Error occured during unmarshaling of XML: ", err)
	}
	return configData
}