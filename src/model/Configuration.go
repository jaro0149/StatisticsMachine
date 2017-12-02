package model

import (
	"io/ioutil"
	"encoding/xml"
	"configuration"
)

type ConfigurationManager struct {}

// The struct of configuration file.
// Attribute NetworkConfiguration - network-based settings.
// Attribute CleaningConfiguration - settings that relate with periodical cleaning of old data entries.
// Attribute LoadAnalyserConfiguration - settings that relate with load analyser.
// Attribute RestConfiguration - settings that relate with routing.
// Attribute RServerConfiguration - configuration of connection to R server.
// Attribute WebServerConfiguration	- configuration of web server.
// Attribute GPIOConfiguration - hardware pins.
// Attribute PredictionAnalyserConfiguration - settings that relate with prediction analyser.
type ConfigData struct {
	NetworkConfiguration    		NetworkConfiguration
	CleaningConfiguration   		CleaningConfiguration
	LoadAnalyserConfiguration 		LoadAnalyserConfiguration
	RestConfiguration       		RestConfiguration
	RServerConfiguration    		RServerConfiguration
	WebServerConfiguration  		WebServerConfiguration
	PHYConfiguration       			PHYConfiguration
	PredictionAnalyserConfiguration	PredictionAnalyserConfiguration
}

// Network-based settings.
// Attribute AdapterName string - The PCAP path to selected network adapter
// (example: /Device/NPF_{A4C8ED88-6688-448F-8737-4451E903E16C}).
// Attribute MaximumFrameSize uint - Maximum possible size of the captured frame.
// Attribute ReadTimeout int - Reading timeout in milliseconds (reading from the network adapter buffer).
// Attribute DataBuffer uint - Maximum amount of time (milliseconds) during which the caching buffer is
// continuously filling before it is sent to the next processing (writing to the database is the last mile).
// Attribute RouterMacAddress - Referencing mac address of router port (from this address, the flow direction is
// determined).
// Attribute LinkBandwidth uint64 - Capacity of observed network connection (both TX and RX) [bytes/s].
type NetworkConfiguration struct {
	AdapterName 		string
	MaximumFrameSize 	uint
	ReadTimeout 		int
	DataBuffer 			uint
	RouterMacAddress	string
	LinkBandwidth		uint64
}

// Cleaning-based settings.
// Attribute CleaningInterval uint - attribute specifies how often should old data entries be removed (ms).
// Attribute CleaningDepth uint - only data entries that are older than this treshhold are removed (ms).
type CleaningConfiguration struct {
	CleaningInterval 	uint
	CleaningDepth 		uint
}

// Settings of traffic load analyser.
// Attribute SmoothingRange uint - time range (milliseconds) that is smoothed to one point in time.
// Attribute SmoothingThreads uint - Initial number of threads that serve data smoothing. This count is subsequently
// decreased if threads cannot be fitted with data slice.
// Attribute ComputeInterval uint - interval between new computations of actual load [ms].
// Attribute ComputeDepth uint - how far to go when it comes to average computation [ms].
type LoadAnalyserConfiguration struct {
	SmoothingRange		uint
	SmoothingThreads	uint
	ComputeInterval		uint
	ComputeDepth		uint
}

// Settings of prediction analyser.
// Attribute SmoothingRange uint - time range (milliseconds) that is smoothed to one point in time.
// Attribute SmoothingThreads uint - Initial number of threads that serve data smoothing. This count is subsequently
// decreased if threads cannot be fitted with data slice.
// Attribute ComputeInterval uint - interval between new computations of prediction [ms].
// Attribute ComputeDepth uint - how far to go when it comes to ARIMA computation [ms].
// Attribute PredictionHorizon uint - ARIMA prediction horizon [ms].
// Attribute Designator	float64 - it describes criterion for changing prediction state - fraction of bandwidth that
// must exceeded from actual load (positivw or negative fraction domain).
type PredictionAnalyserConfiguration struct {
	SmoothingRange		uint
	SmoothingThreads	uint
	ComputeInterval		uint
	ComputeDepth		uint
	PredictionHorizon	uint
	Designator			float64
}

// REST configuration.
// Attribute LocalhostPort uint - listening TCP port (HTTP communication).
// Attribute PathGetDataTypes string - Site: listing of all data types (GET).
// Attribute PathGetDataType string - Site: fetching of information about one data type (GET).
// Attribute PathRemoveDataType string - Site: removing of the specific data type (DELETE).
// Attribute PathWriteNewDataType string - Site: creating of the new data type (POST).
// Attribute PathModifyDataType string - Site: modifying of existing data type (POST).
type RestConfiguration struct {
	LocalhostPort			uint
	PathGetDataTypes		string
	PathGetDataType			string
	PathRemoveDataType		string
	PathWriteNewDataType	string
	PathModifyDataType		string
}

// Web server configuration (Angular 4 scope).
// Attribute LocalhostPort uint - listening TCP port (HTTP communication).
// Attribute RootPath - path to deployed web server page (root directory).
type WebServerConfiguration struct {
	LocalhostPort			uint
	RootPath				string
}

// R server configuration (statistical tool).
// Attribute RemotePort uint - listening TCP port (HTTP communication).
// Attribute RemoteIpAddress string - IP address on which server resides.
// Attribute SessionsCapacity uint - how many sessions can be active concurrently.
type RServerConfiguration struct {
	RemoteIpAddress		string
	RemotePort			uint
	SessionsCapacity	uint
}

// Configuration of GPIO pins.
// Attribute PhyLeftButton uint - physical pin to which the left button is connected.
// Attribute PhyRightButton	uint - physical pin to which the right button is connected.
// Attribute BCM_RS	uint - reset LCD pin.
// Attribute BCM_EN uint - enable LCD pin.
// Attributes BCM_DB4 - BCM_DB7 - data LCD pins.
// Attribute BCM_Backlight uint - software-based back-light control on LCD - unused.
// Attribute BCM_LED_Strip uint - LED strip data pin.
// Attribute LEDsCount uint - number of LEDs assembled in strip.
// Attribute LEDsBrightness uint - LEDs brightness (interval <0, 255>).
type PHYConfiguration struct {
	PhyLeftButton		uint
	PhyRightButton		uint
	BCM_RS				uint
	BCM_EN				uint
	BCM_DB4				uint
	BCM_DB5				uint
	BCM_DB6				uint
	BCM_DB7				uint
	BCM_Backlight		uint
	BCM_LED_Strip		uint
	LEDsCount			uint
	LEDsBrightness		uint
}

// Creating instance of configuration manager.
// Returning *ConfigurationManager - ConfigurationManager object.
func NewConfigurationManager() *ConfigurationManager {
	return &ConfigurationManager{}
}

// Parsing of XML configuration file into the ConfigData struct.
// Returns ConfigData - The struct with all configuration settings. See ConfigData.
func (ConfigurationManager *ConfigurationManager) ReadConfiguration() ConfigData {
	xmlInstance := configuration.NewConfigFileAccessor()
	xmlInstance.OpenXmlConfigurationFile()
	defer xmlInstance.CloseConfigurationFile()
	xmlFileData, _ := ioutil.ReadAll(xmlInstance.XmlFile)
	var configData ConfigData
	err := xml.Unmarshal(xmlFileData, &configData)
	if err != nil {
		configuration.Error.Panicf("Error occurred during unmarshaling of XML %v: ", err)
	}
	return configData
}