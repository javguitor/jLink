package main

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Screen types
type screenType int

const (
	screenMainMenu       screenType = iota // 0
	screenSearchDevices                    // 1
	screenPairedDevices                    // 2
	screenDongleSettings                   // 3
	screenSwitchDevice                     // 4
	screenHeadsetSettings                  // 5
	screenDeviceInfo                       // 6
	screenEqualizer                        // 7
	screenAudioSettings                    // 8
	screenANCSettings                      // 9
	screenFirmwareUpdate                   // 10
)

// Message types
type deviceAttachedMsg struct{ info *jabra_DeviceInfo }
type deviceRemovedMsg struct{ deviceID uint16 }
type headDetectionMsg struct{ left, right bool }
type linkQualityMsg struct{ status int }
type firmwareProgressMsg struct {
	deviceID                       uint16
	eventType, status, percentage  int
}

type batteryTickMsg time.Time
type pairingTickMsg time.Time
type searchTickMsg time.Time
type feedbackDoneMsg struct{}

// Root model
type model struct {
	screen screenType
	width  int
	height int
	cursor int

	// Device state (refreshed from jabraApi globals)
	dongle  *jabra_DeviceInfo
	headset *jabra_DeviceInfo

	// Per-screen cached state
	mainMenuItems    []menuItem
	dongleMenuItems  []menuItem
	headsetMenuItems []menuItem
	searchResults    []pairedDevice
	pairedDevices    []pairedDevice
	eqBands          []equalizerBand
	ancState         *ancScreenState
	audio            *audioState
	infoLines        []string
	fwActions        []fwUpdateAvailability
	fwState          *firmwareUpdateState

	// UI components
	spinner spinner.Model

	// Action feedback (button flash for paired devices / search)
	actionFeedback int // -1 = none
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	return model{
		screen:         screenMainMenu,
		cursor:         0,
		actionFeedback: -1,
		spinner:        s,
	}
}

// Channel subscription functions (blocking reads, re-launched after each msg)
func waitForDeviceAttached() tea.Msg {
	return deviceAttachedMsg{info: <-chDeviceAttached}
}

func waitForDeviceRemoved() tea.Msg {
	return deviceRemovedMsg{deviceID: <-chDeviceRemoved}
}

func waitForHeadDetection() tea.Msg {
	pair := <-chHeadDetection
	return headDetectionMsg{left: pair[0], right: pair[1]}
}

func waitForLinkQuality() tea.Msg {
	return linkQualityMsg{status: <-chLinkQuality}
}

func waitForFirmwareProgress() tea.Msg {
	ev := <-chFirmwareProgress
	return firmwareProgressMsg{
		deviceID:  ev.deviceID,
		eventType: ev.eventType,
		status:    ev.status,
		percentage: ev.percentage,
	}
}

func tickBattery() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return batteryTickMsg(t) })
}

func tickPairing() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return pairingTickMsg(t) })
}

func tickSearch() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return searchTickMsg(t) })
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		waitForDeviceAttached,
		waitForDeviceRemoved,
		waitForHeadDetection,
		waitForLinkQuality,
		waitForFirmwareProgress,
		tickBattery(),
		tickPairing(),
		m.spinner.Tick,
	)
}

func (m model) refreshDeviceState() model {
	if selectedDongle >= 0 {
		if d, ok := deviceManager[selectedDongle]; ok {
			m.dongle = d
		}
	} else {
		m.dongle = nil
	}
	if selectedHeadset >= 0 {
		if h, ok := deviceManager[selectedHeadset]; ok {
			m.headset = h
		}
	} else {
		m.headset = nil
	}
	updateStartMenu()
	m.mainMenuItems = startMenu
	// Ensure cursor doesn't land on a separator when on main menu
	if m.screen == screenMainMenu && m.cursor < len(m.mainMenuItems) && m.mainMenuItems[m.cursor].id == -1 {
		m.cursor = firstSelectableIndex(m.mainMenuItems)
	}
	return m
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case deviceAttachedMsg:
		m = m.refreshDeviceState()
		cmds = append(cmds, waitForDeviceAttached)
		return m, tea.Batch(cmds...)

	case deviceRemovedMsg:
		m = m.refreshDeviceState()
		cmds = append(cmds, waitForDeviceRemoved)
		// If we're on a screen that depends on removed device, go back
		if m.dongle == nil && m.headset == nil && m.screen != screenMainMenu {
			m.screen = screenMainMenu
			m.cursor = 0
		}
		return m, tea.Batch(cmds...)

	case batteryTickMsg:
		if device, exists := deviceManager[selectedHeadset]; exists {
			battery, err := getBatteryStatus(device.deviceID)
			if err == nil {
				device.batteryStatus = battery
			}
		}
		m = m.refreshDeviceState()
		cmds = append(cmds, tickBattery())
		return m, tea.Batch(cmds...)

	case pairingTickMsg:
		if dongle, exists := deviceManager[selectedDongle]; exists {
			if dongle.featureFlags != nil && dongle.featureFlags.pairingList {
				updated := getPairingList(dongle.deviceID)
				dongle.pairingList = updated
			}
		}
		m = m.refreshDeviceState()
		// Refresh paired devices list if we're on that screen
		if m.screen == screenPairedDevices && m.dongle != nil && m.dongle.pairingList != nil {
			m.pairedDevices = m.dongle.pairingList.pairedDevices
		}
		cmds = append(cmds, tickPairing())
		return m, tea.Batch(cmds...)

	case searchTickMsg:
		if m.screen == screenSearchDevices {
			if dongle, exists := deviceManager[selectedDongle]; exists {
				updated := getSearchDeviceList(dongle.deviceID)
				if updated != nil {
					searchDeviceList.count = updated.count
					searchDeviceList.listType = updated.listType
					searchDeviceList.pairedDevices = updated.pairedDevices
				}
			}
			m.searchResults = searchDeviceList.pairedDevices
			cmds = append(cmds, tickSearch())
		}
		return m, tea.Batch(cmds...)

	case headDetectionMsg:
		headDetectionLeft = msg.left
		headDetectionRight = msg.right
		headDetectionSet = true
		m = m.refreshDeviceState()
		cmds = append(cmds, waitForHeadDetection)
		return m, tea.Batch(cmds...)

	case linkQualityMsg:
		linkQualityStatus = msg.status
		linkQualitySet = true
		m = m.refreshDeviceState()
		cmds = append(cmds, waitForLinkQuality)
		return m, tea.Batch(cmds...)

	case firmwareProgressMsg:
		// fwUpdateState is updated by the C callback directly
		if fwUpdateState != nil {
			m.fwState = fwUpdateState
		}
		cmds = append(cmds, waitForFirmwareProgress)
		return m, tea.Batch(cmds...)

	case feedbackDoneMsg:
		m.actionFeedback = -1
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		// Global quit
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		var cmd tea.Cmd
		m, cmd = m.updateScreen(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	}

	return m, nil
}

func (m model) updateScreen(msg tea.KeyMsg) (model, tea.Cmd) {
	switch m.screen {
	case screenMainMenu:
		return m.updateMainMenu(msg)
	case screenSearchDevices:
		return m.updateSearchDevices(msg)
	case screenPairedDevices:
		return m.updatePairedDevices(msg)
	case screenDongleSettings:
		return m.updateDongleSettings(msg)
	case screenSwitchDevice:
		return m.updateSwitchDevice(msg)
	case screenHeadsetSettings:
		return m.updateHeadsetSettings(msg)
	case screenDeviceInfo:
		return m.updateDeviceInfo(msg)
	case screenEqualizer:
		return m.updateEqualizer(msg)
	case screenAudioSettings:
		return m.updateAudioSettings(msg)
	case screenANCSettings:
		return m.updateANCSettings(msg)
	case screenFirmwareUpdate:
		return m.updateFirmwareUpdate(msg)
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	m = m.refreshDeviceState()

	header := renderHeader(m)

	var body string
	switch m.screen {
	case screenMainMenu:
		body = m.viewMainMenu()
	case screenSearchDevices:
		body = m.viewSearchDevices()
	case screenPairedDevices:
		body = m.viewPairedDevices()
	case screenDongleSettings:
		body = m.viewDongleSettings()
	case screenSwitchDevice:
		body = m.viewSwitchDevice()
	case screenHeadsetSettings:
		body = m.viewHeadsetSettings()
	case screenDeviceInfo:
		body = m.viewDeviceInfo()
	case screenEqualizer:
		body = m.viewEqualizer()
	case screenAudioSettings:
		body = m.viewAudioSettings()
	case screenANCSettings:
		body = m.viewANCSettings()
	case screenFirmwareUpdate:
		body = m.viewFirmwareUpdate()
	}

	// Layout: header + box with body
	contentHeight := m.height - 4 // header(1) + padding(1) + footer(1) + padding(1)
	if contentHeight < 3 {
		contentHeight = 3
	}
	contentWidth := m.width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	styledBox := boxStyle.
		Width(contentWidth).
		Height(contentHeight).
		Render(body)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		styledBox,
	)
}

// Helper to trigger action feedback flash
func (m model) triggerFeedback(index int) (model, tea.Cmd) {
	m.actionFeedback = index
	return m, tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return feedbackDoneMsg{}
	})
}
