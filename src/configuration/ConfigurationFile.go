package configuration

import "os"

type ConfigFileAccessor struct {}

// The path to the configuration file (the same directory as application).
const XML_PATH = "configuration.xml"

// Reference to the configuration file. See os.File.
var XmlFile *os.File

// Creating of new ConfigFileAccessor object.
// Returning - instance that controls access to XML configuration file.
func NewConfigFileAccessor() *ConfigFileAccessor {
	return &ConfigFileAccessor{}
}

// Opening of the configuration file (XML settings) so XmlFile is initialised.
func (ConfigFileAccessor *ConfigFileAccessor) OpenXmlConfigurationFile() {
	Info.Println("Opening of the configuration file.")
	xmlFileDemo, err := os.Open(XML_PATH)
	if err != nil {
		Error.Fatal("Error opening file: ", err)
	}
	XmlFile = xmlFileDemo
	Info.Println("Configuration file is opened.")
}

// Closing of the configuration file.
func (ConfigFileAccessor *ConfigFileAccessor) CloseConfigurationFile() {
	Info.Println("Closing of the configuration file.")
	err := XmlFile.Close()
	if err != nil {
		Error.Fatal("Error closing file: ", err)
	}
	XmlFile = nil
	Info.Println("Configuration file is closed.")
}