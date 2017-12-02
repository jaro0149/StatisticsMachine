package machine

import (
	"strings"
)

// Attribute dataTypeId uint - ID of the data type (unique characterisation of the data type).
// Attribute dataTypeName string - identification of the data type displayed on LCD.
// Attribute direction uint - RX: 0, TX: 1.
// Attribute prediction	bool - state of the forecasting (switched on / off).
type DisplayTemplate struct {
	dataTypeId		uint
	dataTypeName 	string
	direction		uint
	prediction		bool
}

// Slice with display templates that is used for sorting.
type DisplayTemplateSlice []DisplayTemplate

// Method that returns number of displays in slice (sort interface). See sort.
// Returning int - slice length.
func (s DisplayTemplateSlice) Len() int {
	return len(s)
}

// Swapping of two display elements in slice (sort interface). See sort.
// Parameter i int - first display template.
// Parameter j int - second display template.
func (s DisplayTemplateSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Comparing of two display elements in slice (sort interface). See sort.
// Parameter i int - first display template.
// Parameter j int - second display template.
// Returning bool - true if "i" element has precedence over "j" element.
func (s DisplayTemplateSlice) Less(i, j int) bool {
	nameI := s[i].dataTypeName
	nameJ := s[j].dataTypeName
	directionI := s[i].direction
	directionJ := s[j].direction
	predictionI := s[i].prediction
	namesComparison := strings.Compare(nameI, nameJ)
	if namesComparison == -1 {
		return true
	} else if namesComparison == 1 {
		return false
	} else if directionI < directionJ {
		return true
	} else if directionI > directionJ {
		return false
	} else if predictionI {
		return true
	} else {
		return false
	}
}