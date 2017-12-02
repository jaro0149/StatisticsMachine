import RPi.GPIO as GPIO
import time
import sys

GPIO.setmode(GPIO.BOARD)
GPIO.setup(int(sys.argv[1]), GPIO.IN, pull_up_down=GPIO.PUD_UP)
GPIO.setup(int(sys.argv[2]), GPIO.IN, pull_up_down=GPIO.PUD_UP)
