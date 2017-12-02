package main

import (
	"model"
	"os"
	"io/ioutil"
	"configuration"
	"controller"
	"machine"
)

func main() {
	// logging initialisation
	configuration.LoggingInit(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)

	// configuration file
	configManager := model.NewConfigurationManager()
	configData := configManager.ReadConfiguration()

	// database
	databaseConnection := configuration.NewDatabaseConnection()
	databaseConnection.ConnectDatabase()
	defer databaseConnection.CloseDatabase()

	// statistical machine
	statisticalMachine := model.NewStatisticalData(databaseConnection)

	// data collector
	framesParser := machine.NewFramesParser(&configData.NetworkConfiguration, statisticalMachine)
	framesParser.StartCapturing()

	// data cleaner
	dataCleaner := machine.NewDataCleaner(&configData.CleaningConfiguration, statisticalMachine)
	dataCleaner.StartCleaning()

	// device manager
	deviceManager := machine.NewDeviceManager(&configData.PHYConfiguration,
		configData.LoadAnalyserConfiguration.SmoothingRange, configData.PredictionAnalyserConfiguration.Designator,
			configData.NetworkConfiguration.LinkBandwidth)
	deviceManager.StartDeviceManager()
	defer deviceManager.CloseDeviceManager()

	// smoothing creators
	smoothingCreator1 := machine.NewSmoothingCreator(configData.LoadAnalyserConfiguration.SmoothingRange,
		configData.LoadAnalyserConfiguration.SmoothingThreads)
	smoothingCreator2 := machine.NewSmoothingCreator(configData.PredictionAnalyserConfiguration.SmoothingRange,
		configData.PredictionAnalyserConfiguration.SmoothingThreads)

	// real-time load analyser
	realTimeLoader := machine.NewLoadAnalyser(&configData.LoadAnalyserConfiguration, deviceManager,
		statisticalMachine, smoothingCreator1)
	realTimeLoader.StartMachine()

	// R server connection
	rServer := configuration.NewRServer(configData.RServerConfiguration.RemotePort,
		configData.RServerConfiguration.RemoteIpAddress, configData.RServerConfiguration.SessionsCapacity)
	//rServer.StartRServer()
	rServer.ConnectToServer()
	defer rServer.CloseAllSessions()

	// predictive load analyser
	predictionAnalyser := machine.NewPredictionAnalyser(&configData.PredictionAnalyserConfiguration, deviceManager,
		statisticalMachine, smoothingCreator2, rServer, configData.NetworkConfiguration.LinkBandwidth)
	predictionAnalyser.StartMachine()

	// rest server
	restServer := controller.NewRestController(&configData.RestConfiguration, statisticalMachine, deviceManager)
	restServer.StartRestController()

	// web server
	webServer := controller.NewWebServer(&configData.WebServerConfiguration)
	webServer.StartWebServer()

	// wait forever
	select{}
}