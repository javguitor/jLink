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
					case 0: // ANC Mode toggle
						if device, exists := deviceManager[selectedHeadset]; exists {
							supportedModes, err := getSupportedAmbienceModes(device.deviceID)
							if err == nil && len(supportedModes) > 0 {
								currentMode, _ := getAmbienceMode(device.deviceID)
								nextIdx := 0
								for i, m := range supportedModes {
									if m == currentMode {
										nextIdx = (i + 1) % len(supportedModes)
										break
									}
								}
								setAmbienceMode(device.deviceID, supportedModes[nextIdx])
								updateHeadsetSettingsMenu()
							}
						}
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
					}
				}
			}
		// ############# Device Info ##################
		case 6:
			switch key {
			case 'q':
				startMenuSelected = -1
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

	if headset.batteryStatus.charging {
		moveCursor(2, width-50)
		fmt.Printf("%s - Battery : [%s]🗲 %d%%", headset.deviceName, batteryBar, levelInPercent)
	} else {
		moveCursor(2, width-48)
		fmt.Printf("%s - Battery: [%s] %d%%", headset.deviceName, batteryBar, levelInPercent)
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
			lines = append(lines, fmt.Sprintf("  Firmware:  %s", fw))
		}
		if esn := getESN(dongle.deviceID); esn != "" {
			lines = append(lines, fmt.Sprintf("  ESN:       %s", esn))
		}
		if sku := getSku(dongle.deviceID); sku != "" {
			lines = append(lines, fmt.Sprintf("  SKU:       %s", sku))
		}
		lines = append(lines, "")
	}

	if headset, exists := deviceManager[selectedHeadset]; exists {
		lines = append(lines, fmt.Sprintf("--- %s ---", headset.deviceName))
		if fw := getFirmwareVersion(headset.deviceID); fw != "" {
			lines = append(lines, fmt.Sprintf("  Firmware:  %s", fw))
		}
		if esn := getESN(headset.deviceID); esn != "" {
			lines = append(lines, fmt.Sprintf("  ESN:       %s", esn))
		}
		lines = append(lines, fmt.Sprintf("  Serial:    %s", headset.serialNumber))
		if headset.batteryStatus != nil {
			lines = append(lines, fmt.Sprintf("  Battery:   %d%%", headset.batteryStatus.levelInPercent))
		}
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
					} else {
						menuState = 5
						headsetSettings()
					}
				case 6: // Device Info
					menuState = 6
					deviceInfo()
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
