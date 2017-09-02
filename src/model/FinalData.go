package model

import (
	"time"
	"sync"
	"strings"
	"sort"
)

// Attribute dataList *[](*OutputData) - list of output data entries (smoothed and predicted data). See OutputData.
// Attribute semaphore *sync.Mutex - synchronisation semaphore.
type DataRouter struct {
	dataList		*[](*OutputData)
	semaphore		*sync.Mutex
}

// Smoothed or predicted data.
// Attribute DataElement float64 - number of bytes.
// Attribute Timestamp time.Time - data element is set on this time.
type FinalData struct {
	DataElement		float64
	Timestamp		time.Time
}

// Prediction creator - output data slices coupled with data type.
// Attribute smoothedData *[](*model.FinalData) - smoothing correction of data slices. See model.FinalData.
// Attribute predictedData *[](*model.FinalData) - prediction up to the specified horizon. See model.FinalData.
// Attribute *model.DataType - data type to which the prediction is created (individual smoothing is needed too).
type OutputData struct {
	*DataType
	smoothedData	*[](*FinalData)
	predictedData	*[](*FinalData)
}

// Creating of DataRouter instance.
// Returning *DataRouter - instance that allows collecting and analysing of final smoothed and predicted data.
func NewDataRouter() *DataRouter {
	dataList := make([](*OutputData), 0)
	semaphore := &sync.Mutex{}
	dataRouter := DataRouter{
		dataList: &dataList,
		semaphore: semaphore,
	}
	return &dataRouter
}

// Inserting of fresh output data entries (sorted by type name).
// Parameter dataType *DataType - data type which is predicted.
// Parameter smoothedData *[](*FinalData) - smoothed data entries.
// Parameter predictedData *[](*FinalData) - final forecasted data entries.
func (DataRouter *DataRouter) SetData(dataType *DataType, smoothedData *[](*FinalData), predictedData *[](*FinalData)) {
	DataRouter.semaphore.Lock()
	index := findDataByType(DataRouter, dataType)
	removeDataByType(DataRouter, index)
	addData(DataRouter, dataType, smoothedData, predictedData)
	sortDataByType(DataRouter)
	DataRouter.semaphore.Unlock()
}

// Removing of the specific data type from data router list.
// Parameter dataType *DataType - data type that was predicted.
func (DataRouter *DataRouter) RemoveDataByType(dataType *DataType) {
	DataRouter.semaphore.Lock()
	index := findDataByType(DataRouter, dataType)
	removeDataByType(DataRouter, index)
	DataRouter.semaphore.Unlock()
}

// Finding of the data type in the router list.
// Parameter DataRouter *DataRouter - struct instance.
// Parameter dataType *DataType - searched data type.
// Returning int - index into the data router list.
func findDataByType(DataRouter *DataRouter, dataType *DataType) int {
	for i, element := range *(DataRouter.dataList) {
		if strings.Compare(element.Name, dataType.Name) == 0 {
			return i
		}
	}
	return -1
}

// Removing of the router list entry by found index.
// Parameter DataRouter *DataRouter - struct instance.
// Parameter index int - index into the data router list.
func removeDataByType(DataRouter *DataRouter, index int) {
	if index >= 0 {
		list := *(DataRouter.dataList)
		newListPart1 := list[0:index]
		newListPart2 := list[index+1:]
		newList := append(newListPart1, newListPart2...)
		DataRouter.dataList = &newList
	}
}

// Adding of new entries to router list - smoothed, predicted data and data type.
// Parameter DataRouter *DataRouter - struct instance.
// Parameter dataType *DataType - output data type.
// Parameter smoothedData *[](*FinalData) - smoothed data.
func addData(DataRouter *DataRouter, dataType *DataType, smoothedData *[](*FinalData),
	predictedData *[](*FinalData)) {
	list := *(DataRouter.dataList)
	outputData := OutputData{
		smoothedData: smoothedData,
		predictedData: predictedData,
		DataType: dataType,
	}
	list = append(list, &outputData)
	DataRouter.dataList = &list
}

// Sorting of the data router list by type name (ascending order).
// Parameter DataRouter *DataRouter - struct instance.
func sortDataByType(DataRouter *DataRouter) {
	list := *(DataRouter.dataList)
	sort.SliceStable(list, func(i, j int) bool {
		return strings.Compare(list[i].Name, list[j].Name) == -1
	})
	DataRouter.dataList = &list
}