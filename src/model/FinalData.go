package model

import (
	"time"
)

type DataRouter struct {}

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
	smoothedData	*[](*FinalData)
	predictedData	*[](*FinalData)
	*DataType
}

// Creating of DataRouter instance.
// Returning *DataRouter - instance that allows collecting and analysing of final smoothed and predicted data.
func NewDataRouter() *DataRouter {
	return &DataRouter{}
}

