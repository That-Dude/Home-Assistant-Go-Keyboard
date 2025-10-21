# Home Assistant Go-Keyboard
Use a regular wireless Numpad to control Home Assistant (this is a bit niche)

# Overview

I have Home Assistant setup in my house to control pretty much everything.

I have tablets mounted on the wall with a beautiful clear theme that allows for lots of interactive control.

The problem is, my wife and kids hate using them!

After some behavioral testing I discovered that they REALLY like pressing physical buttons to activate stuff. For my testing I used a cheap 2.4Ghz wireless Numpad from Amazon, it was very inexpensive at £5.20 delivered.

## Cheap Numpad
<img width="612" height="618" alt="numpad" src="https://github.com/user-attachments/assets/2792109d-edd0-4e15-b272-3e7aa8c44d1d" />

I already had a Raspberry Pi in the house acting as a Zigbee gateway, so I connected the Numpad's wireless USB receiver to the Pi and coded a simple utility to intercept the Numpad button presses and forward them to Home Assistant via MQTT.

This works far better than I expected and it immediately gained approval from the family, so I extended the utility to add some additional features such as double-click and long-press detection.

## Home Assistant device
<img width="890" height="967" alt="Go-Keyboard" src="https://github.com/user-attachments/assets/e5df129e-48aa-4374-9d35-46b01bfa8640" />

## Why do this?
I think this is a pretty niche use case, but if you want physical button and you have a RPi already it works very well..

Pros:
- wireless dongle has a range of at least 20 meters
- battery life using the 1xAAA cell is measured in years
- super inexpensive
- Coded in Go, single tiny binary with no dependencies. Uses less that 1% CPU on a RPi1 (Tested on ARM and x86 linux)
- Home Assistant auto-discovery
- Utility attaches to a selected specific HID device, so it doesn't affect any other attached keyboards or mice
- You could connect lots of these to a single Pi in theory (untested)

Cons:
- Keycaps are not replaceable (I plan to upgrade ,I'd prefer clear keycaps so I can print icons)
- You need a device to plug the receiver into that can run the utility in the background

Todo:
- Write Linux service configuration to auto-start on boot