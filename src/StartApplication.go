package main

import (
	"configuration"
	"io/ioutil"
	"os"
)

func main() {
	configuration.LoggingInit(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)
	configuration.ConnectDatabase()
	defer configuration.CloseDatabase()
	configuration.Info.Println("ddd")
}
