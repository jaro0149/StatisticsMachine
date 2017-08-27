package machine

import (
	"testing"
	"os"
	"io/ioutil"
	"configuration"
	"model"
	"time"
)

// Tested object of PredictionCreator struct. See PredictionCreator.
//
var predictionCreator PredictionCreator

// Scheduling of setup, unit tests and tear-down functions. See testing.M
// Parameter m *testing.M - unit tests machine.
func TestMain(m *testing.M) {
	setUp()
	retCode := m.Run()
	tearDown()
	os.Exit(retCode)
}

// Unit tests preparation - initialisation of logging unit and PredictionCreator object.
//
func setUp() {
	configuration.LoggingInit(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)
	predictionCreator = PredictionCreator{}
}

// Cleaning after performing unit tests.
//
func tearDown() {

}

// Unit test - testing of data vector smoothing #1.
// Parameter t *testing.T - testing engine.
func TestPredictionCreator_SmoothData1(t *testing.T) {
	t.Log("Initialisation of prediction configuration and data slice ...")
	conf := model.PredictionConfiguration{
		SmoothingRange: 5000,
	}
	var dataSlice [](*model.Data)
	dataEntries := 18
	runningTime := time.Now()
	runningBytes := 10
	for i:=0; i<dataEntries; i++ {
		dataSlice = append(dataSlice, &model.Data{
			Bytes: uint(runningBytes),
			Time: runningTime,
		})
		runningTime = runningTime.Add(time.Duration(1000) * time.Millisecond)
		runningBytes += 2
	}

	t.Log("Execution of data smoothing ...")
	smoothedData := predictionCreator.SmoothData(&conf, &dataSlice)

	t.Log("Verification of smoothed data slice ...")
	validData := []float64{14.0, 24.0, 34.0, 42.0}
	validTimeStart := dataSlice[5].Time
	if len(*smoothedData) != len(validData) {
		t.Fatalf("The length of smoothed data is invalid - expected length: %d, actual length: %d",
			len(validData), len(*smoothedData))
	}
	for i:=0; i<len(validData); i++ {
		if validData[i] != (*smoothedData)[i].DataElement {
			t.Errorf("Expected data element of smoothed vector: %f, actual value: %f",
				validData[i], (*smoothedData)[i].DataElement)
		}
	}
	if !validTimeStart.Equal((*smoothedData)[0].Timestamp) {
		t.Errorf("Expected timestamp of smoothed vector: %s, actual value: %s",
			validTimeStart, (*smoothedData)[0].Timestamp)
	}
}

// Unit test - testing of data vector smoothing #2.
// Parameter t *testing.T - testing engine.
func TestPredictionCreator_SmoothData2(t *testing.T) {
	t.Log("Initialisation of prediction configuration and data slice ...")
	conf := model.PredictionConfiguration{
		SmoothingRange: 1000,
	}
	var dataSlice [](*model.Data)
	dataEntries := 3
	runningTime := time.Now()
	runningBytes := 10
	for i:=0; i<dataEntries; i++ {
		dataSlice = append(dataSlice, &model.Data{
			Bytes: uint(runningBytes),
			Time: runningTime,
		})
		runningTime = runningTime.Add(time.Duration(100) * time.Millisecond)
		runningBytes += 5
	}

	t.Log("Execution of data smoothing ...")
	smoothedData := predictionCreator.SmoothData(&conf, &dataSlice)

	t.Log("Verification of smoothed data slice ...")
	validData := []float64{15.0}
	if len(*smoothedData) != len(validData) {
		t.Fatalf("The length of smoothed data is invalid - expected length: %d, actual length: %d",
			len(validData), len(*smoothedData))
	}
	for i:=0; i<len(validData); i++ {
		if validData[i] != (*smoothedData)[i].DataElement {
			t.Errorf("Expected data element of smoothed vector: %f, actual value: %f",
				validData[i], (*smoothedData)[i].DataElement)
		}
	}
}

// Unit test - testing of data vector smoothing #3.
// Parameter t *testing.T - testing engine.
func TestPredictionCreator_SmoothData3(t *testing.T) {
	t.Log("Initialisation of prediction configuration and data slice ...")
	conf := model.PredictionConfiguration{
		SmoothingRange: 1000,
	}
	var dataSlice [](*model.Data)
	dataEntries := 10
	runningTime := time.Now()
	runningBytes := 10
	for i:=0; i<dataEntries; i++ {
		dataSlice = append(dataSlice, &model.Data{
			Bytes: uint(runningBytes),
			Time: runningTime,
		})
		runningTime = runningTime.Add(time.Duration(400) * time.Millisecond)
		runningBytes += 5
	}

	t.Log("Execution of data smoothing ...")
	smoothedData := predictionCreator.SmoothData(&conf, &dataSlice)

	t.Log("Verification of smoothed data slice ...")
	validData := []float64{15, 27.5, 40.0, 52.5}
	if len(*smoothedData) != len(validData) {
		t.Fatalf("The length of smoothed data is invalid - expected length: %d, actual length: %d",
			len(validData), len(*smoothedData))
	}
	for i:=0; i<len(validData); i++ {
		if validData[i] != (*smoothedData)[i].DataElement {
			t.Errorf("Expected data element of smoothed vector: %f, actual value: %f",
				validData[i], (*smoothedData)[i].DataElement)
		}
	}
}

// Unit test - testing of data vector smoothing #4.
// Parameter t *testing.T - testing engine.
func TestPredictionCreator_SmoothData4(t *testing.T) {
	t.Log("Initialisation of prediction configuration and data slice ...")
	conf := model.PredictionConfiguration{
		SmoothingRange: 1000,
	}
	var dataSlice [](*model.Data)
	dataEntries := 4
	runningTime := time.Now()
	runningBytes := 10
	for i:=0; i<dataEntries; i++ {
		dataSlice = append(dataSlice, &model.Data{
			Bytes: uint(runningBytes),
			Time: runningTime,
		})
		runningTime = runningTime.Add(time.Duration(5000) * time.Millisecond)
		runningBytes += 5
	}

	t.Log("Execution of data smoothing ...")
	smoothedData := predictionCreator.SmoothData(&conf, &dataSlice)
	/*for i:=0; i<len(*smoothedData); i++ {
		if (*smoothedData)[i] != nil {
			fmt.Println((*smoothedData)[i].DataElement)
		} else {
			fmt.Println(nil)
		}
	}*/

	t.Log("Verification of smoothed data slice ...")
	validData := []float64{10.0, 0.0, 0.0, 0.0, 0.0, 15.0, 0.0, 0.0, 0.0, 0.0, 20.0, 0.0, 0.0, 0.0, 0.0, 25.0}
	if len(*smoothedData) != len(validData) {
		t.Fatalf("The length of smoothed data is invalid - expected length: %d, actual length: %d",
			len(validData), len(*smoothedData))
	}
	for i:=0; i<len(validData); i++ {
		if validData[i] != (*smoothedData)[i].DataElement {
			t.Errorf("Expected data element of smoothed vector: %f, actual value: %f",
				validData[i], (*smoothedData)[i].DataElement)
		}
	}
}