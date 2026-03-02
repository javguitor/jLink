package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

var (
	loading      = [10]string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	loadingIndex = 0

	// Box Drawing
	horizontalLine = "" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" // 240 total and 3 Bytes per ━

	verticalLine      = "┃"
	leftCornerTop     = "┏"
	rightCornerTop    = "┓"
	leftCornerBottom  = "┗"
	rightCornerBottom = "┛"

	// screens size
	width, height = 0, 0

	// For Navigation
	resetCurrentSelection = false
	currentSelection      = 0
	menuState             = 0
	startMenuSelected     = -1

	// selecet
	selectedItemsPairedDevices = -1
	menuItemsPairedDevices     = [5]string{"Q Back", "1 Connect", "2 Disconnect", "3 Remove", "4 Clear"}

	selectedItemsSearchForNewDevices = -1
	menuItemsSearchForNewDevices     = [2]string{"Q Back", "1 Connect"}

	// Headset settings & equalizer
	equalizerBands  []equalizerBand
	deviceInfoLines []string
)

const (
	batteryFullChar     = "◼"
	batteryEmptyChar    = "◻"
	batteryWidth        = 10
	lowBatteryThreshold = 20
)

func enableRawMode() (*unix.Termios, error) {

	fd := int(os.Stdin.Fd())

	// Get the current terminal settings
	oldSettings, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, err
	}

	newSettings := *oldSettings
	newSettings.Lflag &^= unix.ECHO | unix.ICANON // Disable echo and canonical mode
	newSettings.Iflag &^= unix.ICRNL              // Disable carriage return/newline conversion

	// Apply the new terminal settings
	if err := unix.IoctlSetTermios(fd, unix.TCSETS, &newSettings); err != nil {
		return nil, err
	}

	return oldSettings, nil
}

func restoreTerminal(oldSettings *unix.Termios) {
	unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TCSETS, oldSettings)
}

func startKeysPressedListener() {
	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading input:", err)
			continue
		}

		// Handle arrow keys (escape sequences)
		if n >= 3 && buf[0] == 0x1B && buf[1] == '[' {
			switch buf[2] {
			case 'A': // Up Arrow
				handleUpKey()
			case 'B': // Down Arrow
				handleDownKey()
			}
			continue
		}

		// Handle single-byte input (e.g., 'w', 's', 'q', etc.)
		key := buf[0]
		switch menuState {
		case 0: // StartMenu
			switch key {
			case 'w': // Up
				handleUpKey()
			case 's': // Down
				handleDownKey()
			case '\r': // Enter
				startMenuSelected = currentSelection
			}
		// ############## Search For New Devices #################
		case 1:
			switch key {
			case 'q': // Back To Start Menu
				if err = setDongleInBTPairing(false); err != nil {
					fmt.Println(err) //  remember to add a error window in the ui
				}
				startMenuSelected = -1
			case '1':
				selectedItemsSearchForNewDevices = 1
				if len(searchDeviceList.pairedDevices) != 0 {
					if err := connectNewDevice(uint16(currentSelection)); err != nil {
						fmt.Println(err) //  remember to add a error window in the ui
					} else {
						startMenuSelected = -1
					}
				}
			}
		// ############## See Remembered Paired Devices #################
		case 2:
			switch key {
			case 'q': // Back To Start Menu
				startMenuSelected = -1
			case 'w': // Up
				handleUpKey()
			case 's': // Down
				handleDownKey()
			case '1':
				if err := connectDeviceFromPairedlist(uint16(currentSelection)); err != nil {
					fmt.Println(err) //  remember to add a error window in the ui
				}
				selectedItemsPairedDevices = 1
			case '2':
				if err := disconnectDeviceFromPairedlist(uint16(currentSelection)); err != nil {
					fmt.Println(err) //  remember to add a error window in the ui
				}
				selectedItemsPairedDevices = 2
			case '3':
				if err := removeDeviceFromPairedlist(uint16(currentSelection)); err != nil {
					fmt.Println(err) //  remember to add a error window in the ui
				}
				selectedItemsPairedDevices = 3
			case '4':
				if err := clearPairingList(); err != nil {
					fmt.Println(err) //  remember to add a error window in the ui
				}
				selectedItemsPairedDevices = 4
			}
		// ############# Dongle Settings ##################
		case 3:
			switch key {
			case 'q': // Back To Start Menu
				startMenuSelected = -1
			case 'w': // Up
				handleUpKey()
			case 's': // Down
				handleDownKey()
			case '\r': // Enter
				switch dongleSettignsMenu[currentSelection].id {
				case 0:
					getautoPairingState, _ := getAutoPairing()
					if err := setAutoPairing(!getautoPairingState); err != nil {
						fmt.Println(err) //  remember to add a error window in the ui
					}
					updateDongleSettignsMenu()
				case 1:
					if dongle, exists := deviceManager[selectedDongle]; exists {
						if err := factoryReset(dongle.deviceID); err != nil {
							fmt.Println(err) //  remember to add a error window in the ui
						}
						startMenuSelected = -1
					}
				}
			}
		// ############# switch  device ##################
		case 4:
			switch key {
			case 'q': // Back To Start Menu
				startMenuSelected = -1
			}
		// ############# Headset Settings ##################
		case 5:
			switch key {
			case 'q':
				startMenuSelected = -1
			case 'w':
				handleUpKey()
			case 's':
				handleDownKey()
			case '\r':
				if len(headsetSettingsMenu) > 0 {
					switch headsetSettingsMenu[currentSelection].id {
					case 0: // ANC Mode -> open ANC settings sub-screen
						menuState = 9
						resetCurrentSelection = false
					case 1: // Equalizer
						menuState = 7
						resetCurrentSelection = false
					case 2: // Sidetone toggle
						if device, exists := deviceManager[selectedHeadset]; exists {
							sidetone := findDeviceSetting(device.deviceID, "sidetone")
							if sidetone != nil && len(sidetone.options) > 0 {
								nextKey := (sidetone.current + 1) % len(sidetone.options)
								setDeviceSetting(device.deviceID, sidetone.guid, nextKey)
								updateHeadsetSettingsMenu()
							}
						}
					case 3: // Busy Light toggle
						if device, exists := deviceManager[selectedHeadset]; exists {
							current := getBusyLightStatus(device.deviceID, device.featureFlags)
							setBusyLightStatus(device.deviceID, !current, device.featureFlags)
							updateHeadsetSettingsMenu()
						}
					}
				}
			}
		// ############# Device Info ##################
		case 6:
			switch key {
			case 'q':
				startMenuSelected = -1
			}
		// ############# Audio Settings ##################
		case 8:
			switch key {
			case 'q':
				startMenuSelected = -1
			case 'w':
				handleUpKey()
			case 's':
				handleDownKey()
			case '\r': // Enter on profile
				if currentSelection == 0 && currentAudioState != nil && len(currentAudioState.profiles) > 0 {
					nextIdx := -1
					for i, p := range currentAudioState.profiles {
						if p.index == currentAudioState.activeProfile {
							nextIdx = (i + 1) % len(currentAudioState.profiles)
							break
						}
					}
					if nextIdx < 0 {
						nextIdx = 0
					}
					setAudioProfile(currentAudioState.deviceID, currentAudioState.profiles[nextIdx].index)
					// Re-discover after profile change (sink/source IDs may change)
					currentAudioState = discoverPipeWireDevice()
					refreshAudioState()
				}
			case 'a': // Decrease volume
				if currentAudioState != nil {
					switch currentSelection {
					case 1:
						if currentAudioState.sinkID >= 0 {
							newVol := currentAudioState.outputVolume - 0.05
							if newVol < 0 {
								newVol = 0
							}
							setVolume(currentAudioState.sinkID, newVol)
							currentAudioState.outputVolume = newVol
						}
					case 2:
						if currentAudioState.sourceID >= 0 {
							newVol := currentAudioState.inputVolume - 0.05
							if newVol < 0 {
								newVol = 0
							}
							setVolume(currentAudioState.sourceID, newVol)
							currentAudioState.inputVolume = newVol
						}
					}
				}
			case 'd': // Increase volume
				if currentAudioState != nil {
					switch currentSelection {
					case 1:
						if currentAudioState.sinkID >= 0 {
							newVol := currentAudioState.outputVolume + 0.05
							if newVol > 1.5 {
								newVol = 1.5
							}
							setVolume(currentAudioState.sinkID, newVol)
							currentAudioState.outputVolume = newVol
						}
					case 2:
						if currentAudioState.sourceID >= 0 {
							newVol := currentAudioState.inputVolume + 0.05
							if newVol > 1.5 {
								newVol = 1.5
							}
							setVolume(currentAudioState.sourceID, newVol)
							currentAudioState.inputVolume = newVol
						}
					}
				}
			}
		// ############# ANC Settings ##################
		case 9:
			switch key {
			case 'q':
				menuState = 5
				resetCurrentSelection = false
				currentANCState = nil
				updateHeadsetSettingsMenu()
			case 'w':
				handleUpKey()
			case 's':
				handleDownKey()
			case '\r': // Enter
				if currentANCState != nil {
					if currentSelection == 0 {
						// Cycle mode
						nextIdx := 0
						for i, m := range currentANCState.supportedModes {
							if m == currentANCState.currentMode {
								nextIdx = (i + 1) % len(currentANCState.supportedModes)
								break
							}
						}
						if device, exists := deviceManager[selectedHeadset]; exists {
							setAmbienceMode(device.deviceID, currentANCState.supportedModes[nextIdx])
							currentANCState.currentMode = currentANCState.supportedModes[nextIdx]
							// Clamp selection if level slider disappeared
							_, hasLevel := currentANCState.maxLevels[currentANCState.currentMode]
							if !hasLevel && currentSelection > 0 {
								currentSelection = 0
							}
						}
					} else if currentANCState.loopSupported {
						// Toggle loop checkbox
						_, hasLevel := currentANCState.maxLevels[currentANCState.currentMode]
						loopStartIdx := 1
						if hasLevel {
							loopStartIdx = 2
						}
						if currentSelection >= loopStartIdx {
							loopIdx := currentSelection - loopStartIdx
							if loopIdx >= 0 && loopIdx < len(currentANCState.supportedModes) {
								toggleModeInLoop(currentANCState.supportedModes[loopIdx])
								if device, exists := deviceManager[selectedHeadset]; exists {
									setAmbienceModeLoop(device.deviceID, currentANCState.loopModes)
								}
							}
						}
					}
				}
			case 'a': // Decrease level (more effect = lower number)
				if currentANCState != nil && currentSelection == 1 {
					if maxLvl, ok := currentANCState.maxLevels[currentANCState.currentMode]; ok {
						curLvl := currentANCState.currentLevels[currentANCState.currentMode]
						if curLvl > 0 {
							newLvl := curLvl - 1
							if device, exists := deviceManager[selectedHeadset]; exists {
								if setAmbienceModeLevel(device.deviceID, currentANCState.currentMode, newLvl) == nil {
									currentANCState.currentLevels[currentANCState.currentMode] = newLvl
								}
							}
						}
						_ = maxLvl
					}
				}
			case 'd': // Increase level (less effect = higher number)
				if currentANCState != nil && currentSelection == 1 {
					if maxLvl, ok := currentANCState.maxLevels[currentANCState.currentMode]; ok {
						curLvl := currentANCState.currentLevels[currentANCState.currentMode]
						if curLvl < maxLvl {
							newLvl := curLvl + 1
							if device, exists := deviceManager[selectedHeadset]; exists {
								if setAmbienceModeLevel(device.deviceID, currentANCState.currentMode, newLvl) == nil {
									currentANCState.currentLevels[currentANCState.currentMode] = newLvl
								}
							}
						}
					}
				}
			}
		// ############# Equalizer ##################
		case 7:
			switch key {
			case 'q':
				menuState = 5
				resetCurrentSelection = false
			case 'w':
				handleUpKey()
			case 's':
				handleDownKey()
			case 'a': // Decrease gain
				if len(equalizerBands) > 0 && currentSelection < len(equalizerBands) {
					band := &equalizerBands[currentSelection]
					newGain := band.currentGain - 1.0
					if newGain >= -band.maxGain {
						band.currentGain = newGain
						if device, exists := deviceManager[selectedHeadset]; exists {
							gains := make([]float32, len(equalizerBands))
							for i, b := range equalizerBands {
								gains[i] = b.currentGain
							}
							setEqualizerParameters(device.deviceID, gains)
						}
					}
				}
			case 'd': // Increase gain
				if len(equalizerBands) > 0 && currentSelection < len(equalizerBands) {
					band := &equalizerBands[currentSelection]
					newGain := band.currentGain + 1.0
					if newGain <= band.maxGain {
						band.currentGain = newGain
						if device, exists := deviceManager[selectedHeadset]; exists {
							gains := make([]float32, len(equalizerBands))
							for i, b := range equalizerBands {
								gains[i] = b.currentGain
							}
							setEqualizerParameters(device.deviceID, gains)
						}
					}
				}
			}
		}
	}
}

func handleUpKey() {
	if currentSelection > 0 {
		currentSelection--
	}
}

func handleDownKey() {
	switch menuState {
	case 0: // StartMenu
		if currentSelection < len(startMenu)-1 {
			currentSelection++
		}
	case 2: // See Remembered Paired Devices
		if dongle, exists := deviceManager[selectedDongle]; exists {
			if currentSelection < len(dongle.pairingList.pairedDevices)-1 {
				currentSelection++
			}
		}
	case 3: // Dongle Settings
		if currentSelection < len(dongleSettignsMenu)-1 {
			currentSelection++
		}
	case 5: // Headset Settings
		if currentSelection < len(headsetSettingsMenu)-1 {
			currentSelection++
		}
	case 7: // Equalizer
		if currentSelection < len(equalizerBands)-1 {
			currentSelection++
		}
	case 9: // ANC Settings
		if currentANCState != nil {
			maxIdx := 0 // mode selector
			_, hasLevel := currentANCState.maxLevels[currentANCState.currentMode]
			if hasLevel {
				maxIdx = 1 // level slider
			}
			if currentANCState.loopSupported {
				loopStart := 1
				if hasLevel {
					loopStart = 2
				}
				maxIdx = loopStart + len(currentANCState.supportedModes) - 1
			}
			if currentSelection < maxIdx {
				currentSelection++
			}
		}
	case 8: // Audio Settings
		maxIdx := 0
		if currentAudioState != nil {
			if currentAudioState.sinkID >= 0 {
				maxIdx = 1
			}
			if currentAudioState.sourceID >= 0 {
				maxIdx = 2
			}
		}
		if currentSelection < maxIdx {
			currentSelection++
		}
	}
}

func moveCursor(row, col int) {
	fmt.Printf("\033[%d;%dH", row, col) // ANSI escape to move to row and column
}

func clearScreen() {
	fmt.Print("\033[2J") // Clear the screen
	fmt.Print("\033[H")  // Move the cursor to the top-left corner
}

func getScreenSize() {
	getWidth, getHeight, err := term.GetSize(1)
	if err != nil {
		log.Fatalln(err)
	}
	width, height = getWidth, getHeight
}

func drawingBox() {

	calcHeight := height - 4
	calcWidth := (width - 11) * 3
	if calcWidth > len(horizontalLine) {
		return
	}

	// Using horizontalLine[:(width-11)*3] is faster than using strings.Repeat,
	// as it directly slices the precomputed string to the required length.
	// The factor `3` accounts for each Unicode characte taking 3 byte
	moveCursor(3, 5)
	fmt.Printf("%s%s%s", leftCornerTop, horizontalLine[:calcWidth], rightCornerTop)

	for i := 4; i < calcHeight; i++ {
		moveCursor(i, 5)
		fmt.Print(verticalLine)
		moveCursor(i, width-5)
		fmt.Print(verticalLine)
		moveCursor(i, 6)
	}

	moveCursor(calcHeight, 5)
	fmt.Printf("%s%s%s", leftCornerBottom, horizontalLine[:(width-11)*3], rightCornerBottom)

}

func header() {
	moveCursor(2, 5)
	dongle, exists := deviceManager[selectedDongle]
	if !exists {
		fmt.Printf("Looking For Dongle %s", loading[loadingIndex])
		loadingIndex = (loadingIndex + 1) % len(loading)
		return
	}
	fmt.Printf("%s", dongle.deviceName)

	headset, exists := deviceManager[selectedHeadset]
	if !exists {
		moveCursor(2, width-25)
		fmt.Printf("Looking For HeadSet %s", loading[loadingIndex])
		loadingIndex = (loadingIndex + 1) % len(loading)
		return
	}
	if headset.batteryStatus == nil {
		return
	}

	levelInPercent := headset.batteryStatus.levelInPercent
	filledSegments := int(math.Round(float64(levelInPercent) / 100 * batteryWidth))
	emptySegments := batteryWidth - filledSegments
	var color string
	switch {
	case headset.batteryStatus.batteryLow:
		color = "\033[31m" // Red for low battery
	case levelInPercent <= 65:
		color = "\033[33m" // Yellow for medium battery
	default:
		color = "\033[32m" // Green for high battery
	}

	batteryBar := color +
		strings.Repeat(batteryFullChar, filledSegments) +
		strings.Repeat(batteryEmptyChar, emptySegments) +
		"\033[0m" // Reset color

	btIndicator := ""
	if linkQualitySet {
		switch linkQualityStatus {
		case 2:
			btIndicator = " [BT:High]"
		case 1:
			btIndicator = " [BT:Low]"
		default:
			btIndicator = " [BT:Off]"
		}
	}

	headIndicator := ""
	if headset.featureFlags.onHeadDetection && headDetectionSet {
		switch {
		case headDetectionLeft && headDetectionRight:
			headIndicator = " [Wearing]"
		case headDetectionLeft || headDetectionRight:
			l, r := "off", "off"
			if headDetectionLeft {
				l = "on"
			}
			if headDetectionRight {
				r = "on"
			}
			headIndicator = fmt.Sprintf(" [L:%s R:%s]", l, r)
		default:
			headIndicator = " [Off-head]"
		}
	}

	extraLen := len(btIndicator) + len(headIndicator)
	if headset.batteryStatus.charging {
		moveCursor(2, width-50-extraLen)
		fmt.Printf("%s - Battery : [%s]🗲 %d%%%s%s", headset.deviceName, batteryBar, levelInPercent, btIndicator, headIndicator)
	} else {
		moveCursor(2, width-48-extraLen)
		fmt.Printf("%s - Battery: [%s] %d%%%s%s", headset.deviceName, batteryBar, levelInPercent, btIndicator, headIndicator)
	}
}

func menu(width int) {
	resetCurrentSelection = false // we can make a map to rember what was the last currentSelection
	drawingBox()
	for i, option := range startMenu {
		mid := (width - len(option.label)) / 2

		if i == currentSelection {
			moveCursor(5+i, mid-1)
			fmt.Println("\033[42m", option.label, "\033[0m")
		} else {
			moveCursor(5+i, mid)
			fmt.Println(option.label)
		}
	}
}

func updateSearchDeviceList() {
	for {
		if menuState != 1 {
			return
		}
		if dongle, exists := deviceManager[selectedDongle]; exists {
			updateSearchDeviceLis := getSearchDeviceList(dongle.deviceID)
			if updateSearchDeviceLis != nil {
				searchDeviceList.count = updateSearchDeviceLis.count
				searchDeviceList.listType = updateSearchDeviceLis.listType
				searchDeviceList.pairedDevices = updateSearchDeviceLis.pairedDevices
			}
		}
		time.Sleep(time.Second)
	}
}

func menuSearchForNewDevices() {
	if !resetCurrentSelection {
		currentSelection = 0
		resetCurrentSelection = true
		if err := searchForNewDevices(); err != nil {
			fmt.Println(err)
		}
		go updateSearchDeviceList()
	}

	drawingBox()

	if len(searchDeviceList.pairedDevices) != 0 {
		for i, pairedDevice := range searchDeviceList.pairedDevices {
			moveCursor(4+i, 10)
			device := fmt.Sprintf("%d %s", i+1, pairedDevice.deviceName)
			if i == currentSelection {
				fmt.Println("\033[42m", device, "\033[0m")
			} else {
				fmt.Println(device)
			}
		}
	}

	calcWidth := 0
	for i, item := range menuItemsSearchForNewDevices {
		moveCursor(height-3, 7+calcWidth)

		if i == selectedItemsSearchForNewDevices {
			fmt.Println("\033[44m", item, "\033[0m")
			go func() { // selected animation
				time.Sleep(time.Millisecond * 200)
				selectedItemsSearchForNewDevices = -1
			}()
		} else {
			fmt.Println("\033[42m", item, "\033[0m")
		}
		calcWidth += len(item) + 3 // Add the item's width plus a space for separation
	}

}

func menuPairedDevices() {
	if !resetCurrentSelection {
		currentSelection = 0
		resetCurrentSelection = true
	}

	drawingBox()

	if dongle, exists := deviceManager[selectedDongle]; exists {
		for i, pairedDevice := range dongle.pairingList.pairedDevices {
			moveCursor(4+i, 10)
			device := fmt.Sprintf("%d %s", i+1, pairedDevice.deviceName)
			if pairedDevice.isConnected {
				device += " (Connected)"
			}
			if i == currentSelection {
				fmt.Println("\033[42m", device, "\033[0m")
			} else {
				fmt.Println(device)
			}
		}

		calcWidth := 0
		for i, item := range menuItemsPairedDevices {
			moveCursor(height-3, 7+calcWidth)

			if i == selectedItemsPairedDevices {
				fmt.Println("\033[44m", item, "\033[0m")
				go func() { // selected animation
					time.Sleep(time.Millisecond * 200)
					selectedItemsPairedDevices = -1
				}()
			} else {
				fmt.Println("\033[42m", item, "\033[0m")
			}
			calcWidth += len(item) + 3 // Add the item's width plus a space for separation
		}

	} else {
		startMenuSelected = -1
	}
}

func dongleSettigns() {
	if !resetCurrentSelection {
		currentSelection = 0
		resetCurrentSelection = true
	}

	drawingBox()

	for i, item := range dongleSettignsMenu {

		if i == currentSelection {
			moveCursor(4+i, 9)
			fmt.Println("\033[42m", item.label, "\033[0m")
		} else {
			moveCursor(4+i, 10)
			fmt.Println(item.label)
		}
	}

	moveCursor(height-3, 7)
	fmt.Println("\033[42m", "Q Back", "\033[0m")
}

func headsetSettings() {
	if !resetCurrentSelection {
		currentSelection = 0
		resetCurrentSelection = true
		updateHeadsetSettingsMenu()
	}

	drawingBox()

	for i, item := range headsetSettingsMenu {
		if i == currentSelection {
			moveCursor(4+i, 9)
			fmt.Println("\033[42m", item.label, "\033[0m")
		} else {
			moveCursor(4+i, 10)
			fmt.Println(item.label)
		}
	}

	moveCursor(height-3, 7)
	fmt.Println("\033[42m", "Q Back", "\033[0m")
}

func buildDeviceInfoLines() []string {
	var lines []string

	if dongle, exists := deviceManager[selectedDongle]; exists {
		lines = append(lines, fmt.Sprintf("--- %s ---", dongle.deviceName))
		if fw := getFirmwareVersion(dongle.deviceID); fw != "" {
			fwLine := fmt.Sprintf("  Firmware:  %s", fw)
			if available, latestVer := checkFirmwareUpdate(dongle.deviceID); available && latestVer != "" {
				fwLine += fmt.Sprintf(" -> %s available", latestVer)
			} else if available {
				fwLine += " (update available)"
			} else {
				fwLine += " (up to date)"
			}
			lines = append(lines, fwLine)
		}
		if esn := getESN(dongle.deviceID); esn != "" {
			lines = append(lines, fmt.Sprintf("  ESN:       %s", esn))
		}
		if sku := getSku(dongle.deviceID); sku != "" {
			lines = append(lines, fmt.Sprintf("  SKU:       %s", sku))
		}
		if name := getConnectedBTDeviceName(dongle.deviceID); name != "" {
			lines = append(lines, fmt.Sprintf("  Connected: %s", name))
		}
		if secMode := getSecureConnectionMode(dongle.deviceID); secMode != "" {
			lines = append(lines, fmt.Sprintf("  Security:  %s", secMode))
		}
		constLines := getDeviceConstantLines(dongle.deviceID)
		lines = append(lines, constLines...)
		lines = append(lines, "")
	}

	if headset, exists := deviceManager[selectedHeadset]; exists {
		lines = append(lines, fmt.Sprintf("--- %s ---", headset.deviceName))
		if fw := getFirmwareVersion(headset.deviceID); fw != "" {
			fwLine := fmt.Sprintf("  Firmware:  %s", fw)
			if available, latestVer := checkFirmwareUpdate(headset.deviceID); available && latestVer != "" {
				fwLine += fmt.Sprintf(" -> %s available", latestVer)
			} else if available {
				fwLine += " (update available)"
			} else {
				fwLine += " (up to date)"
			}
			lines = append(lines, fwLine)
		}
		if esn := getESN(headset.deviceID); esn != "" {
			lines = append(lines, fmt.Sprintf("  ESN:       %s", esn))
		}
		lines = append(lines, fmt.Sprintf("  Serial:    %s", headset.serialNumber))
		if headset.batteryStatus != nil {
			lines = append(lines, fmt.Sprintf("  Battery:   %d%%", headset.batteryStatus.levelInPercent))
		}
		constLines := getDeviceConstantLines(headset.deviceID)
		lines = append(lines, constLines...)
		lines = append(lines, "")
	}

	return lines
}

func deviceInfo() {
	if !resetCurrentSelection {
		resetCurrentSelection = true
		deviceInfoLines = buildDeviceInfoLines()
	}

	drawingBox()

	for i, line := range deviceInfoLines {
		moveCursor(4+i, 8)
		fmt.Print(line)
	}

	moveCursor(height-3, 7)
	fmt.Println("\033[42m", "Q Back", "\033[0m")
}

func audioSettings() {
	if !resetCurrentSelection {
		currentSelection = 0
		resetCurrentSelection = true
		refreshAudioState()
	}

	drawingBox()

	if currentAudioState == nil {
		moveCursor(4, 8)
		fmt.Print("No PipeWire audio device found")
		moveCursor(height-3, 7)
		fmt.Println("\033[42m", "Q Back", "\033[0m")
		return
	}

	// Profile line
	profileLabel := "Unknown"
	for _, p := range currentAudioState.profiles {
		if p.index == currentAudioState.activeProfile {
			if p.description != "" {
				profileLabel = p.description
			} else {
				profileLabel = p.name
			}
			break
		}
	}
	moveCursor(4, 8)
	label := fmt.Sprintf("Audio Profile: %s", profileLabel)
	if currentSelection == 0 {
		fmt.Printf("\033[42m %s \033[0m  (Enter)", label)
	} else {
		fmt.Printf(" %s   (Enter)", label)
	}

	// Output volume bar
	if currentAudioState.sinkID >= 0 {
		moveCursor(6, 8)
		outPct := int(currentAudioState.outputVolume * 100)
		if outPct > 150 {
			outPct = 150
		}
		if outPct < 0 {
			outPct = 0
		}
		filled := outPct / 5
		if filled > 20 {
			filled = 20
		}
		empty := 20 - filled
		bar := strings.Repeat("\u2588", filled) + strings.Repeat("\u2591", empty)
		volLabel := fmt.Sprintf("Output Volume: [%s] %3d%%", bar, outPct)
		if currentSelection == 1 {
			fmt.Printf("\033[42m %s \033[0m  (A/D)", volLabel)
		} else {
			fmt.Printf(" %s   (A/D)", volLabel)
		}
	}

	// Input volume bar
	if currentAudioState.sourceID >= 0 {
		moveCursor(8, 8)
		inPct := int(currentAudioState.inputVolume * 100)
		if inPct > 150 {
			inPct = 150
		}
		if inPct < 0 {
			inPct = 0
		}
		filled := inPct / 5
		if filled > 20 {
			filled = 20
		}
		empty := 20 - filled
		bar := strings.Repeat("\u2588", filled) + strings.Repeat("\u2591", empty)
		volLabel := fmt.Sprintf("Input Volume:  [%s] %3d%%", bar, inPct)
		if currentSelection == 2 {
			fmt.Printf("\033[42m %s \033[0m  (A/D)", volLabel)
		} else {
			fmt.Printf(" %s   (A/D)", volLabel)
		}
	}

	moveCursor(height-3, 7)
	fmt.Println("\033[42m", "Q Back  A/D Adjust", "\033[0m")
}

func equalizerSettings() {
	if !resetCurrentSelection {
		currentSelection = 0
		resetCurrentSelection = true
		if device, exists := deviceManager[selectedHeadset]; exists {
			bands, err := getEqualizerParameters(device.deviceID)
			if err == nil {
				equalizerBands = bands
			}
		}
	}

	drawingBox()

	for i, band := range equalizerBands {
		moveCursor(4+i, 8)

		freqLabel := fmt.Sprintf("%5d Hz", band.centerFrequency)

		const barWidth = 20
		normalized := (band.currentGain + band.maxGain) / (2 * band.maxGain)
		if normalized < 0 {
			normalized = 0
		}
		if normalized > 1 {
			normalized = 1
		}
		filled := int(normalized * barWidth)
		empty := barWidth - filled
		bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

		gainLabel := fmt.Sprintf("%+.1f dB", band.currentGain)

		if i == currentSelection {
			fmt.Printf("\033[42m %s  [%s]  %s \033[0m", freqLabel, bar, gainLabel)
		} else {
			fmt.Printf(" %s  [%s]  %s", freqLabel, bar, gainLabel)
		}
	}

	moveCursor(height-3, 7)
	fmt.Println("\033[42m", "Q Back  A/D Adjust", "\033[0m")
}

func ancSettings() {
	if !resetCurrentSelection {
		currentSelection = 0
		resetCurrentSelection = true
		if device, exists := deviceManager[selectedHeadset]; exists {
			initANCScreenState(device.deviceID)
		}
	}

	drawingBox()

	if currentANCState == nil {
		moveCursor(4, 8)
		fmt.Print("ANC not available")
		moveCursor(height-3, 7)
		fmt.Println("\033[42m", "Q Back", "\033[0m")
		return
	}

	row := 4
	// Mode selector
	modeLabel := fmt.Sprintf("ANC Mode: %s", ambienceModeName(currentANCState.currentMode))
	if currentSelection == 0 {
		moveCursor(row, 7)
		fmt.Printf("\033[42m %s \033[0m  (Enter)", modeLabel)
	} else {
		moveCursor(row, 8)
		fmt.Printf(" %s   (Enter)", modeLabel)
	}
	row += 2

	// Level slider (if current mode supports levels)
	if maxLvl, ok := currentANCState.maxLevels[currentANCState.currentMode]; ok && maxLvl > 0 {
		curLvl := currentANCState.currentLevels[currentANCState.currentMode]
		// 0=max effect, maxLvl=min effect. Invert for display: filled = effect strength
		effectFilled := int(maxLvl) - int(curLvl)
		effectEmpty := int(curLvl)
		bar := strings.Repeat("\u2588", effectFilled) + strings.Repeat("\u2591", effectEmpty)
		levelLabel := fmt.Sprintf("Level: [%s] %d/%d", bar, curLvl, maxLvl)
		if currentSelection == 1 {
			moveCursor(row, 7)
			fmt.Printf("\033[42m %s \033[0m  (A/D)", levelLabel)
		} else {
			moveCursor(row, 8)
			fmt.Printf(" %s   (A/D)", levelLabel)
		}
		row += 2
	}

	// Mode loop checkboxes
	if currentANCState.loopSupported {
		moveCursor(row, 8)
		fmt.Print("--- Mode Loop (Enter to toggle) ---")
		row++

		_, hasLevel := currentANCState.maxLevels[currentANCState.currentMode]
		loopStartIdx := 1
		if hasLevel {
			loopStartIdx = 2
		}

		for i, mode := range currentANCState.supportedModes {
			inLoop := false
			for _, lm := range currentANCState.loopModes {
				if lm == mode {
					inLoop = true
					break
				}
			}
			check := " "
			if inLoop {
				check = "X"
			}
			label := fmt.Sprintf("[%s] %s", check, ambienceModeName(mode))
			itemIdx := loopStartIdx + i
			if currentSelection == itemIdx {
				moveCursor(row, 7)
				fmt.Printf("\033[42m %s \033[0m", label)
			} else {
				moveCursor(row, 8)
				fmt.Print(label)
			}
			row++
		}
	}

	moveCursor(height-3, 7)
	fmt.Println("\033[42m", "Q Back  A/D Adjust", "\033[0m")
}

func startUi() {
	sigChan := make(chan os.Signal, 1)
	go func() {
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	}()

	for {
		select {
		case <-sigChan:
			return
		default:
			clearScreen()
			getScreenSize()
			header()

			if startMenuSelected != -1 {
				switch startMenu[startMenuSelected].id {
				case 0: // Search For New Devices
					menuState = 1
					menuSearchForNewDevices()
				case 1: // See Remembered Paired Devices
					menuState = 2
					menuPairedDevices()
				case 2: // Dongle Settings
					menuState = 3
					dongleSettigns()
				case 3: // Switch Device
					moveCursor(4, 5)
					fmt.Println("Switch Device")
					menuState = 4
				case 4: // HeadSet Settings
					if menuState == 7 {
						equalizerSettings()
					} else if menuState == 9 {
						ancSettings()
					} else {
						menuState = 5
						headsetSettings()
					}
				case 6: // Device Info
					menuState = 6
					deviceInfo()
				case 7: // Audio Settings
					if menuState != 8 {
						menuState = 8
						resetCurrentSelection = false
					}
					audioSettings()
				case 5: // Exit
					return
				}
			} else {
				menuState = 0
				menu(width)
			}

			time.Sleep(time.Second / 12) // 12 Fps
		}
	}
}
