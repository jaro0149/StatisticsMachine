package configuration

import (
	"log"
	"io"
)

var (
	// Tracing logger is used for evaluating of debugging information. See log.Logger.
	Trace	*log.Logger
	// Info logger is used for evaluating of informative messages. See log.Logger.
	Info	*log.Logger
	// Warning logger is used for evaluating of warnings that aren't fatal for further execution of program.
	// See log.logger.
	Warning	*log.Logger
	// Error buffer should be used only for fatal errors that doesn't allow further execution of program.
	Error	*log.Logger
)

// Initialisation of all logers so they can be used.
func LoggingInit(
	traceHandle io.Writer,
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer) {

	Trace = log.New(traceHandle,
		"TRACE: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Info = log.New(infoHandle,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Warning = log.New(warningHandle,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Error = log.New(errorHandle,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
}