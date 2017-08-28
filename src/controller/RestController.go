package controller

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"model"
	"fmt"
	"configuration"
)

// Attribute conf *model.RestConfiguration - REST settings - routing paths.
type RestController struct {
	restConfiguration *model.RestConfiguration
}

// Creating instance of the RestController.
// Parameter conf *model.RestConfiguration - REST settings - routing paths.
// Returning *RestController - RestController object.
func NewRestController(conf *model.RestConfiguration) *RestController {
	restController := RestController{restConfiguration: conf}
	return &restController
}

// Starting of REST controller (listening on selected routes).
func (RestController *RestController) StartRestController() {
	configuration.Info.Println("Initialisation of REST services.")
	fireUpServices := func() {
		// Services declaration
		r := httprouter.New()

		// Starting of routing
		startingPath := fmt.Sprintf("localhost:%d", RestController.restConfiguration.LocalhostPort)
		err := http.ListenAndServe(startingPath, r)
		if err != nil {
			configuration.Error.Fatalf("REST server cannot be started: %s", err);
		}
	}
	go fireUpServices()
	configuration.Info.Println("REST services have been initialised successfully.")
}