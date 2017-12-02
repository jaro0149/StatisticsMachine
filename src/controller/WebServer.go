package controller

import (
	"net/http"
	"model"
	"configuration"
	"strconv"
)

// Attribute configuration *model.WebServerConfiguration - web server settings - port and path to web files.
type WebServer struct {
	configuration 	*model.WebServerConfiguration
}

// Creating instance of WebServer.
// Parameter configuration *model.WebServerConfiguration - web server settings - port and path to web files.
// Returning *WebServer - instance of web server manager.
func NewWebServer(conf *model.WebServerConfiguration) *WebServer {
	webServer := WebServer{
		configuration: conf,
	}
	return &webServer
}

// Starting of web server using files located at specified path; server will listen on specified TCP port.
func (WebServer *WebServer) StartWebServer() {
	configuration.Info.Println("Initialisation of WEB server.")
	serverMux := http.NewServeMux()
	srv := &http.Server{
		Addr: ":" + strconv.Itoa(int(WebServer.configuration.LocalhostPort)),
		Handler: serverMux,
	}
	serverMux.Handle("/", http.StripPrefix("/",
		http.FileServer(http.Dir(WebServer.configuration.RootPath))))
	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			configuration.Error.Panicf("REST server cannot be started: %v", err)
		}
	}()
	configuration.Info.Println("WEB server has been started successfully.")
}