package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ======================== Helpers ========================

func buildDeviceInfoLines() ([]string, []fwUpdateAvailability) {
	var lines []string
	var fwActions []fwUpdateAvailability

	if dongle, exists := deviceManager[selectedDongle]; exists {
		lines = append(lines, fmt.Sprintf("--- %s ---", dongle.deviceName))
		if fw := getFirmwareVersion(dongle.deviceID); fw != "" {
			fwLine := fmt.Sprintf("  Firmware:  %s", fw)
			if available, latestVer := checkFirmwareUpdate(dongle.deviceID); available && latestVer != "" {
				fwLine += fmt.Sprintf(" -> %s available", latestVer)
				lines = append(lines, fwLine)
				actionLine := fmt.Sprintf("     >> Download & Update to %s", latestVer)
				fwActions = append(fwActions, fwUpdateAvailability{
					deviceID:   dongle.deviceID,
					deviceName: dongle.deviceName,
					version:    latestVer,
					lineIndex:  len(lines),
				})
				lines = append(lines, actionLine)
			} else if available {
				fwLine += " (update available)"
				lines = append(lines, fwLine)
			} else {
				fwLine += " (up to date)"
				lines = append(lines, fwLine)
			}
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
				lines = append(lines, fwLine)
				actionLine := fmt.Sprintf("     >> Download & Update to %s", latestVer)
				fwActions = append(fwActions, fwUpdateAvailability{
					deviceID:   headset.deviceID,
					deviceName: headset.deviceName,
					version:    latestVer,
					lineIndex:  len(lines),
				})
				lines = append(lines, actionLine)
			} else if available {
				fwLine += " (update available)"
				lines = append(lines, fwLine)
			} else {
				fwLine += " (up to date)"
				lines = append(lines, fwLine)
			}
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

	return lines, fwActions
}

// firstSelectableIndex returns the index of the first selectable menu item (id != -1)
func firstSelectableIndex(items []menuItem) int {
	for i, item := range items {
		if item.id != -1 {
			return i
		}
	}
	return 0
}

// ======================== Main Menu ========================

func (m model) updateMainMenu(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "w", "up":
		for next := m.cursor - 1; next >= 0; next-- {
			if m.mainMenuItems[next].id != -1 {
				m.cursor = next
				break
			}
		}
	case "s", "down":
		for next := m.cursor + 1; next < len(m.mainMenuItems); next++ {
			if m.mainMenuItems[next].id != -1 {
				m.cursor = next
				break
			}
		}
	case "enter":
		if m.cursor >= 0 && m.cursor < len(m.mainMenuItems) {
			item := m.mainMenuItems[m.cursor]
			switch item.id {
			case 0: // Search For New Devices
				m.screen = screenSearchDevices
				m.cursor = 0
				m.searchResults = nil
				searchDeviceList.pairedDevices = nil
				searchDeviceList.count = 0
				searchForNewDevices()
				return m, tickSearch()
			case 1: // See Remembered Paired Devices
				m.screen = screenPairedDevices
				m.cursor = 0
				if m.dongle != nil && m.dongle.pairingList != nil {
					m.pairedDevices = m.dongle.pairingList.pairedDevices
				}
			case 2: // Dongle Settings
				m.screen = screenDongleSettings
				m.cursor = 0
				updateDongleSettignsMenu()
				m.dongleMenuItems = dongleSettignsMenu
			case 3: // Switch Device
				m.screen = screenSwitchDevice
				m.cursor = 0
			case 4: // Headset Settings
				m.screen = screenHeadsetSettings
				m.cursor = 0
				updateHeadsetSettingsMenu()
				m.headsetMenuItems = headsetSettingsMenu
			case 6: // Device Info
				m.screen = screenDeviceInfo
				m.cursor = 0
				m.infoLines, m.fwActions = buildDeviceInfoLines()
				fwUpdatesAvailable = m.fwActions
			case 7: // Audio Settings
				m.screen = screenAudioSettings
				m.cursor = 0
				if currentAudioState == nil {
					currentAudioState = discoverPipeWireDevice()
				}
				if currentAudioState != nil {
					refreshAudioState()
				}
				m.audio = currentAudioState
			case 5: // Exit
				return m, tea.Quit
			}
		}
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

func (m model) viewMainMenu() string {
	logo := renderLogo()

	var lines []string
	lines = append(lines, logo)
	lines = append(lines, "")

	for i, item := range m.mainMenuItems {
		if item.id == -1 {
			// Section separator
			if item.label == "" {
				lines = append(lines, "")
			} else {
				lines = append(lines, mutedStyle.Render(item.label))
			}
			continue
		}
		if i == m.cursor {
			lines = append(lines, selectedStyle.Render(item.label))
		} else {
			lines = append(lines, normalStyle.Render(item.label))
		}
	}

	menu := lipgloss.JoinVertical(lipgloss.Left, lines...)

	return lipgloss.Place(
		m.width-6, m.height-6,
		lipgloss.Center, lipgloss.Center,
		menu,
	)
}

// ======================== Search Devices ========================

func (m model) updateSearchDevices(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "q":
		setDongleInBTPairing(false)
		m.screen = screenMainMenu
		m.cursor = 0
		return m, nil
	case "1":
		if len(m.searchResults) > 0 && m.cursor < len(m.searchResults) {
			if err := connectNewDevice(uint16(m.cursor)); err == nil {
				m.screen = screenMainMenu
				m.cursor = 0
			}
		}
		return m.triggerFeedback(1)
	case "w", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "s", "down":
		if len(m.searchResults) > 0 && m.cursor < len(m.searchResults)-1 {
			m.cursor++
		}
	}
	return m, nil
}

func (m model) viewSearchDevices() string {
	var lines []string

	lines = append(lines, titleStyle.Render("Search For New Devices")+" "+m.spinner.View())
	lines = append(lines, "")

	if len(m.searchResults) == 0 {
		lines = append(lines, mutedStyle.Render("Searching for devices..."))
	} else {
		for i, dev := range m.searchResults {
			label := fmt.Sprintf("%d  %s", i+1, dev.deviceName)
			if i == m.cursor {
				lines = append(lines, selectedStyle.Render(label))
			} else {
				lines = append(lines, normalStyle.Render(label))
			}
		}
	}

	lines = append(lines, "")

	// Footer buttons
	buttons := []string{"Q Back", "1 Connect"}
	var footerParts []string
	for i, btn := range buttons {
		if i == m.actionFeedback {
			footerParts = append(footerParts, buttonActiveStyle.Render(btn))
		} else {
			footerParts = append(footerParts, buttonStyle.Render(btn))
		}
	}
	lines = append(lines, strings.Join(footerParts, " "))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// ======================== Paired Devices ========================

func (m model) updatePairedDevices(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.screen = screenMainMenu
		m.cursor = 0
	case "w", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "s", "down":
		if m.cursor < len(m.pairedDevices)-1 {
			m.cursor++
		}
	case "1": // Connect
		if len(m.pairedDevices) > 0 {
			connectDeviceFromPairedlist(uint16(m.cursor))
		}
		return m.triggerFeedback(1)
	case "2": // Disconnect
		if len(m.pairedDevices) > 0 {
			disconnectDeviceFromPairedlist(uint16(m.cursor))
		}
		return m.triggerFeedback(2)
	case "3": // Remove
		if len(m.pairedDevices) > 0 {
			removeDeviceFromPairedlist(uint16(m.cursor))
		}
		return m.triggerFeedback(3)
	case "4": // Clear
		clearPairingList()
		return m.triggerFeedback(4)
	}
	return m, nil
}

func (m model) viewPairedDevices() string {
	var lines []string

	lines = append(lines, titleStyle.Render("Paired Devices"))
	lines = append(lines, "")

	if len(m.pairedDevices) == 0 {
		lines = append(lines, mutedStyle.Render("No paired devices found"))
	} else {
		for i, dev := range m.pairedDevices {
			label := fmt.Sprintf("%d  %s", i+1, dev.deviceName)
			if dev.isConnected {
				label += successStyle.Render(" (Connected)")
			}
			if i == m.cursor {
				lines = append(lines, selectedStyle.Render(label))
			} else {
				lines = append(lines, normalStyle.Render(label))
			}
		}
	}

	lines = append(lines, "")

	buttons := []string{"Q Back", "1 Connect", "2 Disconnect", "3 Remove", "4 Clear"}
	var footerParts []string
	for i, btn := range buttons {
		if i == m.actionFeedback {
			footerParts = append(footerParts, buttonActiveStyle.Render(btn))
		} else {
			footerParts = append(footerParts, buttonStyle.Render(btn))
		}
	}
	lines = append(lines, strings.Join(footerParts, " "))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// ======================== Dongle Settings ========================

func (m model) updateDongleSettings(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.screen = screenMainMenu
		m.cursor = 0
	case "w", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "s", "down":
		if m.cursor < len(m.dongleMenuItems)-1 {
			m.cursor++
		}
	case "enter":
		if m.cursor < len(m.dongleMenuItems) {
			item := m.dongleMenuItems[m.cursor]
			switch item.id {
			case 0: // AutoPairing toggle
				state, _ := getAutoPairing()
				setAutoPairing(!state)
				updateDongleSettignsMenu()
				m.dongleMenuItems = dongleSettignsMenu
			case 1: // Factory Reset
				if dongle, exists := deviceManager[selectedDongle]; exists {
					factoryReset(dongle.deviceID)
					m.screen = screenMainMenu
					m.cursor = 0
				}
			}
		}
	}
	return m, nil
}

func (m model) viewDongleSettings() string {
	var lines []string

	lines = append(lines, titleStyle.Render("Dongle Settings"))
	lines = append(lines, "")

	for i, item := range m.dongleMenuItems {
		if i == m.cursor {
			lines = append(lines, selectedStyle.Render(item.label))
		} else {
			lines = append(lines, normalStyle.Render(item.label))
		}
	}

	lines = append(lines, "")
	lines = append(lines, renderFooter([]string{"Q Back", "Enter Select"}))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// ======================== Switch Device ========================

func (m model) updateSwitchDevice(msg tea.KeyMsg) (model, tea.Cmd) {
	if msg.String() == "q" {
		m.screen = screenMainMenu
		m.cursor = 0
	}
	return m, nil
}

func (m model) viewSwitchDevice() string {
	var lines []string
	lines = append(lines, titleStyle.Render("Switch Device"))
	lines = append(lines, "")
	lines = append(lines, mutedStyle.Render("Not yet implemented"))
	lines = append(lines, "")
	lines = append(lines, renderFooter([]string{"Q Back"}))
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// ======================== Headset Settings ========================

func (m model) updateHeadsetSettings(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.screen = screenMainMenu
		m.cursor = 0
	case "w", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "s", "down":
		if m.cursor < len(m.headsetMenuItems)-1 {
			m.cursor++
		}
	case "enter":
		if m.cursor < len(m.headsetMenuItems) {
			item := m.headsetMenuItems[m.cursor]
			switch item.id {
			case 0: // ANC Mode -> ANC settings sub-screen
				m.screen = screenANCSettings
				m.cursor = 0
				if device, exists := deviceManager[selectedHeadset]; exists {
					initANCScreenState(device.deviceID)
				}
				m.ancState = currentANCState
			case 1: // Equalizer
				m.screen = screenEqualizer
				m.cursor = 0
				if device, exists := deviceManager[selectedHeadset]; exists {
					bands, err := getEqualizerParameters(device.deviceID)
					if err == nil {
						m.eqBands = bands
					}
				}
			case 2: // Sidetone toggle
				if device, exists := deviceManager[selectedHeadset]; exists {
					sidetone := findDeviceSetting(device.deviceID, "sidetone")
					if sidetone != nil && len(sidetone.options) > 0 {
						nextKey := (sidetone.current + 1) % len(sidetone.options)
						setDeviceSetting(device.deviceID, sidetone.guid, nextKey)
						updateHeadsetSettingsMenu()
						m.headsetMenuItems = headsetSettingsMenu
					}
				}
			case 3: // Busy Light toggle
				if device, exists := deviceManager[selectedHeadset]; exists {
					current := getBusyLightStatus(device.deviceID, device.featureFlags)
					setBusyLightStatus(device.deviceID, !current, device.featureFlags)
					updateHeadsetSettingsMenu()
					m.headsetMenuItems = headsetSettingsMenu
				}
			}
		}
	}
	return m, nil
}

func (m model) viewHeadsetSettings() string {
	var lines []string

	lines = append(lines, titleStyle.Render("Headset Settings"))
	lines = append(lines, "")

	for i, item := range m.headsetMenuItems {
		if i == m.cursor {
			lines = append(lines, selectedStyle.Render(item.label))
		} else {
			lines = append(lines, normalStyle.Render(item.label))
		}
	}

	lines = append(lines, "")
	lines = append(lines, renderFooter([]string{"Q Back", "Enter Select"}))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// ======================== Device Info ========================

func (m model) updateDeviceInfo(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "q":
		fwUpdatesAvailable = nil
		m.screen = screenMainMenu
		m.cursor = 0
	case "w", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "s", "down":
		if len(m.fwActions) > 0 && m.cursor < len(m.fwActions)-1 {
			m.cursor++
		}
	case "enter":
		if len(m.fwActions) > 0 && m.cursor >= 0 && m.cursor < len(m.fwActions) {
			fwa := m.fwActions[m.cursor]
			go startFirmwareDownload(fwa.deviceID, fwa.version)
			m.screen = screenFirmwareUpdate
			m.cursor = 0
		}
	}
	return m, nil
}

func (m model) viewDeviceInfo() string {
	var lines []string

	lines = append(lines, titleStyle.Render("Device Information"))
	lines = append(lines, "")

	for i, line := range m.infoLines {
		isAction := false
		actionIdx := -1
		for j, fwa := range m.fwActions {
			if fwa.lineIndex == i {
				isAction = true
				actionIdx = j
				break
			}
		}
		if isAction {
			if actionIdx == m.cursor {
				lines = append(lines, selectedStyle.Render(line))
			} else {
				lines = append(lines, accentTextStyle.Render(line))
			}
		} else {
			lines = append(lines, normalStyle.Render(line))
		}
	}

	lines = append(lines, "")
	if len(m.fwActions) > 0 {
		lines = append(lines, renderFooter([]string{"Q Back", "W/S Navigate", "Enter Update"}))
	} else {
		lines = append(lines, renderFooter([]string{"Q Back"}))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// ======================== Firmware Update ========================

func (m model) updateFirmwareUpdate(msg tea.KeyMsg) (model, tea.Cmd) {
	m.fwState = fwUpdateState

	switch msg.String() {
	case "q":
		if m.fwState != nil && (m.fwState.phase == fwPhaseCompleted || m.fwState.phase == fwPhaseError || m.fwState.phase == fwPhaseCancelled) {
			fwUpdateState = nil
			m.fwState = nil
			m.screen = screenDeviceInfo
			m.cursor = 0
			m.infoLines, m.fwActions = buildDeviceInfoLines()
			fwUpdatesAvailable = m.fwActions
		}
	case "c":
		if m.fwState != nil && m.fwState.phase == fwPhaseDownload {
			cancelFirmwareDownload(m.fwState.deviceID)
		}
	}
	return m, nil
}

func (m model) viewFirmwareUpdate() string {
	m.fwState = fwUpdateState
	var lines []string

	lines = append(lines, titleStyle.Render("Firmware Update"))
	lines = append(lines, "")

	if m.fwState == nil {
		lines = append(lines, mutedStyle.Render("No firmware update in progress"))
		lines = append(lines, "")
		lines = append(lines, renderFooter([]string{"Q Back"}))
		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	lines = append(lines, fmt.Sprintf("Device: %s", m.fwState.deviceName))
	lines = append(lines, fmt.Sprintf("Target: %s", m.fwState.version))
	lines = append(lines, "")

	phaseLabel := "IDLE"
	switch m.fwState.phase {
	case fwPhaseDownload:
		phaseLabel = "DOWNLOAD"
	case fwPhaseUpdate:
		phaseLabel = "UPDATE"
	case fwPhaseCompleted:
		phaseLabel = successStyle.Render("COMPLETED")
	case fwPhaseCancelled:
		phaseLabel = warningStyle.Render("CANCELLED")
	case fwPhaseError:
		phaseLabel = dangerStyle.Render("ERROR")
	}
	lines = append(lines, fmt.Sprintf("Phase: %s", phaseLabel))
	lines = append(lines, "")

	// Progress bar
	bar := renderProgressBar(m.fwState.percentage, 100, 30)
	lines = append(lines, fmt.Sprintf("%s  %d%%", bar, m.fwState.percentage))
	lines = append(lines, "")
	lines = append(lines, m.fwState.statusMsg)

	lines = append(lines, "")
	switch m.fwState.phase {
	case fwPhaseDownload:
		lines = append(lines, renderFooter([]string{"C Cancel"}))
	case fwPhaseUpdate:
		lines = append(lines, warningStyle.Render("Updating... please wait"))
	case fwPhaseCompleted, fwPhaseError, fwPhaseCancelled:
		lines = append(lines, renderFooter([]string{"Q Back"}))
	default:
		lines = append(lines, mutedStyle.Render("Please wait..."))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// ======================== Audio Settings ========================

func (m model) updateAudioSettings(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.screen = screenMainMenu
		m.cursor = 0
	case "w", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "s", "down":
		maxIdx := 0
		if m.audio != nil {
			if m.audio.sinkID >= 0 {
				maxIdx = 1
			}
			if m.audio.sourceID >= 0 {
				maxIdx = 2
			}
		}
		if m.cursor < maxIdx {
			m.cursor++
		}
	case "enter":
		if m.audio != nil && m.cursor == 0 && len(m.audio.profiles) > 0 {
			nextIdx := -1
			for i, p := range m.audio.profiles {
				if p.index == m.audio.activeProfile {
					nextIdx = (i + 1) % len(m.audio.profiles)
					break
				}
			}
			if nextIdx < 0 {
				nextIdx = 0
			}
			setAudioProfile(m.audio.deviceID, m.audio.profiles[nextIdx].index)
			currentAudioState = discoverPipeWireDevice()
			refreshAudioState()
			m.audio = currentAudioState
		}
	case "a":
		if m.audio != nil {
			switch m.cursor {
			case 1:
				if m.audio.sinkID >= 0 {
					newVol := m.audio.outputVolume - 0.05
					if newVol < 0 {
						newVol = 0
					}
					setVolume(m.audio.sinkID, newVol)
					m.audio.outputVolume = newVol
					currentAudioState = m.audio
				}
			case 2:
				if m.audio.sourceID >= 0 {
					newVol := m.audio.inputVolume - 0.05
					if newVol < 0 {
						newVol = 0
					}
					setVolume(m.audio.sourceID, newVol)
					m.audio.inputVolume = newVol
					currentAudioState = m.audio
				}
			}
		}
	case "d":
		if m.audio != nil {
			switch m.cursor {
			case 1:
				if m.audio.sinkID >= 0 {
					newVol := m.audio.outputVolume + 0.05
					if newVol > 1.5 {
						newVol = 1.5
					}
					setVolume(m.audio.sinkID, newVol)
					m.audio.outputVolume = newVol
					currentAudioState = m.audio
				}
			case 2:
				if m.audio.sourceID >= 0 {
					newVol := m.audio.inputVolume + 0.05
					if newVol > 1.5 {
						newVol = 1.5
					}
					setVolume(m.audio.sourceID, newVol)
					m.audio.inputVolume = newVol
					currentAudioState = m.audio
				}
			}
		}
	}
	return m, nil
}

func (m model) viewAudioSettings() string {
	var lines []string

	lines = append(lines, titleStyle.Render("Audio Settings"))
	lines = append(lines, "")

	if m.audio == nil {
		lines = append(lines, mutedStyle.Render("No PipeWire audio device found"))
		lines = append(lines, "")
		lines = append(lines, renderFooter([]string{"Q Back"}))
		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	// Profile line
	profileLabel := "Unknown"
	for _, p := range m.audio.profiles {
		if p.index == m.audio.activeProfile {
			if p.description != "" {
				profileLabel = p.description
			} else {
				profileLabel = p.name
			}
			break
		}
	}

	profileLine := fmt.Sprintf("Audio Profile: %s  (Enter)", profileLabel)
	if m.cursor == 0 {
		lines = append(lines, selectedStyle.Render(profileLine))
	} else {
		lines = append(lines, normalStyle.Render(profileLine))
	}
	lines = append(lines, "")

	// Output volume
	if m.audio.sinkID >= 0 {
		outPct := int(m.audio.outputVolume * 100)
		if outPct > 150 {
			outPct = 150
		}
		if outPct < 0 {
			outPct = 0
		}
		bar := renderProgressBar(outPct, 150, 20)
		volLabel := fmt.Sprintf("Output Volume: %s %3d%%  (A/D)", bar, outPct)
		if m.cursor == 1 {
			lines = append(lines, selectedStyle.Render(volLabel))
		} else {
			lines = append(lines, normalStyle.Render(volLabel))
		}
		lines = append(lines, "")
	}

	// Input volume
	if m.audio.sourceID >= 0 {
		inPct := int(m.audio.inputVolume * 100)
		if inPct > 150 {
			inPct = 150
		}
		if inPct < 0 {
			inPct = 0
		}
		bar := renderProgressBar(inPct, 150, 20)
		volLabel := fmt.Sprintf("Input Volume:  %s %3d%%  (A/D)", bar, inPct)
		if m.cursor == 2 {
			lines = append(lines, selectedStyle.Render(volLabel))
		} else {
			lines = append(lines, normalStyle.Render(volLabel))
		}
		lines = append(lines, "")
	}

	lines = append(lines, renderFooter([]string{"Q Back", "A/D Adjust"}))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// ======================== Equalizer ========================

func (m model) updateEqualizer(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.screen = screenHeadsetSettings
		m.cursor = 0
		updateHeadsetSettingsMenu()
		m.headsetMenuItems = headsetSettingsMenu
	case "w", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "s", "down":
		if m.cursor < len(m.eqBands)-1 {
			m.cursor++
		}
	case "a":
		if len(m.eqBands) > 0 && m.cursor < len(m.eqBands) {
			band := &m.eqBands[m.cursor]
			newGain := band.currentGain - 1.0
			if newGain >= -band.maxGain {
				band.currentGain = newGain
				if device, exists := deviceManager[selectedHeadset]; exists {
					gains := make([]float32, len(m.eqBands))
					for i, b := range m.eqBands {
						gains[i] = b.currentGain
					}
					setEqualizerParameters(device.deviceID, gains)
				}
			}
		}
	case "d":
		if len(m.eqBands) > 0 && m.cursor < len(m.eqBands) {
			band := &m.eqBands[m.cursor]
			newGain := band.currentGain + 1.0
			if newGain <= band.maxGain {
				band.currentGain = newGain
				if device, exists := deviceManager[selectedHeadset]; exists {
					gains := make([]float32, len(m.eqBands))
					for i, b := range m.eqBands {
						gains[i] = b.currentGain
					}
					setEqualizerParameters(device.deviceID, gains)
				}
			}
		}
	}
	return m, nil
}

func (m model) viewEqualizer() string {
	var lines []string

	lines = append(lines, titleStyle.Render("Equalizer"))
	lines = append(lines, "")

	for i, band := range m.eqBands {
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

		line := fmt.Sprintf("%s  [%s]  %s", freqLabel, bar, gainLabel)
		if i == m.cursor {
			lines = append(lines, selectedStyle.Render(line))
		} else {
			lines = append(lines, normalStyle.Render(line))
		}
	}

	lines = append(lines, "")
	lines = append(lines, renderFooter([]string{"Q Back", "A/D Adjust"}))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// ======================== ANC Settings ========================

func (m model) updateANCSettings(msg tea.KeyMsg) (model, tea.Cmd) {
	if m.ancState == nil {
		if msg.String() == "q" {
			m.screen = screenHeadsetSettings
			m.cursor = 0
			updateHeadsetSettingsMenu()
			m.headsetMenuItems = headsetSettingsMenu
		}
		return m, nil
	}

	switch msg.String() {
	case "q":
		currentANCState = nil
		m.ancState = nil
		updateHeadsetSettingsMenu()
		m.screen = screenHeadsetSettings
		m.cursor = 0
		m.headsetMenuItems = headsetSettingsMenu
	case "w", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "s", "down":
		maxIdx := 0
		_, hasLevel := m.ancState.maxLevels[m.ancState.currentMode]
		if hasLevel {
			maxIdx = 1
		}
		if m.ancState.loopSupported {
			loopStart := 1
			if hasLevel {
				loopStart = 2
			}
			maxIdx = loopStart + len(m.ancState.supportedModes) - 1
		}
		if m.cursor < maxIdx {
			m.cursor++
		}
	case "enter":
		if m.cursor == 0 {
			// Cycle mode
			nextIdx := 0
			for i, mode := range m.ancState.supportedModes {
				if mode == m.ancState.currentMode {
					nextIdx = (i + 1) % len(m.ancState.supportedModes)
					break
				}
			}
			if device, exists := deviceManager[selectedHeadset]; exists {
				setAmbienceMode(device.deviceID, m.ancState.supportedModes[nextIdx])
				m.ancState.currentMode = m.ancState.supportedModes[nextIdx]
				_, hasLevel := m.ancState.maxLevels[m.ancState.currentMode]
				if !hasLevel && m.cursor > 0 {
					m.cursor = 0
				}
			}
		} else if m.ancState.loopSupported {
			_, hasLevel := m.ancState.maxLevels[m.ancState.currentMode]
			loopStartIdx := 1
			if hasLevel {
				loopStartIdx = 2
			}
			if m.cursor >= loopStartIdx {
				loopIdx := m.cursor - loopStartIdx
				if loopIdx >= 0 && loopIdx < len(m.ancState.supportedModes) {
					toggleModeInLoop(m.ancState.supportedModes[loopIdx])
					if device, exists := deviceManager[selectedHeadset]; exists {
						setAmbienceModeLoop(device.deviceID, m.ancState.loopModes)
					}
				}
			}
		}
		currentANCState = m.ancState
	case "a":
		if m.cursor == 1 {
			if maxLvl, ok := m.ancState.maxLevels[m.ancState.currentMode]; ok {
				curLvl := m.ancState.currentLevels[m.ancState.currentMode]
				if curLvl > 0 {
					newLvl := curLvl - 1
					if device, exists := deviceManager[selectedHeadset]; exists {
						if setAmbienceModeLevel(device.deviceID, m.ancState.currentMode, newLvl) == nil {
							m.ancState.currentLevels[m.ancState.currentMode] = newLvl
						}
					}
				}
				_ = maxLvl
			}
		}
		currentANCState = m.ancState
	case "d":
		if m.cursor == 1 {
			if maxLvl, ok := m.ancState.maxLevels[m.ancState.currentMode]; ok {
				curLvl := m.ancState.currentLevels[m.ancState.currentMode]
				if curLvl < maxLvl {
					newLvl := curLvl + 1
					if device, exists := deviceManager[selectedHeadset]; exists {
						if setAmbienceModeLevel(device.deviceID, m.ancState.currentMode, newLvl) == nil {
							m.ancState.currentLevels[m.ancState.currentMode] = newLvl
						}
					}
				}
			}
		}
		currentANCState = m.ancState
	}
	return m, nil
}

func (m model) viewANCSettings() string {
	var lines []string

	lines = append(lines, titleStyle.Render("ANC Settings"))
	lines = append(lines, "")

	if m.ancState == nil {
		lines = append(lines, mutedStyle.Render("ANC not available"))
		lines = append(lines, "")
		lines = append(lines, renderFooter([]string{"Q Back"}))
		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	// Mode selector
	modeLabel := fmt.Sprintf("ANC Mode: %s  (Enter)", ambienceModeName(m.ancState.currentMode))
	if m.cursor == 0 {
		lines = append(lines, selectedStyle.Render(modeLabel))
	} else {
		lines = append(lines, normalStyle.Render(modeLabel))
	}
	lines = append(lines, "")

	// Level slider
	if maxLvl, ok := m.ancState.maxLevels[m.ancState.currentMode]; ok && maxLvl > 0 {
		curLvl := m.ancState.currentLevels[m.ancState.currentMode]
		effectFilled := int(maxLvl) - int(curLvl)
		effectEmpty := int(curLvl)
		bar := strings.Repeat("█", effectFilled) + strings.Repeat("░", effectEmpty)
		levelLabel := fmt.Sprintf("Level: [%s] %d/%d  (A/D)", bar, curLvl, maxLvl)
		if m.cursor == 1 {
			lines = append(lines, selectedStyle.Render(levelLabel))
		} else {
			lines = append(lines, normalStyle.Render(levelLabel))
		}
		lines = append(lines, "")
	}

	// Mode loop checkboxes
	if m.ancState.loopSupported {
		lines = append(lines, mutedStyle.Render("--- Mode Loop (Enter to toggle) ---"))

		_, hasLevel := m.ancState.maxLevels[m.ancState.currentMode]
		loopStartIdx := 1
		if hasLevel {
			loopStartIdx = 2
		}

		for i, mode := range m.ancState.supportedModes {
			inLoop := false
			for _, lm := range m.ancState.loopModes {
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
			if m.cursor == itemIdx {
				lines = append(lines, selectedStyle.Render(label))
			} else {
				lines = append(lines, normalStyle.Render(label))
			}
		}
		lines = append(lines, "")
	}

	lines = append(lines, renderFooter([]string{"Q Back", "A/D Adjust"}))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
