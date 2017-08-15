package configuration

import (
	"time"
	"bytes"
	"fmt"
	"errors"
)

// List or buffer of errors that are collected.
var errorsList []CompositeError = make([]CompositeError, 0, 10)

// Error consists of identification (int), description (string), and time (time.Time).
// Attribute Id int - identification of error.
// Attribute Description string - description of error.
// Attribute Time time.Time - Auto-generated time when an error was added to list. See time.Time.
type CompositeError struct {
	Id			int
	Description	string
	Time		time.Time
}

// Adding of a new error to the error buffer.
// Parameter 'id int' - identification of error (doesn't have to be unique).
// Parameter 'description string' - description of error.
func(compositeError *CompositeError) AddError(id int, description string) {
	actualTime := time.Now()
	errorsList = append(errorsList, CompositeError{Id: id, Description: description, Time: actualTime})
}

// Evaluating of error - if the error buffer is not empty, an error is thrown or returned.
func(compositeError *CompositeError) Evaluate() error {
	if len(errorsList) != 0 {
		var buffer bytes.Buffer
		for i := range errorsList {
			line := fmt.Sprintf("%d, %s: %s\n", errorsList[i].Id, errorsList[i].Time,
				errorsList[i].Description)
			buffer.WriteString(line)
		}
		return errors.New(buffer.String())
	} else {
		return nil
	}
}