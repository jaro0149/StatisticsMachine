package controller

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"model"
	"fmt"
	"configuration"
	"encoding/json"
	"strconv"
)

// Attribute conf *model.RestConfiguration - REST settings - routing paths. See model.RestConfiguration.
// Attribute databaseController *model.StatisticalData - accessing of database operations. See model.StatisticalData.
// Attribute dataRouter *model.DataRouter - data router for setting final (forecasted or smoothed) data entries.
type RestController struct {
	restConfiguration	*model.RestConfiguration
	databaseController	*model.StatisticalData
	dataRouter			*model.DataRouter
}

// Creating instance of the RestController.
// Parameter conf *model.RestConfiguration - REST settings - routing paths. See model.RestConfiguration.
// Parameter databaseController *model.StatisticalData - accessing of database operations. See model.StatisticalData.
// Parameter dataRouter *model.DataRouter - data router for setting final (forecasted or smoothed) data entries.
// Returning *RestController - RestController object.
func NewRestController(conf *model.RestConfiguration, databaseController *model.StatisticalData,
	dataRouter *model.DataRouter) *RestController {
	restController := RestController {
		restConfiguration: conf,
		databaseController: databaseController,
		dataRouter: dataRouter,
	}
	return &restController
}

// Starting of REST controller (listening on selected routes).
func (RestController *RestController) StartRestController() {
	configuration.Info.Println("Initialisation of REST services.")
	fireUpServices := func() {
		// Services declaration
		r := httprouter.New()
		r.GET(RestController.restConfiguration.PathGetDataTypes, RestController.GetDataTypes)
		r.GET(RestController.restConfiguration.PathGetDataType, RestController.GetDataType)
		r.DELETE(RestController.restConfiguration.PathRemoveDataType, RestController.RemoveDataType)
		r.POST(RestController.restConfiguration.PathWriteNewDataType, RestController.WriteNewDataType)
		r.POST(RestController.restConfiguration.PathModifyDataType, RestController.ModifyDataType)
		// Starting of routing
		startingPath := fmt.Sprintf("localhost:%d", RestController.restConfiguration.LocalhostPort)
		err := http.ListenAndServe(startingPath, r)
		if err != nil {
			configuration.Error.Fatalf("REST server cannot be started: %s", err)
		}
	}
	go fireUpServices()
	configuration.Info.Println("REST services have been initialised successfully.")
}

// Fetching of all data types from database (REST API).
// Parameter w http.ResponseWriter - HTTP response channel. See http.ResponseWriter.
// Parameter r *http.Request - HTTP request header. See http.Request.
func (RestController *RestController) GetDataTypes(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	dataTypes := RestController.databaseController.ListDataTypes()
	jsonBytes, err := json.Marshal(*dataTypes)
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprintf(w, "%s", jsonBytes)
	} else {
		msg := fmt.Sprintf("An error occurred during marshaling of list of data types: %s\n", err)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(500)
		fmt.Fprintf(w, "%s", msg)
		configuration.Error.Printf(msg)
	}
}

// Fetching of single data type with specific ID from database (REST API).
// Parameter w http.ResponseWriter - HTTP response channel. See http.ResponseWriter.
// Parameter r *http.Request - HTTP request header. See http.Request.
// Parameter p httprouter.Params - URI parameter - id. See httprouter.Params.
func (RestController *RestController) GetDataType(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id, err01 := strconv.Atoi(p.ByName("id"))
	if err01 == nil {
		dataType, err02 := RestController.databaseController.GetDataType(uint(id))
		if err02 == nil {
			jsonBytes, err03 := json.Marshal(*dataType)
			if err03 == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(200)
				fmt.Fprintf(w, "%s", jsonBytes)
			} else {
				msg := fmt.Sprintf("An error occurred during marshaling of list of data types: %s\n", err03)
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(500)
				fmt.Fprintf(w, "%s", msg)
				configuration.Error.Printf(msg)
			}
		} else {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(404)
			fmt.Fprintf(w, "%s", err02)
		}
	} else {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(400)
		fmt.Fprintf(w, "%s", err01)
	}
}

// Removing of selected data type by id (REST API).
// Parameter w http.ResponseWriter - HTTP response channel. See http.ResponseWriter.
// Parameter r *http.Request - HTTP request header. See http.Request.
// Parameter p httprouter.Params - URI parameter - id. See httprouter.Params.
func (RestController *RestController) RemoveDataType(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id, err01 := strconv.Atoi(p.ByName("id"))
	if err01 == nil {
		dataType, err02 := RestController.databaseController.RemoveDataType(uint(id))
		if err02 == nil {
			RestController.dataRouter.RemoveDataByType(dataType)
			w.WriteHeader(200)
		} else {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(404)
			fmt.Fprintf(w, "%s", err02)
		}
	} else {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(400)
		fmt.Fprintf(w, "%s", err01)
	}
}

// Creating of new data type (REST API).
// Parameter w http.ResponseWriter - HTTP response channel. See http.ResponseWriter.
// Parameter r *http.Request - HTTP request header. See http.Request.
func (RestController *RestController) WriteNewDataType(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	dataType := model.DataType{}
	err01 := json.NewDecoder(r.Body).Decode(&dataType)
	if err01 == nil {
		newDataType, err02 := RestController.databaseController.WriteNewDataType(&dataType)
		if err02 == nil {
			jsonBytes, _ := json.Marshal(*newDataType)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			fmt.Fprintf(w, "%s", jsonBytes)
		} else {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(400)
			fmt.Fprintf(w, "%s", err02)
		}
	} else {
		msg := fmt.Sprintf("Input JSON cannot be decoded, http body: %s\n", err01)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(400)
		fmt.Fprintf(w, "%s", msg)
	}
}

// Modifying of existing data type in database (REST API).
// Parameter w http.ResponseWriter - HTTP response channel. See http.ResponseWriter.
// Parameter r *http.Request - HTTP request header. See http.Request.
// Parameter p httprouter.Params - URI parameter - id. See httprouter.Params.
func (RestController *RestController) ModifyDataType(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	errorsBucket := configuration.NewCompositeError()
	dataType := model.DataType{}
	err01 := json.NewDecoder(r.Body).Decode(&dataType)
	if err01 != nil {
		errorsBucket.AddError(1, fmt.Sprint(err01))
	}
	id, err02 := strconv.Atoi(p.ByName("id"))
	if err02 != nil {
		errorsBucket.AddError(1, fmt.Sprint(err02))
	}
	err03 := errorsBucket.Evaluate()
	if err03 == nil {
		err04 := RestController.databaseController.ModifyDataType(uint(id), &dataType)
		if err04 == nil {
			w.WriteHeader(200)
		} else {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(400)
			fmt.Fprintf(w, "%v", err04)
		}
	} else {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(400)
		fmt.Fprintf(w, "%v", err03)
	}
}
