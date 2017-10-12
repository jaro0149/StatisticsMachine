package main

import (
	"io/ioutil"
	"os"
	"configuration"
	"model"
	"machine"
)

func main() {
	configuration.LoggingInit(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)
	databaseConnection := configuration.NewDatabaseConnection()
	databaseConnection.ConnectDatabase()
	defer databaseConnection.CloseDatabase()

	networkConfiguration := model.NetworkConfiguration{
		RouterMacAddress: "20:89:84:41:4e:d8",
		DataBuffer: 1000,
		ReadTimeout: 10,
		MaximumFrameSize: 1600,
		AdapterName: "/Device/NPF_{A4C8ED88-6688-448F-8737-4451E903E16C}",
	}
	statisticsManager := model.NewStatisticalData(databaseConnection)
	framesParser := machine.NewFramesParser(&networkConfiguration, statisticsManager)
	framesParser.StartCapturing()
	select{}
}