package configuration

import "os"

// The path to the configuration file (the same directory as application).
const XML_PATH = "configuration.xml?parseTime=true"

// Reference to the configuration file. See os.File.
var XmlFile *os.File

// Opening of the configuration file (XML settings) so XmlFile is initialised.
func OpenXmlConfigurationFile() {
	Info.Println("Opening of the configuration file.")
	xmlFileDemo, err := os.Open(XML_PATH)
	if err != nil {
		Error.Fatal("Error opening file: ", err)
	}
	XmlFile = xmlFileDemo
	Info.Println("Configuration file is opened.")
}

// Closing of the configuration file.
func CloseConfigurationFile() {
	Info.Println("Closing of the configuration file.")
	err := XmlFile.Close()
	if err != nil {
		Error.Fatal("Error closing file: ", err)
	}
	XmlFile = nil
	Info.Println("Configuration file is closed.")
}