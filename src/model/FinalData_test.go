package model

import (
	"testing"
	"time"
	"strings"
)

func TestDataRouterSetData(t *testing.T) {
	t.Log("Final data preparation ...")
	smoothedData := [](*FinalData){
		&FinalData{DataElement: 10.0, Timestamp: time.Now()},
		&FinalData{DataElement: 15.0, Timestamp: time.Now()},
	}
	finalData := [](*FinalData){
		&FinalData{DataElement: 5.0, Timestamp: time.Now()},
		&FinalData{DataElement: 0.5, Timestamp: time.Now()},
	}
	dataType01 := DataType{
		ID: 1,
		Name: "DataType 03",
		Forecasting: true,
		NetworkProtocol: 10,
		TransportProtocol: 5,
		Port: 45,
	}
	dataType02 := DataType{
		ID: 2,
		Name: "DataType 01",
		Forecasting: false,
		NetworkProtocol: 11,
		TransportProtocol: 6,
		Port: 46,
	}
	dataType03 := DataType{
		ID: 1,
		Name: "DataType 03",
		Forecasting: false,
		NetworkProtocol: 11,
		TransportProtocol: 6,
		Port: 46,
	}

	t.Log("Setting of final data packages of two distinct data types ...")
	dataRouter.SetData(&dataType01, nil, nil)
	dataRouter.SetData(&dataType02, nil, nil)

	t.Log("Setting of final data package of one data type that has already been inserted ...")
	dataRouter.SetData(&dataType03, &smoothedData, &finalData)

	t.Log("Verification of written final data ...")
	list := *dataRouter.dataList
	if len(list) != 2 {
		t.Fatalf("Expected number of final data entries: %d, returned number of data entries: %d",
		2, len(list))
	}
	firstListEntry := list[0]
	secondListEntry := list[1]
	if strings.Compare(firstListEntry.Name, dataType02.Name) != 0 {
		t.Errorf("Expected data type name (first entry): %s, actual data type name: %s",
			dataType02.Name, firstListEntry.Name)
	}
	if strings.Compare(secondListEntry.Name, dataType03.Name) != 0 {
		t.Errorf("Expected data type name (second entry): %s, actual data type name: %s",
			dataType03.Name, secondListEntry.Name)
	}
	if secondListEntry.smoothedData == nil {
		t.Errorf("Expected final data smoothed entry: not nil, but got nil entry.")
	}
	if secondListEntry.predictedData == nil {
		t.Errorf("Expected final data predicted entry: not nil, but got nil entry.")
	}
}

func TestDataRouterRemoveData(t *testing.T) {
	t.Log("Final data preparation ...")
	smoothedData := [](*FinalData){
		&FinalData{DataElement: 10.0, Timestamp: time.Now()},
		&FinalData{DataElement: 15.0, Timestamp: time.Now()},
	}
	finalData := [](*FinalData){
		&FinalData{DataElement: 5.0, Timestamp: time.Now()},
		&FinalData{DataElement: 0.5, Timestamp: time.Now()},
	}
	dataType01 := DataType{
		ID: 1,
		Name: "DataType 03",
		Forecasting: true,
		NetworkProtocol: 10,
		TransportProtocol: 5,
		Port: 45,
	}
	dataType02 := DataType{
		ID: 2,
		Name: "DataType 01",
		Forecasting: false,
		NetworkProtocol: 11,
		TransportProtocol: 6,
		Port: 46,
	}

	t.Log("Setting of final data packages of two distinct data types ...")
	dataRouter.SetData(&dataType01, &smoothedData, &finalData)
	dataRouter.SetData(&dataType02, &smoothedData, &finalData)

	t.Log("Removing of the selected data type from final data set ...")
	dataRouter.RemoveDataByType(&dataType02)

	t.Log("Verification of written final data ...")
	list := *dataRouter.dataList
	if len(list) != 1 {
		t.Fatalf("Expected number of final data entries: %d, returned number of data entries: %d",
			1, len(list))
	}
	listEntry := list[0]
	if strings.Compare(listEntry.Name, dataType01.Name) != 0 {
		t.Errorf("Expected data type name (first entry): %s, actual data type name: %s",
			dataType02.Name, listEntry.Name)
	}
}