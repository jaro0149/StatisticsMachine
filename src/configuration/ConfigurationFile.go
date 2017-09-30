package configuration

import "os"

// Attribute XmlFile *os.File - Reference to the configuration file. See os.File.
type ConfigFileAccessor struct {
	XmlFile *os.File
}

// The path to the configuration file (the same directory as application).
const XML_PATH = "configuration.xml"

// Creating of new ConfigFileAccessor object.
// Returning - instance that controls access to XML configuration file.
func NewConfigFileAccessor() *ConfigFileAccessor {
	return &ConfigFileAccessor{}
}

// Opening of the configuration file (XML settings) so XmlFile is initialised.
// Returning *os.File - opened XML configuration file. See os.File.
func (ConfigFileAccessor *ConfigFileAccessor) OpenXmlConfigurationFile() *os.File {
	Info.Println("Opening of the configuration file.")
	xmlFileDemo, err := os.Open(XML_PATH)
	if err != nil {
		Error.Panic("Error opening file: ", err)
	}
	ConfigFileAccessor.XmlFile = xmlFileDemo
	Info.Println("Configuration file is opened.")
	return xmlFileDemo
}

// Closing of the configuration file.
func (ConfigFileAccessor *ConfigFileAccessor) CloseConfigurationFile() {
	Info.Println("Closing of the configuration file.")
	err := ConfigFileAccessor.XmlFile.Close()
	if err != nil {
		Error.Panic("Error closing file: ", err)
	}
	ConfigFileAccessor.XmlFile = nil
	Info.Println("Configuration file is closed.")
}