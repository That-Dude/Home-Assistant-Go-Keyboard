package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/MarinX/keylogger"
	"gopkg.in/yaml.v3"
)

type Config struct {
	MQTT struct {
		Broker   string `yaml:"broker"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		DeviceID string `yaml:"device_id"`
	} `yaml:"mqtt"`
	Input struct {
		KeyboardName string `yaml:"keyboard_name"`
	} `yaml:"input"`
	Timing struct {
		DoublePressMS int `yaml:"double_press_ms"`
		LongPressMS   int `yaml:"long_press_ms"`
	} `yaml:"timing"`
}

func main() {
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	// Default fallbacks
	if cfg.Timing.DoublePressMS == 0 {
		cfg.Timing.DoublePressMS = 250
	}
	if cfg.Timing.LongPressMS == 0 {
		cfg.Timing.LongPressMS = 500
	}

	// MQTT setup
	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.MQTT.Broker)
	opts.SetUsername(cfg.MQTT.Username)
	opts.SetPassword(cfg.MQTT.Password)
	opts.SetClientID("go-keyboard-realtime")

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("❌ MQTT connection failed: %v", token.Error())
	}
	defer client.Disconnect(250)
	fmt.Println("✅ Connected to MQTT broker")

	// Keyboard device
	devicePath, err := findDeviceByName(cfg.Input.KeyboardName)
	if err != nil {
		fmt.Printf("⚠️ Could not find '%s', falling back to first keyboard...\n", cfg.Input.KeyboardName)
		devicePath = keylogger.FindKeyboardDevice()
		if devicePath == "" {
			log.Fatalf("❌ No keyboard found")
		}
	}
	fmt.Printf("🎹 Using keyboard device: %s\n", devicePath)

	k, err := keylogger.New(devicePath)
	if err != nil {
		log.Fatalf("❌ Failed to open keyboard device: %v", err)
	}
	defer k.Close()

	events := k.Read()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	fmt.Println("🎯 Ready — press any key. Press ESC to quit.\n")

	registeredKeys := map[string]bool{}
	currentKeys := map[string]bool{}

	doublePressThreshold := time.Duration(cfg.Timing.DoublePressMS) * time.Millisecond
	longPressThreshold := time.Duration(cfg.Timing.LongPressMS) * time.Millisecond

	// --- Event handling ---
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("⚠️ Recovered from panic in event loop: %v", r)
			}
		}()

		lastPressTime := make(map[string]time.Time)
		pressStart := make(map[string]time.Time)
		isHeld := make(map[string]bool)
		lastWasDouble := make(map[string]bool)

		for e := range events {
			if e.Type != keylogger.EvKey {
				continue
			}

			keyStr := e.KeyString()
			safeKey := sanitizeKeyName(keyStr)

			stateTopic := fmt.Sprintf("homeassistant/binary_sensor/%s_%s/state",
				cfg.MQTT.DeviceID, strings.ToLower(safeKey))
			doubleTopic := fmt.Sprintf("homeassistant/binary_sensor/%s_%s_double/state",
				cfg.MQTT.DeviceID, strings.ToLower(safeKey))
			longTopic := fmt.Sprintf("homeassistant/binary_sensor/%s_%s_long/state",
				cfg.MQTT.DeviceID, strings.ToLower(safeKey))

			// Register sensors once
			if !registeredKeys[keyStr] {
				register := func(name, topic, suffix string) {
					discovery := fmt.Sprintf("homeassistant/binary_sensor/%s_%s/config",
						cfg.MQTT.DeviceID, strings.ToLower(safeKey+suffix))
					payload := fmt.Sprintf(`{
						"name": "%s",
						"state_topic": "%s",
						"payload_on": "ON",
						"payload_off": "OFF",
						"unique_id": "%s_%s%s",
						"device": {
							"identifiers": ["%s"],
							"name": "Go Keyboard",
							"manufacturer": "GoLang",
							"model": "Realtime Keyboard"
						}
					}`, name, topic, cfg.MQTT.DeviceID, safeKey, suffix, cfg.MQTT.DeviceID)
					client.Publish(discovery, 1, true, payload)
				}
				register(keyStr, stateTopic, "")
				register(keyStr+" (Double Press)", doubleTopic, "_double")
				register(keyStr+" (Long Press)", longTopic, "_long")
				registeredKeys[keyStr] = true
			}

			if e.KeyPress() {
				now := time.Now()
				isHeld[keyStr] = true
				pressStart[keyStr] = now

				// --- DOUBLE PRESS ---
				if prev, ok := lastPressTime[keyStr]; ok && now.Sub(prev) < doublePressThreshold {
					fmt.Printf("🎯 Double press: %s\n", keyStr)
					lastWasDouble[keyStr] = true
					client.Publish(doubleTopic, 0, false, "ON")
					time.AfterFunc(200*time.Millisecond, func() {
						client.Publish(doubleTopic, 0, false, "OFF")
					})
					lastPressTime[keyStr] = time.Time{}
					continue
				}

				lastPressTime[keyStr] = now
				lastWasDouble[keyStr] = false

				// --- LONG PRESS DETECTION ---
				go func(k string, start time.Time) {
					time.Sleep(longPressThreshold)
					if isHeld[k] && pressStart[k] == start {
						fmt.Printf("⏱️ Long press: %s\n", k)
						client.Publish(longTopic, 0, false, "ON")
						time.AfterFunc(200*time.Millisecond, func() {
							client.Publish(longTopic, 0, false, "OFF")
						})
						lastWasDouble[k] = true // cancel single
					}
				}(keyStr, now)

			} else if e.KeyRelease() {
				isHeld[keyStr] = false
				releaseTime := time.Now()
				duration := releaseTime.Sub(pressStart[keyStr])
				thisPress := pressStart[keyStr]

				// Delay single press to confirm no double press follows
				go func(k string, pressTime time.Time, heldDuration time.Duration) {
					time.Sleep(doublePressThreshold)
					if !lastWasDouble[k] && heldDuration < longPressThreshold {
						fmt.Printf("🖐️ Single press: %s\n", k)
						client.Publish(stateTopic, 0, false, "ON")
						time.AfterFunc(150*time.Millisecond, func() {
							client.Publish(stateTopic, 0, false, "OFF")
						})
					}
				}(keyStr, thisPress, duration)
			}
		}
	}()

	// --- Live dashboard ---
	go func() {
		for {
			fmt.Print("\033[H\033[2J") // clear screen
			fmt.Println("=== Go Keyboard Dashboard (Real-Time) ===")
			fmt.Println("Keys Currently Held:")
			if len(currentKeys) == 0 {
				fmt.Println("  (none)")
			} else {
				for k := range currentKeys {
					fmt.Printf("  %s\n", k)
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	<-sigChan
	fmt.Println("🛑 Exiting program.")
}

// --- Utility functions ---

func loadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cfg Config
	err = yaml.NewDecoder(f).Decode(&cfg)
	return &cfg, err
}

func findDeviceByName(name string) (string, error) {
	f, err := os.Open("/proc/bus/input/devices")
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "N: Name=") && strings.Contains(line, name) {
			for scanner.Scan() {
				line = scanner.Text()
				if strings.Contains(line, "Handlers=") && strings.Contains(line, "event") {
					fields := strings.Fields(line)
					for _, f := range fields {
						if strings.HasPrefix(f, "event") {
							return "/dev/input/" + f, nil
						}
					}
				}
			}
		}
	}
	return "", fmt.Errorf("device '%s' not found", name)
}

// --- Helper for safe MQTT topic naming ---
func sanitizeKeyName(k string) string {
	k = strings.ToLower(strings.TrimSpace(k))
	k = strings.ReplaceAll(k, " ", "_")
	k = strings.ReplaceAll(k, "+", "plus")
	k = strings.ReplaceAll(k, "-", "minus")
	k = strings.ReplaceAll(k, "*", "asterisk")
	k = strings.ReplaceAll(k, "/", "slash")
	k = strings.ReplaceAll(k, "\\", "backslash")
	k = strings.ReplaceAll(k, ".", "dot")
	k = strings.ReplaceAll(k, ",", "comma")
	k = strings.ReplaceAll(k, "=", "equals")
	k = strings.ReplaceAll(k, "'", "quote")
	k = strings.ReplaceAll(k, "[", "lbracket")
	k = strings.ReplaceAll(k, "]", "rbracket")
	return k
}
