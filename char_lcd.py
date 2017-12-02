#!/usr/bin/python
import sys
import Adafruit_CharLCD as LCD

# Raspberry Pi pin configuration:
lcd_rs        = int(sys.argv[1])
lcd_en        = int(sys.argv[2]) 
lcd_d4        = int(sys.argv[3])
lcd_d5        = int(sys.argv[4])
lcd_d6        = int(sys.argv[5])
lcd_d7        = int(sys.argv[6])
lcd_backlight = int(sys.argv[7])

# Define LCD column and row size for 16x2 LCD.
lcd_columns = 16
lcd_rows    = 2

# Initialize the LCD using the pins above.
lcd = LCD.Adafruit_CharLCD(lcd_rs, lcd_en, lcd_d4, lcd_d5, lcd_d6, lcd_d7,
                           lcd_columns, lcd_rows, lcd_backlight)

# Print a two line message
lcd.message(sys.argv[8] + '\n' + sys.argv[9])
