package machine

import (
	"model"
	"gobot.io/x/gobot"
	"gobot.io/x/gobot/platforms/raspi"
	"gobot.io/x/gobot/drivers/gpio"
	"configuration"
	"os/exec"
	"fmt"
	"sync"
	"strconv"
	"sort"
)

// Initial first LCD line.
const BOOT_FIRST_LINE = "Vnorene systemy"
// Initial second LCD line.
const BOOT_SECOND_LINE = "ZS 2017"
// Maximum length of LCD line (number of characters).
const LINE_LENGTH uint = 16
// String representation of input traffic direction.
const DIRECTION_RX = "RX"
// String representation of upstream traffic direction.
const DIRECTION_TX = "TX"
// Conversion ration between bytes and kilo-bytes.
const CONVERSION_RATIO_KB = 1000
// Conversion ration between bytes and mega-bytes.
const CONVERSION_RATIO_MB = 1000000
// When this bytes treshold is reached or exceeded, mega-bytes format instead of kilo-bytes format must be used.
const CONVERSION_TRESHOLD_KB_MB = 10000000000
// When this bytes treshold is reached or exceeded, kilo-bytes format instead of bytes format must be used.
const CONVERSION_TRESHOLD_B_KB = 10000
// Basic rate - 1000 ms = 1s - it doesn't have to be explicitly displayed as nnn/1s.
const BASIC_RATE = 1000
// String representation of bytes.
const B_UNIT = "B"
// String representation of kilo-bytes.
const KB_UNIT = "kB"
// String representation of mega-bytes.
const MB_UNIT = "MB"

// Attribute configData *model.GPIOConfiguration - GPIO bits layout. See model.GPIOConfiguration.
// Attribute lcdMutex *sync.Mutex - semaphore that controls access to LCD device. See sync.Mutex.
// Attribute displayMutex *sync.Mutex - semaphore that controls access to LCD displayed information. See sync.Mutex.
// Attribute allDisplays *map[DisplayTemplate]*OutputData - actual list of displays - information that can be shown
// on LCD. See DisplayTemplate.
// Attribute actualDisplay *DisplayTemplate - identification of information that are actually presented on LCD. See
// DisplayTemplate.
// Attribute smoothingRange uint - smoothing range in milliseconds.
// Attribute ledMutex *sync.Mutex - controlling of access to LED Neopixel strip.
// Attribute designator	float64 - it describes criterion for changing prediction state - fraction of bandwidth that
// must exceeded from actual load (positivw or negative fraction domain).
// Attribute linkBandwidth uint64 - observed link bandwidth (maximum load) [bytes/s].
// Attribute robot *gobot.Robot - buttons listeners.
type DeviceManager struct {
	configData		*model.PHYConfiguration
	lcdMutex		*sync.Mutex
	displayMutex	*sync.Mutex
	ledMutex		*sync.Mutex
	allDisplays		*map[DisplayTemplate]float64
	actualDisplay	*DisplayTemplate
	smoothingRange	uint
	designator		float64
	linkBandwidth	uint64
	robot			*gobot.Robot
}

// Building of DeviceManager object (assigment or initialisation of required attributes).
// Parameter configData *model.GPIOConfiguration - GPIO bits layout. See model.GPIOConfiguration.
// Parameter smoothingRange uint - smoothing range in milliseconds.
// Parameter designator	float64 - it describes criterion for changing prediction state - fraction of bandwidth that
// must exceeded from actual load (positivw or negative fraction domain).
// Returning *DeviceManager - built instance of DeviceManager structure (its reference). See DeviceManager.
func NewDeviceManager(conf *model.PHYConfiguration, smoothingRange uint, designator	float64,
	linkBandwidth	uint64) *DeviceManager {
	var lcdMutex = &sync.Mutex{}
	var displayMutex = &sync.Mutex{}
	var ledMutex = &sync.Mutex{}
	allDisplays := make(map[DisplayTemplate]float64)
	ioDeviceManager := DeviceManager {
		configData:			conf,
		lcdMutex:			lcdMutex,
		actualDisplay:		nil,
		allDisplays:		&allDisplays,
		smoothingRange:		smoothingRange,
		displayMutex:		displayMutex,
		ledMutex:			ledMutex,
		designator:			designator,
		linkBandwidth:		linkBandwidth,
	}
	return &ioDeviceManager
}

// Starting of DeviceManager  listening to button events and initialisation of LCD display.
func (DeviceManager *DeviceManager) StartDeviceManager() {
	DeviceManager.initButtonPins()
	go DeviceManager.runButtonsHandlers()
	DeviceManager.WriteMessageOnLcd(BOOT_FIRST_LINE, BOOT_SECOND_LINE)
}

// Initialisation of button pins - mode and pull ip resistor.
func (DeviceManager *DeviceManager) initButtonPins() {
	err := exec.Command(
		"python",
		"gpio_init.py",
		fmt.Sprint(DeviceManager.configData.PhyLeftButton),
		fmt.Sprint(DeviceManager.configData.PhyRightButton),
	).Run()
	if err != nil {
		configuration.Error.Panicf("An error occurred during configuration of buttons pins: %v", err)
	}
}

// Starting of button handlers (left and right button for changing of displayed information).
func (DeviceManager *DeviceManager) runButtonsHandlers()  {
	configuration.Info.Println("Configuration of button handlers started.")
	r := raspi.NewAdaptor()
	leftButton := gpio.NewButtonDriver(r, strconv.Itoa(int(DeviceManager.configData.PhyLeftButton)))
	rightButton := gpio.NewButtonDriver(r, strconv.Itoa(int(DeviceManager.configData.PhyRightButton)))
	work := func() {
		leftButton.On(gpio.ButtonRelease, func(data interface{}) {
			DeviceManager.handleLeftButtonPushed()
		})
		rightButton.On(gpio.ButtonRelease, func(data interface{}) {
			DeviceManager.handleRightButtonPushed()
		})
	}
	robot := gobot.NewRobot("buttonBot",
		[]gobot.Connection{r},
		[]gobot.Device{leftButton, rightButton},
		work,
	)
	DeviceManager.robot = robot
	configuration.Info.Println("Button handlers have been successfully started.")
	robot.Start()
}

// Left button handler - displaying of previous load / prediction information.
func (DeviceManager *DeviceManager) handleLeftButtonPushed() {
	configuration.Info.Println("Left button has been pushed.")
	DeviceManager.displayMutex.Lock()
	defer DeviceManager.displayMutex.Unlock()
	if len(*DeviceManager.allDisplays) != 0 {
		sortedDisplays := getSortedDisplays(DeviceManager.allDisplays)
		index := getIndexOfActualDataType(sortedDisplays, DeviceManager.actualDisplay)
		if index != 0 {
			DeviceManager.setPreviousDisplay(sortedDisplays, index)
		}
	}
}

// Right button handler - displaying of next load / prediction information.
func (DeviceManager *DeviceManager) handleRightButtonPushed() {
	configuration.Info.Println("Right button has been pushed.")
	DeviceManager.displayMutex.Lock()
	defer DeviceManager.displayMutex.Unlock()
	if len(*DeviceManager.allDisplays) != 0 {
		sortedDisplays := getSortedDisplays(DeviceManager.allDisplays)
		index := getIndexOfActualDataType(sortedDisplays, DeviceManager.actualDisplay)
		if index != uint(len(*sortedDisplays) - 1) {
			DeviceManager.setNextDisplay(sortedDisplays, index)
		}
	}
}

// Sorting of displays by their name, direction, and prediction flag.
// Parameter allDisplays *map[DisplayTemplate]*CalculatedData - all displays. See DisplayTemplate.
// Returning *[]DisplayTemplate - sorted display templates (keys of map).
func getSortedDisplays(allDisplays *map[DisplayTemplate]float64) *[]DisplayTemplate {
	var displayStack []DisplayTemplate
	for display := range *allDisplays {
		displayStack = append(displayStack, display)
	}
	sort.Sort(DisplayTemplateSlice(displayStack))
	return &displayStack
}

// Fetching of index of display that is actually shown to user via LDC interface.
// Parameter displays *[]DisplayTemplate - sorted displays. See DisplayTemplate.
// Parameter searchedDisplay *DisplayTemplate - searched display. See DisplayTemplate.
// Returning uint - index of found display; if it is not found, first display index is returned (0).
func getIndexOfActualDataType(displays *[]DisplayTemplate, searchedDisplay *DisplayTemplate) uint {
	if searchedDisplay != nil {
		for index, display := range *displays {
			if display == *searchedDisplay {
				return uint(index)
			}
		}
		for index, display := range *displays {
			if display.dataTypeName == searchedDisplay.dataTypeName {
				return uint(index)
			}
		}
	}
	return 0
}

// Writing of message to LCD device.
// Parameter line1 string - first line.
// Parameter line2 string - second line.
func (DeviceManager *DeviceManager) WriteMessageOnLcd(line1 string, line2 string) {
	DeviceManager.lcdMutex.Lock()
	defer DeviceManager.lcdMutex.Unlock()
	err := exec.Command(
		"python",
		"char_lcd.py",
		fmt.Sprint(DeviceManager.configData.BCM_RS),
		fmt.Sprint(DeviceManager.configData.BCM_EN),
		fmt.Sprint(DeviceManager.configData.BCM_DB4),
		fmt.Sprint(DeviceManager.configData.BCM_DB5),
		fmt.Sprint(DeviceManager.configData.BCM_DB6),
		fmt.Sprint(DeviceManager.configData.BCM_DB7),
		fmt.Sprint(DeviceManager.configData.BCM_Backlight),
		line1,
		line2,
	).Run()
	if err != nil {
		configuration.Error.Panicf("An error occurred during writing of message to LCD display: %v", err)
	}
}

// Clearing of both lines of LCD.
func (DeviceManager *DeviceManager) ClearLcd() {
	DeviceManager.lcdMutex.Lock()
	defer DeviceManager.lcdMutex.Unlock()
	err := exec.Command(
		"python",
		"char_lcd.py",
		fmt.Sprint(DeviceManager.configData.BCM_RS),
		fmt.Sprint(DeviceManager.configData.BCM_EN),
		fmt.Sprint(DeviceManager.configData.BCM_DB4),
		fmt.Sprint(DeviceManager.configData.BCM_DB5),
		fmt.Sprint(DeviceManager.configData.BCM_DB6),
		fmt.Sprint(DeviceManager.configData.BCM_DB7),
		fmt.Sprint(DeviceManager.configData.BCM_Backlight),
		BOOT_FIRST_LINE,
		BOOT_SECOND_LINE,
	).Run()
	if err != nil {
		configuration.Error.Panicf("An error occurred during clearing of LCD display: %v", err)
	}
}

// Processing of new average load identified by display template and resulting value.
// Parameter display *DisplayTemplate - fresh display information.
// Parameter result float64 - computed average load.
func (DeviceManager *DeviceManager) UpdateDisplayByLoad(display *DisplayTemplate, result float64) {
	DeviceManager.displayMutex.Lock()
	defer DeviceManager.displayMutex.Unlock()
	(*DeviceManager.allDisplays)[*display] = result
	if DeviceManager.actualDisplay != nil && *DeviceManager.actualDisplay == *display {
		DeviceManager.updateDisplayByLoadI(display, result)
	} else if DeviceManager.actualDisplay == nil {
		DeviceManager.updateDisplayByLoadI(display, result)
		DeviceManager.actualDisplay = display
	}
}

func (DeviceManager *DeviceManager) updateDisplayByLoadI(display *DisplayTemplate, result float64) {
	line1, line2 := getMeanLines(DeviceManager.smoothingRange, display, result)
	DeviceManager.WriteMessageOnLcd(line1, line2)
	DeviceManager.updateLcdDisplay(result)
}

// Updating of display by results of ARIMA forecasting model.
// Parameter display *DisplayTemplate - fresh display information.
// Parameter result float64 - computed prediction of load.
func (DeviceManager *DeviceManager) UpdateDisplayByPrediction(display *DisplayTemplate, result float64) {
	DeviceManager.displayMutex.Lock()
	defer DeviceManager.displayMutex.Unlock()
	(*DeviceManager.allDisplays)[*display] = result
	if DeviceManager.actualDisplay != nil && *DeviceManager.actualDisplay == *display {
		DeviceManager.updateDisplayByPredictionI(display, result)
	} else if DeviceManager.actualDisplay == nil {
		DeviceManager.updateDisplayByPredictionI(display, result)
		DeviceManager.actualDisplay = display
	}
}

func (DeviceManager *DeviceManager) updateDisplayByPredictionI(display *DisplayTemplate, result float64) {
	actualLoad := findMeanLoadOfTemplate(DeviceManager.allDisplays, display.dataTypeName, display.direction)
	state := getStateFromPredictedAndActualValue(result, actualLoad, DeviceManager.designator,
		DeviceManager.linkBandwidth)
	line1, line2 := getPredictionLines(DeviceManager.smoothingRange, display, result, state)
	DeviceManager.WriteMessageOnLcd(line1, line2)
	DeviceManager.updateLcdDisplay(result)
}

// Parsing of state string from predicted load, actual load, and designator fraction.
// Parameter predictedValue float64 - predicted load.
// Parameter actualValue float64 - actual load.
// Parameter designator float64 - fraction of bandwidth that defines range in which actual load must be placed
// to still state.
// Parameter bandwidth uint64 - maximum link load [bytes/sec].
func getStateFromPredictedAndActualValue(predictedValue float64, actualValue float64, designator float64,
	bandwidth uint64) string {
	lowerLimit := actualValue - float64(actualValue)*designator
	if lowerLimit < 0.0 {
		lowerLimit = 0.0
	}
	upperLimit := actualValue + float64(actualValue)*designator
	if upperLimit > float64(bandwidth) {
		upperLimit = float64(bandwidth)
	}
	if predictedValue >= lowerLimit && predictedValue <= upperLimit {
		return "S"
	} else if predictedValue < lowerLimit {
		return "D"
	} else {
		return "R"
	}
}

// Finding of the mean load of adjacent display that is used for presentation of actual load.
// Parameter allDisplays *map[DisplayTemplate]float64 - all displays.
// Parameter dataTypeName string - name of the data type that is searched.
// Prameter direction uint - direction of load measurement.
// Returning float64 - found actual mean load or 0.0 if it is not found within input constraints.
func findMeanLoadOfTemplate(allDisplays *map[DisplayTemplate]float64, dataTypeName string, direction uint) float64 {
	for display := range *allDisplays {
		if display.dataTypeName == dataTypeName && display.direction == direction && display.prediction == false {
			meanLoad := (*allDisplays)[display]
			return meanLoad
		}
	}
	return 0.0
}

// Parsing of display template and computed load to string lines of LCD.
// Parameter smoothingRange uint - smoothing range used by SmoothingCreator.
// Parameter display *DisplayTemplate - modified display.
// Parameter result float64 - computed average for specific display.
// Returning line1 string - first LCD line.
// Returning line2 string - second LCD line.
func getMeanLines(smoothingRange uint, display *DisplayTemplate, result float64) (line1 string, line2 string) {
	var direction string
	if display.direction == 0 {
		direction = DIRECTION_RX
	} else {
		direction = DIRECTION_TX
	}
	truncatedName := display.dataTypeName
	if uint(len(truncatedName)) > LINE_LENGTH - 3 {
		truncatedName = truncatedName[:LINE_LENGTH-3]
	}
	var formattedResult string
	var unit string
	if result < CONVERSION_TRESHOLD_B_KB {
		formattedResult = fmt.Sprint(uint64(result))
		unit = B_UNIT
	} else if result < CONVERSION_TRESHOLD_KB_MB {
		formattedResult = fmt.Sprint(uint64(result/ CONVERSION_RATIO_KB))
		unit = KB_UNIT
	} else {
		formattedResult = fmt.Sprint(uint64(result/ CONVERSION_RATIO_MB))
		unit = MB_UNIT
	}
	var rate string
	if smoothingRange == BASIC_RATE {
		rate = "/s"
	} else {
		rate = "/" + string(smoothingRange/BASIC_RATE)
	}
	line1Out := direction + " " + truncatedName
	line2Out := formattedResult + " " + unit + rate
	return line1Out, line2Out
}

// Parsing of display template and computed prediction to string lines of LCD.
// Parameter smoothingRange uint - smoothing range used by SmoothingCreator.
// Parameter display *DisplayTemplate - modified display.
// Parameter result float64 - computed prediction for specific display.
// Parameter state string - R (load raises) / D (load drops) / S (load is still).
// Returning line1 string - first LCD line.
// Returning line2 string - second LCD line.
func getPredictionLines(smoothingRange uint, display *DisplayTemplate, result float64, state string) (
	line1 string, line2 string) {
	var direction string
	if display.direction == 0 {
		direction = DIRECTION_RX
	} else {
		direction = DIRECTION_TX
	}
	truncatedName := display.dataTypeName
	if uint(len(truncatedName)) > LINE_LENGTH - 5 {
		truncatedName = truncatedName[:LINE_LENGTH-5]
	}
	var formattedResult string
	var unit string
	if result < CONVERSION_TRESHOLD_B_KB {
		formattedResult = fmt.Sprint(uint64(result))
		unit = B_UNIT
	} else if result < CONVERSION_TRESHOLD_KB_MB {
		formattedResult = fmt.Sprint(uint64(result/ CONVERSION_RATIO_KB))
		unit = KB_UNIT
	} else {
		formattedResult = fmt.Sprint(uint64(result/ CONVERSION_RATIO_MB))
		unit = MB_UNIT
	}
	var rate string
	if smoothingRange == BASIC_RATE {
		rate = "/s"
	} else {
		rate = "/" + string(smoothingRange/BASIC_RATE)
	}
	line1Out := direction + " " + truncatedName + " " + state
	line2Out := formattedResult + " " + unit + rate
	return line1Out, line2Out
}

// Handling of data type removal event.
// Parameter dataTypeId uint - data type ID.
func (DeviceManager *DeviceManager) RemoveDataType(dataTypeId uint) {
	DeviceManager.displayMutex.Lock()
	defer DeviceManager.displayMutex.Unlock()
	var foundDisplays []DisplayTemplate
	for display := range *DeviceManager.allDisplays {
		if display.dataTypeId == dataTypeId {
			foundDisplays = append(foundDisplays, display)
		}
	}
	DeviceManager.deleteDisplays(&foundDisplays)
}

// Handling of turned off prediction.
// Parameter dataTypeId uint - data type ID.
func (DeviceManager *DeviceManager) TurnOffPrediction(dataTypeId uint) {
	DeviceManager.displayMutex.Lock()
	defer DeviceManager.displayMutex.Unlock()
	var foundDisplays []DisplayTemplate
	for display := range *DeviceManager.allDisplays {
		if display.dataTypeId == dataTypeId && display.prediction {
			foundDisplays = append(foundDisplays, display)
		}
	}
	DeviceManager.deleteDisplays(&foundDisplays)
}

// Removal of selected display templates.
// Parameter foundDisplays *[](*DisplayTemplate) - displays that are going to be removed. See DisplayTemplate.
func (DeviceManager *DeviceManager) deleteDisplays(foundDisplays *[]DisplayTemplate) {
	for _, display := range *foundDisplays {
		if DeviceManager.actualDisplay != nil && *DeviceManager.actualDisplay == display {
			DeviceManager.recoverFromRemovedDisplay()
		}
		delete(*DeviceManager.allDisplays, display)
	}
}

// Refreshing of LCD content after data type removal.
func (DeviceManager *DeviceManager) recoverFromRemovedDisplay() {
	sortedDisplays := getSortedDisplays(DeviceManager.allDisplays)
	index := getIndexOfActualDataType(sortedDisplays, DeviceManager.actualDisplay)
	if index != uint(len(*sortedDisplays) - 1) {
		DeviceManager.setNextDisplay(sortedDisplays, index)
	} else if index != 0 {
		DeviceManager.setPreviousDisplay(sortedDisplays, index)
	} else {
		DeviceManager.WriteMessageOnLcd(BOOT_FIRST_LINE, BOOT_SECOND_LINE)
	}
}

// Displaying of next display content on LCD.
// Parameter sortedDisplays *[]DisplayTemplate - sorted list of displays. See DisplayTemplate.
// Parameter index uint - index of actual display (sortedDisplays slice).
func (DeviceManager *DeviceManager) setNextDisplay(sortedDisplays *[]DisplayTemplate, index uint) {
	nextDisplay := (*sortedDisplays)[index + 1]
	if !nextDisplay.prediction {
		nextValue := (*DeviceManager.allDisplays)[nextDisplay]
		line1, line2 := getMeanLines(DeviceManager.smoothingRange, &nextDisplay, nextValue)
		DeviceManager.WriteMessageOnLcd(line1, line2)
		DeviceManager.actualDisplay = &nextDisplay
		DeviceManager.updateLcdDisplay(nextValue)
	} else {
		nextValue := (*DeviceManager.allDisplays)[nextDisplay]
		actualLoad := findMeanLoadOfTemplate(DeviceManager.allDisplays, nextDisplay.dataTypeName, nextDisplay.direction)
		state := getStateFromPredictedAndActualValue(nextValue, actualLoad, DeviceManager.designator,
			DeviceManager.linkBandwidth)
		line1, line2 := getPredictionLines(DeviceManager.smoothingRange, &nextDisplay, nextValue, state)
		DeviceManager.WriteMessageOnLcd(line1, line2)
		DeviceManager.updateLcdDisplay(nextValue)
	}
	DeviceManager.actualDisplay = &nextDisplay
}

// Displaying of previous display content on LCD.
// Parameter sortedDisplays *[]DisplayTemplate - sorted list of displays. See DisplayTemplate.
// Parameter index uint - index of actual display (sortedDisplays slice).
func (DeviceManager *DeviceManager) setPreviousDisplay(sortedDisplays *[]DisplayTemplate, index uint) {
	previousDisplay := (*sortedDisplays)[index - 1]
	if !previousDisplay.prediction {
		previousValue := (*DeviceManager.allDisplays)[previousDisplay]
		line1, line2 := getMeanLines(DeviceManager.smoothingRange, &previousDisplay, previousValue)
		DeviceManager.WriteMessageOnLcd(line1, line2)
		DeviceManager.updateLcdDisplay(previousValue)
	} else {
		previousValue := (*DeviceManager.allDisplays)[previousDisplay]
		actualLoad := findMeanLoadOfTemplate(DeviceManager.allDisplays, previousDisplay.dataTypeName,
			previousDisplay.direction)
		state := getStateFromPredictedAndActualValue(previousValue, actualLoad, DeviceManager.designator,
			DeviceManager.linkBandwidth)
		line1, line2 := getPredictionLines(DeviceManager.smoothingRange, &previousDisplay, previousValue, state)
		DeviceManager.WriteMessageOnLcd(line1, line2)
		DeviceManager.updateLcdDisplay(previousValue)
	}
	DeviceManager.actualDisplay = &previousDisplay
}

// Modification of data type name.
// Parameter dataTypeId uint - unique ID of modified data type.
// Parameter dataTypeName string - new data type name.
func (DeviceManager *DeviceManager) ModifyDataTypeName(dataTypeId uint, dataTypeName string) {
	DeviceManager.displayMutex.Lock()
	defer DeviceManager.displayMutex.Unlock()
	displaysToRemove := make([]DisplayTemplate, 0)
	displaysToAdd := make([]DisplayTemplate, 0)
	valuesToAdd := make([]float64, 0)
	displaysMap := *DeviceManager.allDisplays
	for display := range displaysMap {
		if display.dataTypeId == dataTypeId {
			updatedDisplay := DisplayTemplate{
				dataTypeName: dataTypeName,
				dataTypeId: display.dataTypeId,
				prediction: display.prediction,
				direction: display.direction,
			}
			displaysToAdd = append(displaysToAdd, updatedDisplay)
			valuesToAdd = append(valuesToAdd, (*DeviceManager.allDisplays)[display])
			displaysToRemove = append(displaysToRemove, display)
		}
	}
	for _, display := range displaysToRemove {
		delete(*DeviceManager.allDisplays, display)
	}
	for i := 0; i < len(valuesToAdd); i++ {
		(*DeviceManager.allDisplays)[displaysToAdd[i]] = valuesToAdd[i]
	}
	if DeviceManager.actualDisplay.dataTypeId == dataTypeId {
		DeviceManager.actualDisplay.dataTypeName = dataTypeName
	}
}

// Updating of LED strip.
// Parameter refreshedValue float64 - new value that is going to be displayed.
func (DeviceManager *DeviceManager) updateLcdDisplay(refreshedValue float64) {
	coefficient := float64(SPACE_MAX)/float64(DeviceManager.linkBandwidth)
	k := uint16(coefficient*refreshedValue)
	rgbSpace, err := NewRgbSpace(k)
	if err == nil {
		rc, gc, bc := rgbSpace.ColorComponents()
		DeviceManager.FlashLedStrip(uint(rc), uint(gc), uint(bc))
	} else {
		configuration.Error.Panicf("LED strip cannot be updated: %v", err)
	}
}

// Flashing of LED strip according to configured values of RGB components; interval <0, 255>.
// Parameter redComponent uint - red component of RGB code.
// Parameter greenComponent uint - green component of RGB code.
// Parameter blueComponent uint - blue component of RGB code.
func (DeviceManager *DeviceManager) FlashLedStrip(redComponent, greenComponent, blueComponent uint) {
	DeviceManager.ledMutex.Lock()
	defer DeviceManager.ledMutex.Unlock()
	err := exec.Command(
		"python",
		"led_strip.py",
		fmt.Sprint(DeviceManager.configData.BCM_LED_Strip),
		fmt.Sprint(DeviceManager.configData.LEDsCount),
		fmt.Sprint(DeviceManager.configData.LEDsBrightness),
		fmt.Sprint(redComponent),
		fmt.Sprint(greenComponent),
		fmt.Sprint(blueComponent),
	).Run()
	if err != nil {
		message := fmt.Sprintf("%v", err)
		if message != "signal: segmentation fault" {
			configuration.Error.Panicf("An error occurred during setting of colors on LED strip: %v", err)
		}
	}
}

// Closing of robot instances.
func (DeviceManager *DeviceManager) CloseDeviceManager() {
	configuration.Info.Println("Stopping of device manager.")
	if DeviceManager.robot != nil {
		DeviceManager.robot.Stop()
	}
	configuration.Info.Println("Device manager has been stopped.")
}