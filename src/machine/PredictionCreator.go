package machine

import (
	"model"
	"time"
	"math"
	"sync"
)

// Initial number of threads that serve data smoothing. This count is subsequently decreased if threads cannot be
// fitted with data slice.
const SMOOTHING_THREADS_START = 4

// Prediction creator - important data slices and semaphores.
// Attribute smoothedData *[](*model.FinalData) - smoothing correction of data slices. See model.FinalData.
// Attribute predictedData *[](*model.FinalData) - prediction up to the specified horizon. See model.FinalData.
// Attribute *model.DataType - data type to which the prediction is created (individual smoothing is needed too).
type PredictionCreator struct {
	smoothedData	*[](*model.FinalData)
	predictedData	*[](*model.FinalData)
	*model.DataType
}

// Initial smoothing of data slice - creating of periodic intervals with specified cell size (time window).
// Parameter conf *model.PredictionConfiguration - configuration settings - smoothing cell size.
// See model.PredictionConfiguration.
// Parameter dataSlice *[](*model.Data) - original data slice with frames bytes and timestamps. See model.Data.
// Returning *[](*model.FinalData) - smoothed data vector. See model.FinalData.
func (PredictionCreator *PredictionCreator) SmoothData(conf *model.PredictionConfiguration,
	dataSlice *[](*model.Data)) (*[](*model.FinalData)) {
	if len(*dataSlice) != 0 {
		mutex := &sync.Mutex{}
		smoothedData := initSmoothingSlice(conf, dataSlice)
		assignSmoothingJobs(conf, dataSlice, SMOOTHING_THREADS_START, smoothedData, mutex)
		return smoothedData
	} else {
		smoothedData := make([](*model.FinalData), 0)
		return &smoothedData
	}
}

// Initialisation of the smoothing slice.
// Parameter conf *model.PredictionConfiguration - configuration settings - smoothing cell size.
// Parameter dataSlice *[](*model.Data) - original data slice with frames bytes and timestamps. See model.Data.
// Returning (*[](*model.FinalData) - initialised smoothed slice (zeros) with specific final size. See model.FinalData.
func initSmoothingSlice(conf *model.PredictionConfiguration, dataSlice *[](*model.Data)) (*[](*model.FinalData)) {
	parts := findPartsCount(conf, dataSlice)
	smoothedData := make([](*model.FinalData), parts)
	return &smoothedData
}

// Assigning of smoothing jobs to several threads (concurrent evaluation of smoothing vector).
// PredictionCreator *PredictionCreator - object of this class. See PredictionCreator.
// Parameter conf *model.PredictionConfiguration - configuration settings - smoothing cell size.
// See model.PredictionConfiguration.
// Parameter dataSlice *[](*model.Data) - original data slice with frames bytes and timestamps. See model.Data.
// Parameter numberOfThreads int - actual number of threads - this count is subsequently decreased if the selected count
// cannot fit length of the data slice (recursion).
// Parameter smoothedData *[](*model.FinalData) - smoothed data vector (empty). See model.FinalData.
// Parameter  smoothingMutex *sync.Mutex - semaphore that controls access to smoothedData parameter. See sync.Mutex.
func assignSmoothingJobs(conf *model.PredictionConfiguration, dataSlice *[](*model.Data), numberOfThreads int,
	smoothedData *[](*model.FinalData), smoothingMutex *sync.Mutex) {
	dataSliceBody := *dataSlice
	parts := findPartsCount(conf, dataSlice)
	threadParts := uint(math.Floor(float64(parts) / float64(numberOfThreads)))
	if threadParts >= 1 {
		sliceLength := conf.SmoothingRange * threadParts
		startIndex := uint(0)
		waitingGroup := &sync.WaitGroup{}
		waitingGroup.Add(numberOfThreads)
		for i:=0; i<numberOfThreads; i++ {
			smoothingIndex := uint(i) * threadParts
			sliceStart := dataSliceBody[0].Time.Add(time.Duration(smoothingIndex*conf.SmoothingRange)*time.Millisecond)
			if i != numberOfThreads - 1 {
				endIndex := getNextIndex(dataSlice, startIndex, sliceStart, sliceLength) - 1
				smoothingThread(conf, dataSlice, startIndex, sliceStart, endIndex, sliceLength, smoothingIndex,
					smoothedData, smoothingMutex, waitingGroup)
				startIndex = endIndex + 1
			} else {
				endIndex := uint(len(dataSliceBody)) - 1
				sliceLength = uint(dataSliceBody[endIndex].Time.Sub(sliceStart).Nanoseconds()/1000000)
				smoothingThread(conf, dataSlice, startIndex, sliceStart, endIndex, sliceLength, smoothingIndex,
					smoothedData, smoothingMutex, waitingGroup)
			}
		}
		waitingGroup.Wait()
	} else {
		assignSmoothingJobs(conf, dataSlice, numberOfThreads - 1, smoothedData, smoothingMutex)
	}
}

// Smoothing of selected part of the data slice.
// PredictionCreator *PredictionCreator - object of this class. See PredictionCreator.
// Parameter conf *model.PredictionConfiguration - configuration settings - smoothing cell size.
// See model.PredictionConfiguration.
// Parameter dataSlice *[](*model.Data) - original data slice with frames bytes and timestamps. See model.Data.
// Parameter startIndex uint - starting index of the data slice part.
// Parameter sliceStart time.Time - starting time of data slice part. See time.Time.
// Parameter endIndex uint - the last index of the data slice part.
// Parameter sliceLength uint - the length of the actual data slice (milliseconds).
// Parameter smoothingIndex uint - index from which new smoothed data entries are written.
// Parameter smoothedData *[](*model.FinalData) - smoothed data vector (empty). See model.FinalData.
// Parameter smoothingMutex *sync.Mutex - semaphore that controls access to smoothedData parameter. See sync.Mutex.
// Parameter waitingGroup *sync.WaitGroup - this parameter ensures that final smoothed slice is returned only after all
// smoothing jobs are done. See sync.WaitGroup
func smoothingThread(conf *model.PredictionConfiguration, dataSlice *[](*model.Data), startIndex uint,
	sliceStart time.Time, endIndex uint, sliceLength uint, smoothingIndex uint, smoothedData *[](*model.FinalData),
	smoothingMutex *sync.Mutex,	waitingGroup *sync.WaitGroup) {
	defer waitingGroup.Done()
	dataSliceBody := *dataSlice
	smoothedDataRef := *smoothedData
	runningTime := sliceStart.Add(time.Duration(conf.SmoothingRange) * time.Millisecond)
	if startIndex <= endIndex {
		var dataBuffer []uint
		for i := startIndex; i <= endIndex || uint(runningTime.Sub(sliceStart).Nanoseconds()/1000000) <= sliceLength; {
			if dataSliceBody[i].Time.Before(runningTime) {
				dataBuffer = append(dataBuffer, dataSliceBody[i].Bytes)
				i++
			} else {
				avg := average(&dataBuffer)
				smoothingMutex.Lock()
				smoothedDataRef[smoothingIndex] = &model.FinalData{
					DataElement: avg,
					Timestamp:   runningTime}
				smoothingMutex.Unlock()
				runningTime = runningTime.Add(time.Duration(conf.SmoothingRange) * time.Millisecond)
				smoothingIndex ++
				dataBuffer = nil
			}
		}
		if len(dataBuffer) != 0 {
			avg := average(&dataBuffer)
			smoothingMutex.Lock()
			smoothedDataRef[smoothingIndex] = &model.FinalData{
				DataElement: avg,
				Timestamp:   runningTime}
			smoothingMutex.Unlock()
		}
	} else {
		parts := uint(math.Ceil(float64(sliceLength)/float64(conf.SmoothingRange)))
		for i:=uint(0); i<parts; i++ {
			smoothedDataRef[smoothingIndex+i] = &model.FinalData{
				DataElement: float64(0.0),
				Timestamp:   runningTime}
		}
	}
}

// Computing average of the uint slice.
// Parameter dataBuffer *([]uint) - slice with bytes count.
// Returning float64 - average or mean with constant weights.
func average(dataBuffer *([]uint)) float64 {
	if len(*dataBuffer) != 0 {
		var sum uint64 = 0
		for i := range *dataBuffer {
			sum += uint64((*dataBuffer)[i])
		}
		average := float64(sum) / float64(len(*dataBuffer))
		return average
	} else {
		return float64(0)
	}
}

// Determining of next index in the data slice that occurs within elapsed slice length.
// Parameter dataSlice *[](*model.Data) - original data slice with frames bytes and timestamps. See model.Data.
// Parameter startIndex uint - index after which the next index is searched.
// Parameter sliceStart time.Time - starting time of data slice part. See time.Time.
// Parameter sliceLength uint - length of the data slice that is assigned to one thread (milliseconds).
// Returning uint - next index into the data slice (after elapsed sliceLength).
func getNextIndex(dataSlice *[](*model.Data), startIndex uint, sliceStart time.Time, sliceLength uint) uint {
	dataSliceRef := *dataSlice
	dataLength := uint(len(dataSliceRef))
	timeCorner := sliceStart.Add(time.Duration(sliceLength) * time.Millisecond)
	indexCounter := startIndex
	for ; (indexCounter != dataLength) && (dataSliceRef[indexCounter].Time.Before(timeCorner)); indexCounter++ {}
	return indexCounter
}

// Estimation of count of smoothing parts.
// Parameter conf *model.PredictionConfiguration - configuration settings - smoothing cell size.
// Parameter dataSlice *[](*model.Data) - original data slice with frames bytes and timestamps. See model.Data.
// Returning uint - estimated count of smoothing parts.
func findPartsCount(conf *model.PredictionConfiguration, dataSlice *[](*model.Data)) uint {
	arrayLength := (*dataSlice)[len(*dataSlice)-1].Time.Sub((*dataSlice)[0].Time).Nanoseconds()/1000000
	parts := uint(math.Ceil(float64(arrayLength) / float64(conf.SmoothingRange)))
	if parts * conf.SmoothingRange == uint(arrayLength) {
		parts++
	}
	return parts
}