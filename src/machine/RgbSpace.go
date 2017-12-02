package machine

import (
	"fmt"
	"configuration"
)

const RGB_MIN = uint8(0)
const RGB_MAX = uint8(255)
const SPACE_MIN = uint16(0)
const SPACE_MAX = uint16(1020)
const TRESHOLD_INC_B_STILL_R = uint16(255)
const TRESHOLD_DSC_R_STILL_B = uint16(510)
const TRESHOLD_INC_G_STILL_R = uint16(765)

// Attribute rgbValue uint16 - Actual value of RGB space.
type RgbSpace struct {
	rgbValue uint16
}

// Building of RGB space from integer value that must be in the interval <0, 1020>.
// Parameter rgbValue uint16 - integer value that must be in the interval <0, 1020>.
// Returning *RgbSpace - created instance of RgbSpace struct.
// Returning error - error message.
func NewRgbSpace(rgbValue uint16) (*RgbSpace, error) {
	if rgbValue >= SPACE_MIN && rgbValue <= SPACE_MAX {
		rgbSpace := RgbSpace{
			rgbValue: rgbValue,
		}
		return &rgbSpace, nil
	} else {
		compositeError := configuration.NewCompositeError()
		compositeError.AddError(1, fmt.Sprintf("Specified color value is out of range <0, 1020>: %d",
			rgbValue))
		return nil, compositeError.Evaluate()
	}
}

// Getting of invidual RGB components from RGB space.
// Returning redComponent uint8 - red component <0,255>.
// Returning greenComponent uint8 - green component <0,255>.
// Returning blueComponent uint8 - blue component <0,255>.
func (RgbSpace *RgbSpace) ColorComponents() (redComponent, greenComponent, blueComponent uint8) {
	if RgbSpace.rgbValue <= TRESHOLD_INC_B_STILL_R {
		red := RGB_MAX
		green := RGB_MIN
		blue := uint8(RgbSpace.rgbValue)
		return red, green, blue
	} else if RgbSpace.rgbValue <= TRESHOLD_DSC_R_STILL_B {
		red := uint8(TRESHOLD_DSC_R_STILL_B - RgbSpace.rgbValue)
		green := RGB_MIN
		blue := RGB_MAX
		return red, green, blue
	} else if RgbSpace.rgbValue <= TRESHOLD_INC_G_STILL_R {
		red := RGB_MIN
		green := uint8(RgbSpace.rgbValue -  TRESHOLD_DSC_R_STILL_B)
		blue := RGB_MAX
		return red, green, blue
	} else {
		red := RGB_MIN
		green := RGB_MAX
		blue := uint8(SPACE_MAX - RgbSpace.rgbValue)
		return red, green, blue
	}
}