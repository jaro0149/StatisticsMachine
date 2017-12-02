package machine

import (
	"model"
	"time"
	"configuration"
	"sync"
)

// Sleeping delay between checks whether a new periodical "tick" had appeared on channel.
const THREAD_SLEEPING_DELAY = 50

// Attribute configuration *model.LoadAnalyserConfiguration - computation interval and depth. See
// model.LoadAnalyserConfiguration.
// Attribute deviceManager *DeviceManager - device manager that is notified when a new real-time load is computed.
// See DeviceManager.
// Attribute statisticalData *model.StatisticalData - source of captured statistical data. See model.StatisticalData.
// Attribute smoothingCreator *SmoothingCreator - tools that are used for performing of smoothing over defined range.
// See SmoothingCreator.
type LoadAnalyser struct {
	configuration		*model.LoadAnalyserConfiguration
	deviceManager		*DeviceManager
	statisticalData 	*model.StatisticalData
	smoothingCreator	*SmoothingCreator
}

// Creating of the instance of LoadAnalyser structure.
// Parameter configuration *model.LoadAnalyserConfiguration - computation interval and depth.
// Parameter deviceManager *DeviceManager - device manager that is notified when a new real-time load is computed.
// See DeviceManager.
// Parameter statisticalData *model.StatisticalData - source of captured statistical data. See model.StatisticalData.
// Parameter smoothingCreator *SmoothingCreator - tools that are used for performing of smoothing over defined range.
// See SmoothingCreator.
// Returning *LoadAnalyser - reference to created object.
func NewLoadAnalyser(configuration *model.LoadAnalyserConfiguration, deviceManager *DeviceManager,
	statisticalData *model.StatisticalData, smoothingCreator *SmoothingCreator) *LoadAnalyser {
	realTimeLoader := LoadAnalyser{
		statisticalData: statisticalData,
		smoothingCreator: smoothingCreator,
		deviceManager: deviceManager,
		configuration: configuration,
	}
	return &realTimeLoader
}

// Starting of periodical computation of load over all configured data types that are stored in database (both RX and
// TX direction).
func (RealTimeLoader *LoadAnalyser) StartMachine() {
	configuration.Info.Println("Starting of the real-time load analyser.")
	depth := RealTimeLoader.configuration.ComputeDepth
	tickChannel := time.Tick(time.Millisecond * time.Duration(RealTimeLoader.configuration.ComputeInterval))
	go func() {
		for {
			select {
			case <-tickChannel:
				actualTime := time.Now()
				shiftedTime := actualTime.Add(-time.Duration(depth) * time.Millisecond)
				RealTimeLoader.computeAverageLoad(&shiftedTime)
			}
			time.Sleep(time.Millisecond * THREAD_SLEEPING_DELAY)
		}
	}()
}

// Computation of mean load over last time range. The result is pushed to DeviceManager.
// Parameter limit time.Time - time that specidied lower bound of computation interval over which an average is
// performed. See time.Time.
func (RealTimeLoader *LoadAnalyser) computeAverageLoad(limit *time.Time) {
	RealTimeLoader.statisticalData.UltimateLock()
	defer RealTimeLoader.statisticalData.UltimateUnlock()
	dataTypes := RealTimeLoader.statisticalData.ListDataTypes()
	waitGroup := sync.WaitGroup{}
	for _, dataType := range *dataTypes {
		waitGroup.Add(1)
		go RealTimeLoader.workingAverager(dataType, limit, &waitGroup)
		waitGroup.Wait()
	}
}

// Computation of mean load over last time range - machine that processes one data type.
// Parameter limit time.Time - time that specidied lower bound of computation interval over which an average is
// performed. See time.Time.
// Parameter dataType *model.DataType - Analysed data type for which both RX and TX traffic is processed.
// Parameter waitGroup *sync.WaitGroup - Design pattern of synchronised computation.
func (RealTimeLoader *LoadAnalyser) workingAverager(dataType *model.DataType, limit *time.Time,
	waitGroup *sync.WaitGroup) {
	dataTypeName := dataType.Name
	// list	data
	rxData, err01 := RealTimeLoader.statisticalData.ListLastDataEntries(dataTypeName, *limit, uint(0))
	txData, err02 := RealTimeLoader.statisticalData.ListLastDataEntries(dataTypeName, *limit, uint(1))
	if err01 == nil && err02 == nil {
		// smooth data
		smoothedRxData := RealTimeLoader.smoothingCreator.SmoothData(rxData)
		smoothedTxData := RealTimeLoader.smoothingCreator.SmoothData(txData)
		// compute averages
		rxAverage := averageLoad(smoothedRxData)
		txAverage := averageLoad(smoothedTxData)
		// building of output structures
		loadIdRx := DisplayTemplate{
			dataTypeId: dataType.ID,
			dataTypeName: dataType.Name,
			direction: 0,
			prediction: false,
		}
		loadIdTx := DisplayTemplate{
			dataTypeId: dataType.ID,
			dataTypeName: dataType.Name,
			direction: 1,
			prediction: false,
		}
		// notify device manager
		RealTimeLoader.deviceManager.UpdateDisplayByLoad(&loadIdRx, rxAverage)
		RealTimeLoader.deviceManager.UpdateDisplayByLoad(&loadIdTx, txAverage)
	} else if err01 != nil {
		configuration.Error.Panicf("An error occurred during fetching of last " +
			"statistical entries (RX): %v", err01)
	} else {
		configuration.Error.Panicf("An error occurred during fetching of last " +
			"statistical entries (TX): %v", err02)
	}
	waitGroup.Done()
}

// Average computation from captured statistics.
// Parameter data *[](*model.FinalData) - input slice with data from which average is computed.
// Returning float64 - computed average.
func averageLoad(data *[](*model.FinalData)) float64 {
	dataRef := *data
	if len(dataRef) != 0 {
		var sum uint64 = 0
		for i := range dataRef {
			sum += uint64(dataRef[i].DataElement)
		}
		average := float64(sum) / float64(len(dataRef))
		return average
	} else {
		return float64(0)
	}
}