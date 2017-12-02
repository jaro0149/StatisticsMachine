#!/usr/bin/python
import sys
import time
from neopixel import *

# LED strip configuration constants
LED_FREQ_HZ		= 800000				# LED signal frequency in hertz (usually 800khz)
LED_DMA			= 5					# DMA channel to use for generating signal (try 5)
LED_INVERT		= False					# true to invert the signal (when using NPN transistor level shift)
LED_CHANNEL		= 0					# set to '1' for GPIOs 13, 19, 41, 45 or 53
LED_STRIP		= ws.WS2811_STRIP_RGB   		# strip type and colour ordering
WAIT_MS			= 10 					# timeout between LED flashes

ledPin = int(sys.argv[1])			# BCM pin number
ledCount = int(sys.argv[2])			# number of leds in the strip
ledBrightness = int(sys.argv[3])        	# set to 0 for darkest and 255 for brightest
redComponent = int(sys.argv[4])			# red component value in the range of <0, 255>
greenComponent = int(sys.argv[5])		# green component value in the range of <0, 255>
blueComponent = int(sys.argv[6])		# blue component value in the range of <0, 255>

# Color definition
displayedColor = Color(redComponent, greenComponent, blueComponent)

# LED strip initialisation
strip = Adafruit_NeoPixel(ledCount, ledPin, LED_FREQ_HZ, LED_DMA, LED_INVERT, ledBrightness, LED_CHANNEL, LED_STRIP)
strip.begin()

# Sequential flashing of LEDs in strip
for i in range(strip.numPixels()):
	strip.setPixelColor(i, displayedColor)
	# strip.show()
	time.sleep(WAIT_MS/1000.0)
strip.show()
