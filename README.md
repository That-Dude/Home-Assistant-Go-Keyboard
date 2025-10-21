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
I think this is a pretty niche use case, but if you want physical buttons and you have a RPi already it works well.

Pros:
- wireless dongle has a range of at least 20 meters
- battery life using the 1xAAA cell is measured in years
- super inexpensive
- Coded in Go, single tiny binary with no dependencies. Uses less that 1% CPU on a RPi1 (Tested on ARM and x86 linux)
- Home Assistant auto-discovery
- Utility attaches to a selected specific HID device, so it doesn't affect any other attached keyboards or mice
- You could connect lots of these to a single Pi in theory (untested)
- Runs in Docker

Cons:
- Keycaps are not replaceable (I plan to upgrade ,I'd prefer clear keycaps so I can print icons)
- You need a device to plug the receiver into that can run the utility in the background

## Installation

On your linux device eg Raspberry Pi, choose somewhere to install the program:

```bash
cd ~/
git clone https://github.com/That-Dude/gokeyboard
cd gokeyboard
```
plug in your number pad (or wireless keyboard) and get the device name:
```bash
sudo cat /proc/bus/input/devices | grep Name=

N: Name="vc4-hdmi"
N: Name="vc4-hdmi HDMI Jack"
N: Name="YICHIP 2.4G Receiver"
N: Name="YICHIP 2.4G Receiver Mouse"
N: Name="YICHIP 2.4G Receiver System Control"
N: Name="YICHIP 2.4G Receiver Consumer Control"
```
In this case my device is called `YICHIP 2.4G Receiver`, make a note of this or copy it to the clipboard.

```bash
mv config.yaml.example config.yaml
nano config.yaml
```

Update your MQTT details and add the device detected above to the 'keyboard_name' line:

```
mqtt:
  broker: "tcp://homeassistant.local:1883"
  username: "mqttuser"
  password: "mqttpassword"
  device_id: "go_keyboard_numpad"

input:
  keyboard_name: "YICHIP 2.4G Receiver"

timing:
  double_press_ms: 250
  long_press_ms: 500
```
run the utility:
```bash
./gokeyboardsudo apt install evtest
```

And that's it, now when you press button on the keyboard it will show up in Home Assistant as a new Device with binary sensors for single, double and long presses.

# Docker
Ensure you have go installed.

```bash
docker build -t gokeyboard .
```

```bash
docker compose up
```
Ensure everything is working and asuming it is:

```bash
docker compose up -d
```
To get the correct /dev/input/event ID for the docker compose file you could use evtest ,it was `/dev/input/event2` in my case:
```bash
sudo apt install evtest
sudo evtest

/dev/input/event0:	vc4-hdmi
/dev/input/event1:	vc4-hdmi HDMI Jack
/dev/input/event2:	YICHIP 2.4G Receiver
/dev/input/event3:	YICHIP 2.4G Receiver Mouse
/dev/input/event4:	YICHIP 2.4G Receiver System Control
/dev/input/event5:	YICHIP 2.4G Receiver Consumer Control
```
