package machine

import (
	"model"
	"time"
	"math"
	"sync"
)

// Attribute SmoothingRange uint - time range (milliseconds) that is smoothed to one point in time.
// Attribute SmoothingThreads uint - Initial number of threads that serve data smoothing. This count is subsequently
// decreased if threads cannot be fitted with data slice.
type SmoothingCreator struct {
	smoothingRange		uint
	smoothingThreads	uint
}

// Creating instance of the SmoothingCreator.
// Parameter SmoothingRange uint - time range (milliseconds) that is smoothed to one point in time.
// Parameter SmoothingThreads uint - Initial number of threads that serve data smoothing. This count is subsequently
// decreased if threads cannot be fitted with data slice.
// Returning *SmoothingCreator - SmoothingCreator object.
func NewSmoothingCreator(smoothingRange uint, smoothingThreads uint) *SmoothingCreator {
	predictionCreator := SmoothingCreator{
		smoothingRange: smoothingRange,
		smoothingThreads: smoothingThreads,
	}
	return &predictionCreator
}

// Initial smoothing of data slice - creating of periodic intervals with specified cell size (time window).
// Parameter dataSlice *[](*model.Data) - original data slice with frames bytes and timestamps. See model.Data.
// Returning *[](*model.FinalData) - smoothed data vector. See model.FinalData.
func (SmoothingCreator *SmoothingCreator) SmoothData(dataSlice *[](*model.Data)) (*[](*model.FinalData)) {
	if len(*dataSlice) != 0 {
		mutex := &sync.Mutex{}
		smoothedData := initSmoothingSlice(SmoothingCreator.smoothingRange, dataSlice)
		assignSmoothingJobs(SmoothingCreator.smoothingRange, dataSlice, int(SmoothingCreator.smoothingThreads),
			smoothedData, mutex)
		return smoothedData
	} else {
		smoothedData := make([](*model.FinalData), 0)
		return &smoothedData
	}
}

// Initialisation of the smoothing slice.
// Parameter smoothingRange uint - configuration setting - smoothing cell size.
// Parameter dataSlice *[](*model.Data) - original data slice with frames bytes and timestamps. See model.Data.
// Returning (*[](*model.FinalData) - initialised smoothed slice (zeros) with specific final size. See model.FinalData.
func initSmoothingSlice(smoothingRange uint, dataSlice *[](*model.Data)) (*[](*model.FinalData)) {
	parts := findPartsCount(smoothingRange, dataSlice)
	smoothedData := make([](*model.FinalData), parts)
	return &smoothedData
}

// Assigning of smoothing jobs to several threads (concurrent evaluation of smoothing vector).
// Parameter smoothingRange uint - configuration setting - smoothing cell size.
// Parameter dataSlice *[](*model.Data) - original data slice with frames bytes and timestamps. See model.Data.
// Parameter numberOfThreads int - actual number of threads - this count is subsequently decreased if the selected count
// cannot fit length of the data slice (recursion).
// Parameter smoothedData *[](*model.FinalData) - smoothed data vector (empty). See model.FinalData.
// Parameter  smoothingMutex *sync.Mutex - semaphore that controls access to smoothedData parameter. See sync.Mutex.
func assignSmoothingJobs(smoothingRange	uint, dataSlice *[](*model.Data), numberOfThreads int,
	smoothedData *[](*model.FinalData), smoothingMutex *sync.Mutex) {
	dataSliceBody := *dataSlice
	parts := findPartsCount(smoothingRange, dataSlice)
	threadParts := uint(math.Floor(float64(parts) / float64(numberOfThreads)))
	if threadParts >= 1 {
		sliceLength := smoothingRange * threadParts
		startIndex := uint(0)
		waitingGroup := &sync.WaitGroup{}
		waitingGroup.Add(numberOfThreads)
		for i:=0; i<numberOfThreads; i++ {
			smoothingIndex := uint(i) * threadParts
			sliceStart := dataSliceBody[0].Time.Add(time.Duration(smoothingRange*smoothingIndex)*time.Millisecond)
			if i != numberOfThreads - 1 {
				endIndex := getNextIndex(dataSlice, startIndex, sliceStart, sliceLength) - 1
				smoothingThread(smoothingRange, dataSlice, startIndex, sliceStart, endIndex, sliceLength, smoothingIndex,
					smoothedData, smoothingMutex, waitingGroup)
				startIndex = endIndex + 1
			} else {
				endIndex := uint(len(dataSliceBody)) - 1
				sliceLength = uint(dataSliceBody[endIndex].Time.Sub(sliceStart).Nanoseconds()/1000000)
				smoothingThread(smoothingRange, dataSlice, startIndex, sliceStart, endIndex, sliceLength, smoothingIndex,
					smoothedData, smoothingMutex, waitingGroup)
			}
		}
		waitingGroup.Wait()
	} else {
		assignSmoothingJobs(smoothingRange, dataSlice, numberOfThreads - 1, smoothedData, smoothingMutex)
	}
}

// Smoothing of selected part of the data slice.
// Parameter smoothingRange uint - configuration setting - smoothing cell size.
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
func smoothingThread(smoothingRange uint, dataSlice *[](*model.Data), startIndex uint,
	sliceStart time.Time, endIndex uint, sliceLength uint, smoothingIndex uint, smoothedData *[](*model.FinalData),
	smoothingMutex *sync.Mutex,	waitingGroup *sync.WaitGroup) {
	defer waitingGroup.Done()
	dataSliceBody := *dataSlice
	smoothedDataRef := *smoothedData
	runningTime := sliceStart.Add(time.Duration(smoothingRange) * time.Millisecond)
	if startIndex <= endIndex {
		var dataBuffer []uint
		for i := startIndex; i <= endIndex || uint(runningTime.Sub(sliceStart).Nanoseconds()/1000000) <= sliceLength; {
			if dataSliceBody[i].Time.Before(runningTime) {
				dataBuffer = append(dataBuffer, dataSliceBody[i].Bytes)
				i++
			} else {
				sumX := sum(&dataBuffer)
				smoothingMutex.Lock()
				smoothedDataRef[smoothingIndex] = &model.FinalData{
					DataElement: sumX,
					Timestamp:   runningTime}
				smoothingMutex.Unlock()
				runningTime = runningTime.Add(time.Duration(smoothingRange) * time.Millisecond)
				smoothingIndex ++
				dataBuffer = nil
			}
		}
		if len(dataBuffer) != 0 {
			sumX := sum(&dataBuffer)
			smoothingMutex.Lock()
			smoothedDataRef[smoothingIndex] = &model.FinalData{
				DataElement: sumX,
				Timestamp:   runningTime}
			smoothingMutex.Unlock()
		}
	} else {
		parts := uint(math.Ceil(float64(sliceLength)/float64(smoothingRange)))
		for i:=uint(0); i<parts; i++ {
			smoothedDataRef[smoothingIndex+i] = &model.FinalData{
				DataElement: uint64(0),
				Timestamp:   runningTime}
		}
	}
}

// Computing sum of the uint slice.
// Parameter dataBuffer *([]uint) - slice with bytes count.
// Returning float64 - sum.
func sum(dataBuffer *([]uint)) uint64 {
	dataRef := *dataBuffer
	if len(dataRef) != 0 {
		var sum uint64 = 0
		for i := range dataRef {
			sum += uint64(dataRef[i])
		}
		return sum
	} else {
		return uint64(0)
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
// Parameter smoothingRange uint - configuration setting - smoothing cell size.
// Parameter dataSlice *[](*model.Data) - original data slice with frames bytes and timestamps. See model.Data.
// Returning uint - estimated count of smoothing parts.
func findPartsCount(smoothingRange uint, dataSlice *[](*model.Data)) uint {
	arrayLength := (*dataSlice)[len(*dataSlice)-1].Time.Sub((*dataSlice)[0].Time).Nanoseconds()/1000000
	parts := uint(math.Ceil(float64(arrayLength) / float64(smoothingRange)))
	if parts * smoothingRange == uint(arrayLength) {
		parts++
	}
	return parts
}