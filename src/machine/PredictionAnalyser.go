package machine

import (
	"model"
	"time"
	"configuration"
	"math"
	"sync"
	"strconv"
	"bytes"
	"reflect"
	"fmt"
)

// R command for fitting of data to ARIMA model.
const AUTO_ARIMA_COMMAND = "model = auto.arima(tsData, seasonal=FALSE, stepwise=TRUE)"
// R command for parsing of mean from predicted structure.
const PARSE_MEAN_COMMAND = "as.numeric(data$mean)"
// R command for forecasting (first part).
const FORECAST_COMMAND_START = "data = forecast(model, h="
// R command for forecasting (second part).
const FORECAST_COMMAND_END = ")"
// R command for building of time series structure (first part).
const CREATE_TS_START = "tsData = ts("
// R command for building of time series structure (second part).
const CREATE_TS_END = ")"

// Attribute configuration *model.PredictionAnalyserConfiguration - computation interval, depth, horizon. See
// model.LoadAnalyserConfiguration.
// Attribute deviceManager *DeviceManager - device manager that is notified when a new real-time load is computed.
// See DeviceManager.
// Attribute statisticalData *model.StatisticalData - source of captured statistical data. See model.StatisticalData.
// Attribute smoothingCreator *SmoothingCreator - tools that are used for performing of smoothing over defined range.
// See SmoothingCreator.
// Attribute rServer *configuration.RServer - connection to R statistical server. See configuration.RServer.
// Attribute linkBandwidth uint64 - observed link bandwidth (maximum load) [bytes/s].
type PredictionAnalyser struct {
	configuration		*model.PredictionAnalyserConfiguration
	deviceManager		*DeviceManager
	statisticalData 	*model.StatisticalData
	smoothingCreator	*SmoothingCreator
	rServer				*configuration.RServer
	linkBandwidth		uint64
}

// Creating of the instance of PredictionAnalyser structure.
// Parameter configuration *model.PredictionAnalyserConfiguration - computation interval, depth, horizon.
// Parameter deviceManager *DeviceManager - device manager that is notified when a new real-time load is computed.
// See DeviceManager.
// Parameter statisticalData *model.StatisticalData - source of captured statistical data. See model.StatisticalData.
// Parameter smoothingCreator *SmoothingCreator - tools that are used for performing of smoothing over defined range.
// See SmoothingCreator.
// Parameter rServer *configuration.RServer - connection to R statistical server. See configuration.RServer.
// Parameter linkBandwidth uint64 - observed link bandwidth (maximum load) [bytes/s].
// Returning *LoadAnalyser - reference to created object.
func NewPredictionAnalyser(configuration *model.PredictionAnalyserConfiguration, deviceManager *DeviceManager,
		statisticalData *model.StatisticalData, smoothingCreator *SmoothingCreator,
		rServer *configuration.RServer, linkBandwidth uint64) *PredictionAnalyser {
	predictionLoader := PredictionAnalyser{
		statisticalData: statisticalData,
		smoothingCreator: smoothingCreator,
		deviceManager: deviceManager,
		configuration: configuration,
		rServer: rServer,
		linkBandwidth: linkBandwidth,
	}
	return &predictionLoader
}

// Starting of periodical computation of ARIMA over all data types with enabled prediction that are stored in database
// (both RX and TX direction).
func (PredictionAnalyser *PredictionAnalyser) StartMachine() {
	configuration.Info.Println("Starting of the predictive load analyser.")
	depth := PredictionAnalyser.configuration.ComputeDepth
	horizonPoints := 	uint(math.Ceil(float64(PredictionAnalyser.configuration.PredictionHorizon) /
						float64(PredictionAnalyser.configuration.SmoothingRange)))
	tickChannel := time.Tick(time.Millisecond * time.Duration(PredictionAnalyser.configuration.ComputeInterval))
	go func() {
		for {
			select {
			case <-tickChannel:
				timeLimit := time.Now().Add(-time.Duration(depth) * time.Millisecond)
				PredictionAnalyser.computePrediction(&timeLimit, horizonPoints)
			}
			time.Sleep(time.Millisecond * THREAD_SLEEPING_DELAY)
		}
	}()
}

// Computation of prediction - procedure that is applied for each data type with enabled prediction.
// Parameter limit *time.Time - forecasted values are built from statistical entries that are older than limit.
// Parameter horizonPoints uint - how many points to go at future within ARIMA model.
func (PredictionAnalyser *PredictionAnalyser) computePrediction(limit *time.Time, horizonPoints uint) {
	PredictionAnalyser.statisticalData.UltimateLock()
	defer PredictionAnalyser.statisticalData.UltimateUnlock()
	dataTypes := PredictionAnalyser.statisticalData.ListDataTypes()
	waitGroup := sync.WaitGroup{}
	for _, dataType := range *dataTypes {
		waitGroup.Add(1)
		go PredictionAnalyser.workingMethod(dataType, limit, horizonPoints, &waitGroup)
	}
	waitGroup.Wait()
}

// Computation of prediction - procedure that is applied for selected data type.
// Parameter limit *time.Time - forecasted values are built from statistical entries that are older than limit.
// Parameter horizonPoints uint - how many points to go at future within ARIMA model.
// Parameter waitGroup *sync.WaitGroup - Design pattern of synchronised computation.
func (PredictionAnalyser *PredictionAnalyser) workingMethod(dataType *model.DataType, limit *time.Time,
	horizonPoints uint, waitGroup *sync.WaitGroup) {
	if dataType.Forecasting {
		// list data
		rxData, err01 := PredictionAnalyser.statisticalData.ListLastDataEntries(dataType.Name, *limit, 0)
		txData, err02 := PredictionAnalyser.statisticalData.ListLastDataEntries(dataType.Name, *limit, 1)
		if err01 == nil && err02 == nil {
			// smooth data
			smoothedRxData := PredictionAnalyser.smoothingCreator.SmoothData(rxData)
			smoothedTxData := PredictionAnalyser.smoothingCreator.SmoothData(txData)
			parallelData := [](*[]uint64){
				transformFinalDataToUintArray(smoothedRxData),
				transformFinalDataToUintArray(smoothedTxData),
			}
			// compute predictions
			predictions := parallelArimaComputations(PredictionAnalyser.rServer, &parallelData, horizonPoints)
			// standardise vectors
			rxStandardized := standardizeVector(PredictionAnalyser.linkBandwidth, (*predictions)[0])
			txStandardized := standardizeVector(PredictionAnalyser.linkBandwidth, (*predictions)[1])
			// compute averages of predictions
			rxAverage := averagePrediction(rxStandardized)
			txAverage := averagePrediction(txStandardized)
			// building of output structures
			loadIdRx := DisplayTemplate{
				dataTypeId:   dataType.ID,
				dataTypeName: dataType.Name,
				direction:    0,
				prediction:   true,
			}
			loadIdTx := DisplayTemplate{
				dataTypeId:   dataType.ID,
				dataTypeName: dataType.Name,
				direction:    1,
				prediction:   true,
			}
			// notify device manager
			PredictionAnalyser.deviceManager.UpdateDisplayByPrediction(&loadIdRx, rxAverage)
			PredictionAnalyser.deviceManager.UpdateDisplayByPrediction(&loadIdTx, txAverage)
		} else if err01 != nil {
			configuration.Error.Panicf("An error occurred during fetching of last "+
				"statistical entries (RX): %v", err01)
		} else {
			configuration.Error.Panicf("An error occurred during fetching of last "+
				"statistical entries (TX): %v", err02)
		}
	}
	waitGroup.Done()
}

// Formatting of input vector - negative values are set to 0 while values bigger than bandwidth are set to bandwidth.
// Parameter bandwidth uint64 - link bandwidth - maximum link load [bytes/sec].
// Parameter prediction *[]uint64 - input (predicted) values that must be formatted.
// Returning *[]uint64 - standardized vector.
func standardizeVector(bandwidth uint64, prediction *[]uint64) *[]uint64 {
	var outputVector []uint64
	for _, element := range *prediction {
		if element < 0 {
			outputVector = append(outputVector, 0)
		} else if element > bandwidth {
			outputVector = append(outputVector, bandwidth)
		} else {
			outputVector = append(outputVector, element)
		}
	}
	return &outputVector
}

// Transformation of final data to uint array.
// Parameter inputData *[](*model.FinalData) - input FinalData slice. See model.FinalData.
// Returning *[]uint64 - converted vector.
func transformFinalDataToUintArray(inputData *[](*model.FinalData)) *[]uint64 {
	var outputArray []uint64
	for _, finalData := range *inputData {
		outputArray = append(outputArray, (*finalData).DataElement)
	}
	return &outputArray
}

// Starting of parallel jobs for evaluating of prediction using R ARIMA library.
// Parameter rServer *configuration.RServer - R instance for creation of new sessions. See configuration.RServer.
// Parameter inputVectors *[](*[]uint64) - input vectors - for each vector a new prediction job is allocated.
// Parameter horizon uint - prediction horizon [number of entries].
func parallelArimaComputations(rServer *configuration.RServer, inputVectors *[](*[]uint64),
	horizon uint) *[](*[]uint64) {
	waitGroup := sync.WaitGroup{}
	mutex := sync.Mutex{}
	results := make([](*[]uint64), len(*inputVectors))
	for i, inputVector := range *inputVectors {
		waitGroup.Add(1)
		evaluateArima(rServer, &waitGroup, &mutex, i, inputVector, horizon, &results)
	}
	waitGroup.Wait()
	return &results
}

// ARIMA evaluation (R commands).
// Parameter rServer *configuration.RServer - R instance for creation of new sessions. See configuration.RServer.
// Parameter waitGroup *sync.WaitGroup - Signalisation of done job. See sync.WaitingGroup.
// Parameter mutex *sync.Mutex - Synchronised access to vector with results. See sync.Mutex.
// Parameter i int - allocated index that points in the resulting vector.
// Parameter horizon uint - prediction horizon [number of entries].
// Parameter results *[](*[]uint64) - shared slice with results - vectors with predicted values.
func evaluateArima(rServer *configuration.RServer, waitGroup *sync.WaitGroup, mutex *sync.Mutex, i int,
	inputVector *([]uint64), horizon uint, results *[](*[]uint64)) {
	defer waitGroup.Done()
	if len(*inputVector) != 0 {
		session := *rServer.GetSession()
		defer (*rServer).ReleaseSession(&session)
		_, err01 := session.Eval(CREATE_TS_START + uintSliceToRVector(inputVector) + CREATE_TS_END)
		if err01 != nil {
			configuration.Error.Panicf("An error occurred during execution of R TS instruction: %v", err01)
		}
		_, err02 := session.Eval(AUTO_ARIMA_COMMAND)
		if err02 != nil {
			configuration.Error.Panicf("An error occurred during execution of R AUTO.ARIMA instruction: %v",
				err02)
		}
		_, err03 := session.Eval(FORECAST_COMMAND_START + strconv.Itoa(int(horizon)) + FORECAST_COMMAND_END)
		if err03 != nil {
			configuration.Error.Panicf("An error occurred during execution of R FORECAST instruction: %v",
				err03)
		}
		mean, err04 := session.Eval(PARSE_MEAN_COMMAND)
		if err04 != nil {
			configuration.Error.Panicf("An error occurred during execution of R PARSE MEAN instruction: %v",
				err04)
		}
		floatSlice := reflectInterfaceToUintSlice(&mean)
		mutex.Lock()
		defer mutex.Unlock()
		(*results)[i] = floatSlice
	} else {
		mutex.Lock()
		defer mutex.Unlock()
		zeroVector := make([]uint64, horizon)
		(*results)[i] = &zeroVector
	}
}

// Conversion of slice to R command (definition of vector).
// Parameter inputVector *([]uint64) - input slice.
// Returning string - output R command.
func uintSliceToRVector(inputVector *([]uint64)) string {
	parsedInputVector := *inputVector
	var buffer bytes.Buffer
	buffer.WriteString("c(")
	for i, num := range parsedInputVector {
		str := fmt.Sprint(num)
		buffer.WriteString(str)
		if i != len(parsedInputVector) - 1 {
			buffer.WriteString(",")
		}
	}
	buffer.WriteString(")")
	return buffer.String()
}

// Parsing of R returning values to uint slice.
// Parameter interfaceValues *(interface{}) - input values with unknown type.
// Returning *([]uint64) - reflected uint slice.
func reflectInterfaceToUintSlice(interfaceValues *(interface{})) *([]uint64) {
	iterableValues := reflect.ValueOf(*interfaceValues)
	var finalSlice []uint64
	for i := 0; i < iterableValues.Len(); i++ {
		finalSlice = append(finalSlice, uint64(iterableValues.Index(i).Float()))
	}
	return &finalSlice
}

// Computation of average from uint64 slice.
// Parameter data *[]uint64 - input vector with data entries.
// Returning float64 - mean value (average).
func averagePrediction(data *[]uint64) float64 {
	dataRef := *data
	if len(dataRef) != 0 {
		var sum uint64 = 0
		for i := range dataRef {
			sum += dataRef[i]
		}
		average := float64(sum) / float64(len(dataRef))
		return average
	} else {
		return float64(0)
	}
}