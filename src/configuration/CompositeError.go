package configuration

import (
	"time"
	"bytes"
	"fmt"
	"errors"
)

// Attribute errorsList []ErrorEntry - list of collected errors.
type CompositeError struct {
	errorsList []ErrorEntry
}

// Error consists of identification (int), description (string), and time (time.Time).
// Attribute Id int - identification of error.
// Attribute Description string - description of error.
// Attribute Time time.Time - Auto-generated time when an error was added to list. See time.Time.
type ErrorEntry struct {
	Id			int
	Description	string
	Time		time.Time
}

// Creating of the new composite error.
// Returning *CompositeError - object of CompositeError.
func NewCompositeError() *CompositeError {
	return &CompositeError{}
}

// Adding of a new error to the error buffer.
// Parameter 'id int' - identification of error (doesn't have to be unique).
// Parameter 'description string' - description of error.
func(CompositeError *CompositeError) AddError(id int, description string) {
	actualTime := time.Now()
	CompositeError.errorsList = append(CompositeError.errorsList, ErrorEntry {
		Id: id,
		Description: description,
		Time: actualTime,
	})
}

// Evaluating of error - if the error buffer is not empty, an error is thrown or returned.
func(CompositeError *CompositeError) Evaluate() error {
	if len(CompositeError.errorsList) != 0 {
		var buffer bytes.Buffer
		for i := range CompositeError.errorsList {
			line := fmt.Sprintf("%d, %s: %s\n",
				CompositeError.errorsList[i].Id,
				CompositeError.errorsList[i].Time,
				CompositeError.errorsList[i].Description)
			buffer.WriteString(line)
		}
		return errors.New(buffer.String())
	} else {
		return nil
	}
}