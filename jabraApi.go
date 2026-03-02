package main

/*
#cgo CFLAGS: -Iheaders
#cgo LDFLAGS: -Llib -ljabra

#include "Common.h"
#include "GoWrapper.h"
#include "JabraDeviceConfig.h"
#include "Interface_AmbienceModes.h"
#include "Interface_Firmware.h"
#include "Interface_Bluetooth.h"
#include "Interface_Constants.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

type jabra_DeviceInfo struct {
	deviceID               uint16
	productID              uint16
	vendorID               uint16
	deviceName             string
	usbDevicePath          string
	parentInstanceId       string
	errStatus              errorStatusCode
	isDongle               bool
	dongleName             string
	variant                string
	serialNumber           string
	isInFirmwareUpdateMode bool
	deviceConnection       deviceConnectionType
	connectionID           uint32
	parentDeviceID         uint16
	deviceEventsMask       uint32
	featureFlags           *featureFlags
	batteryStatus          *batteryStatus
	pairingList            *pairingList
}

type batteryComponent int

const (
	unknown       batteryComponent = iota // Unable to determine the component.
	headband                              // Generally applies to headsets with headband that only contains one battery.
	combinde                              // For headsets that contains multiple batteries but is not capable of sending each individual state.
	right                                 // The battery in the right unit
	left                                  // The battery in the left unit
	cradleBattery                         // The battery in the cradle
	remoteControl                         // The battery in the remote control
)

type batteryStatusUnit struct {
	levelInPercent uint8
	component      batteryComponent
}

type batteryStatus struct {
	levelInPercent  uint8
	charging        bool
	batteryLow      bool
	component       batteryComponent
	extraUnitsCount uint32 // count of extra units
	extraUnits      []batteryStatusUnit
}

type deviceListType int

const (
	searchResult deviceListType = iota
	pairedDevices
	searchComplete
)

type pairedDevice struct {
	deviceName   string
	deviceBTAddr [6]byte
	isConnected  bool
}

type pairingList struct {
	count         uint16
	listType      deviceListType
	pairedDevices []pairedDevice
}

type secureConnectionMode int

const (
	legacyMode     secureConnectionMode = iota // Normal pairing allowed
	secureMode                                 // Device is allowed to connect a audio gateway eg. a mobile phone
	restrictedMode                             // Pairing not allowed
)

type featureFlags struct {
	busyLight                          bool
	factoryReset                       bool
	pairingList                        bool
	remoteMMI                          bool
	musicEqualizer                     bool
	earbudInterconnectionStatus        bool
	stepRate                           bool
	heartRate                          bool
	rrInterval                         bool
	ringtoneUpload                     bool
	imageUpload                        bool
	needsExplicitRebootAfterOta        bool
	needsToBePutIncCradleToCompleteFwu bool
	remoteMMIv2                        bool
	logging                            bool
	preferredSoftphoneListInDevice     bool
	voiceAssistant                     bool
	playRingtone                       bool
	setDateTime                        bool
	fullWizardMode                     bool
	limitedWizardMode                  bool
	onHeadDetection                    bool
	settingsChangeNotification         bool
	audioStreaming                     bool
	customerSupport                    bool
	mySound                            bool
	uiConfigurableButtons              bool
	manualBusyLight                    bool
	whiteboard                         bool
	video                              bool
	ambienceModes                      bool
	sealingTest                        bool
	amasupport                         bool
	ambienceModesLoop                  bool
	ffanc                              bool
	googleBisto                        bool
	virtualDirector                    bool
	pictureInPicture                   bool
	dateTimeIsUTC                      bool
	remoteControl                      bool
	userConfigurableHdr                bool
	dectBasicPairing                   bool
	dectSecurePairing                  bool
	dectOtaFwuSupported                bool
	xpressURL                          bool
	passwordProvisioning               bool
	ethernet                           bool
	wlan                               bool
	ethernetAuthenticationCertificate  bool
	ethernetAuthenticationMschapv2     bool
	wlanAuthenticationCertificate      bool
	wlanAuthenticationMschapv2         bool
}

type hidInput int

const (
	undefined hidInput = iota
	offHook
	mute
	flash
	redial
	key0
	key1
	key2
	key3
	key4
	key5
	key6
	key7
	key8
	key9
	keyStar
	keyPound
	keyClear
	online
	speedDial
	voiceMail
	lineBusy
	rejectCall
	outOfRange
	pseudoOffHook
	button1
	button2
	button3
	volumeUp
	volumeDown
	fireAlarm
	jackConnection
	qdConnection
	headsetConnection
)

type devices map[int]*jabra_DeviceInfo
type deviceConnectionType int

const (
	deviceConnectionType_USB deviceConnectionType = iota
	deviceConnectionType_BT
	deviceConnectionType_DECT
)

type menuItem struct {
	id    int
	label string
}

type equalizerBand struct {
	maxGain         float32
	centerFrequency int
	currentGain     float32
}

// EQ presets: gains in dB for 5 bands (Sub-bass, Bass, Mid, Presence, Brilliance).
// Values are clamped to the device's maxGain when applied.
type eqPreset struct {
	name  string
	gains [5]float32
}

var eqPresets = []eqPreset{
	{name: "Flat", gains: [5]float32{0, 0, 0, 0, 0}},
	{name: "Speech", gains: [5]float32{-2, -1, 1, 3, 1.5}},
	{name: "Bass Boost", gains: [5]float32{4, 2.5, 0, 0, 0}},
	{name: "Treble Boost", gains: [5]float32{0, 0, 0, 2, 4}},
	{name: "Podcast", gains: [5]float32{-3, -1.5, 1.5, 3.5, 1}},
	{name: "Smooth", gains: [5]float32{1, 0.5, 0, -1, -0.5}},
	{name: "Energize", gains: [5]float32{2, 0.5, -1, 1.5, 3}},
	{name: "Deep Bass", gains: [5]float32{6, 4, 1, 0, -1}},
}

func applyEQPreset(deviceID uint16, bands []equalizerBand, presetIdx int) {
	if presetIdx < 0 || presetIdx >= len(eqPresets) || len(bands) == 0 {
		return
	}
	preset := eqPresets[presetIdx]
	gains := make([]float32, len(bands))
	for i := range bands {
		g := float32(0)
		if i < len(preset.gains) {
			g = preset.gains[i]
		}
		// Clamp to device limits
		if g > bands[i].maxGain {
			g = bands[i].maxGain
		}
		if g < -bands[i].maxGain {
			g = -bands[i].maxGain
		}
		gains[i] = g
		bands[i].currentGain = g
	}
	setEqualizerParameters(deviceID, gains)
}

// detectEQPreset checks if current band gains match any known preset.
// Returns preset index or -1 for custom.
func detectEQPreset(bands []equalizerBand) int {
	for idx, preset := range eqPresets {
		match := true
		for i := range bands {
			expected := float32(0)
			if i < len(preset.gains) {
				expected = preset.gains[i]
				if expected > bands[i].maxGain {
					expected = bands[i].maxGain
				}
				if expected < -bands[i].maxGain {
					expected = -bands[i].maxGain
				}
			}
			diff := bands[i].currentGain - expected
			if diff < -0.5 || diff > 0.5 {
				match = false
				break
			}
		}
		if match {
			return idx
		}
	}
	return -1
}

type deviceSetting struct {
	guid    string
	name    string
	current int
	options []string
}

type firmwareProgressEvent struct {
	deviceID                      uint16
	eventType, status, percentage int
}

var (
	// deviceManager
	deviceManager   devices
	selectedHeadset int = -1
	selectedDongle  int = -1

	// Dynamic menu
	startMenu           = []menuItem{}
	dongleSettignsMenu  = []menuItem{}
	headsetSettingsMenu = []menuItem{}

	// holding all the new devide found on BT
	searchDeviceList *pairingList = &pairingList{
		count:         0,
		listType:      searchResult,
		pairedDevices: make([]pairedDevice, 0),
	}

	// Notification channels for C callbacks -> Bubbletea
	// firmwareProgressEvent carries data from the C firmware callback to Bubbletea
	chDeviceAttached   = make(chan *jabra_DeviceInfo, 4)
	chDeviceRemoved    = make(chan uint16, 4)
	chHeadDetection    = make(chan [2]bool, 4)
	chLinkQuality      = make(chan int, 4)
	chFirmwareProgress = make(chan firmwareProgressEvent, 8)

	// Head detection state
	headDetectionLeft  bool
	headDetectionRight bool
	headDetectionSet   bool

	// BT Link Quality state
	linkQualityStatus int // 0=OFF, 1=LOW, 2=HIGH
	linkQualitySet    bool

	// PipeWire audio state
	currentAudioState *audioState
)

type audioProfile struct {
	index       int
	name        string
	description string
}

type audioState struct {
	deviceID      int
	sinkID        int
	sourceID      int
	profiles      []audioProfile
	activeProfile int
	outputVolume  float64
	inputVolume   float64
}

/****************************************************************************/
/*                             C CALLBACKS	                                */
/****************************************************************************/

// Reminder: If you plan to use this function, make sure to update the `goWrapper.h` file accordingly.
// //export firstScanForDevicesDone
// func firstScanForDevicesDone() {
// 	 fmt.Println("First scan for devices done!")
// }

//export deviceAttachedFunc
func deviceAttachedFunc(deviceInfo C.Jabra_DeviceInfo) {

	goDeviceInfo := &jabra_DeviceInfo{
		deviceID:               uint16(deviceInfo.deviceID),
		productID:              uint16(deviceInfo.productID),
		vendorID:               uint16(deviceInfo.vendorID),
		deviceName:             C.GoString(deviceInfo.deviceName),
		usbDevicePath:          C.GoString(deviceInfo.usbDevicePath),
		parentInstanceId:       C.GoString(deviceInfo.parentInstanceId),
		errStatus:              errorStatusCode(deviceInfo.errStatus),
		isDongle:               bool(deviceInfo.isDongle),
		dongleName:             C.GoString(deviceInfo.dongleName),
		variant:                C.GoString(deviceInfo.variant),
		serialNumber:           C.GoString(deviceInfo.serialNumber),
		isInFirmwareUpdateMode: bool(deviceInfo.isInFirmwareUpdateMode),
		deviceConnection:       deviceConnectionType(deviceInfo.deviceconnection),
		connectionID:           uint32(deviceInfo.connectionId),
		parentDeviceID:         uint16(deviceInfo.parentDeviceId),
	}
	goDeviceInfo.deviceEventsMask = getDeviceEventsMask(goDeviceInfo.deviceID)
	goDeviceInfo.featureFlags = getSupportedFeature(goDeviceInfo.deviceID)

	if !goDeviceInfo.isDongle {
		battery, err := getBatteryStatus(goDeviceInfo.deviceID)
		if err == nil {
			goDeviceInfo.batteryStatus = battery
		}
		if goDeviceInfo.featureFlags.onHeadDetection {
			C.Jabra_SetHeadDetectionStatusListener(
				C.ushort(goDeviceInfo.deviceID),
				(C.HeadDetectionStatusListener)(C.headDetectionStatusFunc),
			)
		}
		// Register link quality listener (silently ignores Not_Supported)
		C.Jabra_SetLinkQualityStatusListener(
			C.ushort(goDeviceInfo.deviceID),
			(C.LinkQualityStatusListener)(C.linkQualityStatusFunc),
		)
	} else {
		if goDeviceInfo.featureFlags.pairingList {
			goDeviceInfo.pairingList = getPairingList(goDeviceInfo.deviceID)
		}
	}

	if isNewDevice := serialNumberCheck(goDeviceInfo); isNewDevice {
		deviceManager.add(goDeviceInfo)
	}
	C.Jabra_FreeDeviceInfo(deviceInfo)

	select {
	case chDeviceAttached <- goDeviceInfo:
	default:
	}
}

//export deviceRemovedFunc
func deviceRemovedFunc(deviceID uint16) {
	deviceManager.removed(deviceID)

	select {
	case chDeviceRemoved <- deviceID:
	default:
	}
}

//export headDetectionStatusFunc
func headDetectionStatusFunc(deviceID C.ushort, status C.HeadDetectionStatus) {
	headDetectionLeft = bool(status.leftOn)
	headDetectionRight = bool(status.rightOn)
	headDetectionSet = true

	select {
	case chHeadDetection <- [2]bool{headDetectionLeft, headDetectionRight}:
	default:
	}
}

//export linkQualityStatusFunc
func linkQualityStatusFunc(deviceID C.ushort, status C.LinkQuality) {
	linkQualityStatus = int(status)
	linkQualitySet = true

	select {
	case chLinkQuality <- linkQualityStatus:
	default:
	}
}

// //export buttonInDataRawHidFunc
// func buttonInDataRawHidFunc(deviceid uint16, usagepage uint16, usage uint16, buttonindata bool) {
// 	// log.Println("RawHid", deviceid, usagepage, usage, buttonindata)

// }

// //export buttonInDataTranslatedFunc
// func buttonInDataTranslatedFunc(deviceid uint16, jabra_HidInput hidInput, buttonindata bool) {
// 	// log.Println("Translated", deviceid, jabra_HidInput, buttonindata)
// 	// if jabra_HidInput == headsetConnection {
// 	// 	fmt.Println("headsetConnection")
// 	// }
// }


// Reminder: If you plan to use this function, make sure to update the `goWrapper.h` file accordingly.
// The current callback behavior is inconsistent. While the charging status updates as expected,
// the `levelInPercent` callback is sometimes delayed.
// //export batteryStatusUpdate
// func batteryStatusUpdate(deviceID uint16, cBatteryStatus *C.Jabra_BatteryStatus) {

// 	for _, device := range deviceManager {
// 		if device.deviceID == deviceID {
// 			// A shallow copy is not used here because this can be slower due to the extra allocation,
// 			// and modifying the pointer directly is faster than copying the entire structure.
// 			device.batteryStatus.levelInPercent = uint8(cBatteryStatus.levelInPercent)
// 			device.batteryStatus.charging = bool(cBatteryStatus.charging)
// 			device.batteryStatus.batteryLow = bool(cBatteryStatus.batteryLow)
// 			device.batteryStatus.component = batteryComponent(cBatteryStatus.component)
// 			device.batteryStatus.extraUnitsCount = uint32(cBatteryStatus.extraUnitsCount)

// 			if cBatteryStatus.extraUnitsCount > 0 && cBatteryStatus.extraUnits != nil {
// 				clear(device.batteryStatus.extraUnits)
// 				extraUnits := (*[1 << 30]C.Jabra_BatteryStatusUnit)(unsafe.Pointer(cBatteryStatus.extraUnits))[:cBatteryStatus.extraUnitsCount:cBatteryStatus.extraUnitsCount]
// 				for _, unit := range extraUnits {
// 					device.batteryStatus.extraUnits = append(device.batteryStatus.extraUnits, batteryStatusUnit{
// 						levelInPercent: uint8(unit.levelInPercent),
// 						component:      batteryComponent(unit.component),
// 					})
// 				}
// 			}
// 		}
// 	}

// 	C.Jabra_FreeBatteryStatus(cBatteryStatus)
// }

/****************************************************************************/
/*                           GENERAL UTILITES                               */
/****************************************************************************/

func updateDongleSettignsMenu() {
	dongleSettignsMenu = []menuItem{}

	if dongle, exists := deviceManager[selectedDongle]; exists {
		getautoPairingState, _ := getAutoPairing()
		if getautoPairingState {
			dongleSettignsMenu = append(dongleSettignsMenu, menuItem{id: 0, label: "AutoPairing ON"})
		} else {
			dongleSettignsMenu = append(dongleSettignsMenu, menuItem{id: 0, label: "AutoPairing OFF"})
		}

		if dongle.featureFlags.factoryReset {
			dongleSettignsMenu = append(dongleSettignsMenu, menuItem{id: 1, label: "Factory Reset"})
		}
	}
}

// menuItem id=-1 is a non-selectable separator/section header
func updateStartMenu() {
	startMenu = []menuItem{}

	if dongle, dongleexists := deviceManager[selectedDongle]; dongleexists {
		// -- Bluetooth section --
		startMenu = append(startMenu, menuItem{id: -1, label: "--- Bluetooth ---"})
		startMenu = append(startMenu, menuItem{id: 0, label: "Search For New Devices"})
		if dongle.featureFlags.pairingList && dongle.pairingList.count != 0 {
			startMenu = append(startMenu, menuItem{id: 1, label: "Paired Devices"})
		}

		// -- Settings section --
		startMenu = append(startMenu, menuItem{id: -1, label: "--- Settings ---"})
		startMenu = append(startMenu, menuItem{id: 2, label: fmt.Sprintf("%s Settings", dongle.deviceName)})
	}

	if device, deviceexists := deviceManager[selectedHeadset]; deviceexists {
		startMenu = append(startMenu, menuItem{id: 4, label: fmt.Sprintf("%s Settings", device.deviceName)})
	}

	if currentAudioState == nil {
		currentAudioState = discoverPipeWireDevice()
	}
	if currentAudioState != nil {
		startMenu = append(startMenu, menuItem{id: 7, label: "Audio Settings"})
	}

	if len(deviceManager) > 0 {
		// -- Info section --
		startMenu = append(startMenu, menuItem{id: -1, label: "--- Info ---"})
		startMenu = append(startMenu, menuItem{id: 6, label: "Device Info"})
	}

	startMenu = append(startMenu, menuItem{id: -1, label: ""})
	startMenu = append(startMenu, menuItem{id: 5, label: "Exit"})
}

func serialNumberCheck(deviceInfo *jabra_DeviceInfo) bool {
	if deviceInfo.serialNumber == "" {
		return false
	}

	var isNewDevice bool

	if deviceInfo.isDongle {
		if dongle, exists := deviceManager[selectedDongle]; exists {
			if dongle.serialNumber == deviceInfo.serialNumber && dongle.deviceConnection == deviceInfo.deviceConnection {
				dongle.deviceID = deviceInfo.deviceID
				isNewDevice = false
			} else {
				isNewDevice = true
			}
		} else {
			if selectedDongle == -1 {
				isNewDevice = true
			} else {
				panic("what is going on")
			}
		}
	} else {

		if device, exists := deviceManager[selectedHeadset]; exists {
			if device.serialNumber == deviceInfo.serialNumber && device.deviceConnection == deviceInfo.deviceConnection {
				device.deviceID = deviceInfo.deviceID
				isNewDevice = false
			} else {

				isNewDevice = true

			}
		} else {
			if selectedHeadset == -1 {
				isNewDevice = true
			} else {
				panic("what is going on")
			}
		}

	}

	return isNewDevice
}

func (d *devices) add(deviceInfo *jabra_DeviceInfo) {
	if *d == nil {
		*d = make(map[int]*jabra_DeviceInfo)
	}

	id := len(*d)
	if deviceInfo.isDongle {
		if selectedDongle == -1 {
			selectedDongle = id
		}
	} else {
		if selectedHeadset == -1 {
			selectedHeadset = id
		} else if existing, ok := (*d)[selectedHeadset]; ok &&
			existing.deviceConnection == deviceConnectionType_USB &&
			deviceInfo.deviceConnection == deviceConnectionType_BT {
			// Prefer BT-connected device (actual headset) over USB (deskstand/dock)
			selectedHeadset = id
		}
	}

	(*d)[id] = deviceInfo
	updateStartMenu()
	updateDongleSettignsMenu()
}

func (d *devices) removed(deviceID uint16) {
	if *d == nil {
		return
	}

	var (
		checkDongleExists  bool
		checkHeadSetExists bool
	)

	newDevices := make(map[int]*jabra_DeviceInfo)

	nextIndex := 0
	for i := 0; i < len(*d); i++ {
		device, exists := (*d)[i]
		if !exists || device.deviceID == deviceID {
			continue
		}

		if device.isDongle {
			checkDongleExists = true
			selectedDongle = nextIndex
		} else {
			checkHeadSetExists = true
			selectedHeadset = nextIndex
		}

		newDevices[nextIndex] = device
		nextIndex++
	}
	if !checkDongleExists {
		selectedDongle = -1
	}
	if !checkHeadSetExists {
		selectedHeadset = -1
		linkQualitySet = false
	}

	*d = newDevices
	updateStartMenu()
}

func uninitialize() {
	if uninit := C.Jabra_Uninitialize(); !uninit {
		fmt.Println("Failed Uninitialize")
	}
}

func factoryReset(deviceID uint16) error {
	if err := returnCode(int(C.Jabra_FactoryReset(C.ushort(deviceID)))); err != nil {
		return err
	}
	return nil
}

func getJabraSdkVersion() string {
	const bufferSize = 16

	buffer := make([]byte, bufferSize)
	cBuffer := (*C.char)(unsafe.Pointer(&buffer[0]))
	err := returnCode(int(C.Jabra_GetVersion(cBuffer, C.int(bufferSize))))
	if err != nil {
		log.Fatalln(err)
	}

	return C.GoString(cBuffer)
}

func getSupportedFeature(deviceID uint16) *featureFlags {

	var featureFlag featureFlags
	var count C.uint32_t

	cFeatures := C.Jabra_GetSupportedFeatures(C.ushort(deviceID), &count)
	if cFeatures == nil {
		return &featureFlag
	}
	features := (*[1 << 30]uint32)(unsafe.Pointer(cFeatures))[:count:count]

	for _, cFeature := range features {
		switch cFeature {
		case 1000:
			featureFlag.busyLight = true
		case 1001:
			featureFlag.factoryReset = true
		case 1002:
			featureFlag.pairingList = true
		case 1003:
			featureFlag.remoteMMI = true
		case 1004:
			featureFlag.musicEqualizer = true
		case 1005:
			featureFlag.earbudInterconnectionStatus = true
		case 1006:
			featureFlag.stepRate = true
		case 1007:
			featureFlag.heartRate = true
		case 1008:
			featureFlag.rrInterval = true
		case 1009:
			featureFlag.ringtoneUpload = true
		case 1010:
			featureFlag.imageUpload = true
		case 1011:
			featureFlag.needsExplicitRebootAfterOta = true
		case 1012:
			featureFlag.needsToBePutIncCradleToCompleteFwu = true
		case 1013:
			featureFlag.remoteMMIv2 = true
		case 1014:
			featureFlag.logging = true
		case 1015:
			featureFlag.preferredSoftphoneListInDevice = true
		case 1016:
			featureFlag.voiceAssistant = true
		case 1017:
			featureFlag.playRingtone = true
		case 1018:
			featureFlag.setDateTime = true
		case 1019:
			featureFlag.fullWizardMode = true
		case 1020:
			featureFlag.limitedWizardMode = true
		case 1021:
			featureFlag.onHeadDetection = true
		case 1022:
			featureFlag.settingsChangeNotification = true
		case 1023:
			featureFlag.audioStreaming = true
		case 1024:
			featureFlag.customerSupport = true
		case 1025:
			featureFlag.mySound = true
		case 1026:
			featureFlag.uiConfigurableButtons = true
		case 1027:
			featureFlag.manualBusyLight = true
		case 1028:
			featureFlag.whiteboard = true
		case 1029:
			featureFlag.video = true
		case 1030:
			featureFlag.ambienceModes = true
		case 1031:
			featureFlag.sealingTest = true
		case 1032:
			featureFlag.amasupport = true
		case 1033:
			featureFlag.ambienceModesLoop = true
		case 1034:
			featureFlag.ffanc = true
		case 1035:
			featureFlag.googleBisto = true
		case 1036:
			featureFlag.virtualDirector = true
		case 1037:
			featureFlag.pictureInPicture = true
		case 1038:
			featureFlag.dateTimeIsUTC = true
		case 1039:
			featureFlag.remoteControl = true
		case 1040:
			featureFlag.userConfigurableHdr = true
		case 1041:
			featureFlag.dectBasicPairing = true
		case 1042:
			featureFlag.dectSecurePairing = true
		case 1043:
			featureFlag.dectOtaFwuSupported = true
		case 1044:
			featureFlag.xpressURL = true
		case 1045:
			featureFlag.passwordProvisioning = true
		case 1046:
			featureFlag.ethernet = true
		case 1047:
			featureFlag.wlan = true
		case 1048:
			featureFlag.ethernetAuthenticationCertificate = true
		case 1049:
			featureFlag.ethernetAuthenticationMschapv2 = true
		case 1050:
			featureFlag.wlanAuthenticationCertificate = true
		case 1051:
			featureFlag.wlanAuthenticationMschapv2 = true
		}
	}
	// Free features array
	C.Jabra_FreeSupportedFeatures(cFeatures)

	return &featureFlag
}

func getDeviceEventsMask(deviceID uint16) uint32 {
	return uint32(C.Jabra_GetSupportedDeviceEvents(C.ushort(deviceID)))
}

/****************************************************************************/
/*                               BLUETOOTH                                  */
/****************************************************************************/

func searchForNewDevices() error {
	if err := setDongleInBTPairing(true); err != nil {
		return err
	}

	if dongle, exists := deviceManager[selectedDongle]; exists { // it take 20 Sec
		if err := returnCode(int(C.Jabra_SearchNewDevices(C.ushort(dongle.deviceID)))); err != nil {
			return err
		}
	}
	return nil
}

func setDongleInBTPairing(pairing bool) error {

	if dongle, exists := deviceManager[selectedDongle]; exists {
		if pairing {
			if err := returnCode(int(C.Jabra_SetBTPairing(C.ushort(dongle.deviceID)))); err != nil {
				return err
			}
		} else {
			if err := returnCode(int(C.Jabra_StopBTPairing(C.ushort(dongle.deviceID)))); err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("no dongle found")
	}

	return nil
}

func connectNewDevice(pairingID uint16) error {
	var returnErr error

	if searchDeviceList.count != 0 {
		cDevices := make([]C.Jabra_PairedDevice, len(searchDeviceList.pairedDevices))

		for i, device := range searchDeviceList.pairedDevices {
			cDevices[i].deviceName = C.CString(device.deviceName)
			cDevices[i].isConnected = C.bool(device.isConnected)

			for j := 0; j < 6; j++ {
				cDevices[i].deviceBTAddr[j] = C.uint8_t(device.deviceBTAddr[j])
			}
		}

		if err := returnCode(int(C.Jabra_ConnectNewDevice(C.ushort(pairingID), (*C.Jabra_PairedDevice)(unsafe.Pointer(&cDevices[0]))))); err != nil {
			returnErr = err
		}

		if err := setDongleInBTPairing(false); err != nil {
			returnErr = err
		}

		for _, cDevice := range cDevices {
			C.free(unsafe.Pointer(cDevice.deviceName))
		}
	}

	return returnErr

}

func getSearchDeviceList(deviceID uint16) *pairingList {
	var searchDeviceList *pairingList

	cPairingList := C.Jabra_GetSearchDeviceList(C.ushort(deviceID))
	if cPairingList != nil {
		searchDeviceList = &pairingList{
			count:    uint16(cPairingList.count),
			listType: deviceListType(cPairingList.listType),
		}

		// safely cast a pointer to an array to a slice of a potentially unknown or large size
		// with a length and capacity limited to the actual number of items.
		// Similar to how in Go you might do: arr := [100]int{}; slice := arr[:10:10]
		// This creates a slice that references the first 10 elements with a capacity of 10
		cPairedDevices := (*[1 << 30]C.Jabra_PairedDevice)(unsafe.Pointer(cPairingList.pairedDevice))[:cPairingList.count:cPairingList.count]
		for _, cDevice := range cPairedDevices {
			bTDevice := pairedDevice{
				deviceName:  C.GoString(cDevice.deviceName),
				isConnected: bool(cDevice.isConnected),
			}
			copy(bTDevice.deviceBTAddr[:], C.GoBytes(unsafe.Pointer(&cDevice.deviceBTAddr[0]), 6))
			searchDeviceList.pairedDevices = append(searchDeviceList.pairedDevices, bTDevice)
		}

		C.Jabra_FreePairingList(cPairingList)
	}

	return searchDeviceList

}

func getPairingList(deviceID uint16) *pairingList {
	var createPairingList *pairingList

	cPairingList := C.Jabra_GetPairingList(C.ushort(deviceID))
	if cPairingList == nil {
		return &pairingList{
			count:         0,
			listType:      -1,
			pairedDevices: make([]pairedDevice, 0),
		}
	}
	createPairingList = &pairingList{
		count:    uint16(cPairingList.count),
		listType: deviceListType(cPairingList.listType),
	}

	cPairedDevices := (*[1 << 30]C.Jabra_PairedDevice)(unsafe.Pointer(cPairingList.pairedDevice))[:cPairingList.count:cPairingList.count]
	for _, cDevice := range cPairedDevices {
		bTDevice := pairedDevice{
			deviceName:  C.GoString(cDevice.deviceName),
			isConnected: bool(cDevice.isConnected),
		}
		copy(bTDevice.deviceBTAddr[:], C.GoBytes(unsafe.Pointer(&cDevice.deviceBTAddr[0]), 6))
		createPairingList.pairedDevices = append(createPairingList.pairedDevices, bTDevice)
	}
	C.Jabra_FreePairingList(cPairingList)

	return createPairingList
}

// Clear the pairingList
func clearPairingList() error {

	if dongle, exists := deviceManager[selectedDongle]; exists {
		if err := returnCode(int(C.Jabra_ClearPairingList(C.ushort(dongle.deviceID)))); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("no dongle found")
	}

	return nil
}

// Remove device from pairingList where deviceListType is pairedDevices
func removeDeviceFromPairedlist(pairingID uint16) error {

	var reterr error

	if dongle, exists := deviceManager[selectedDongle]; exists {
		if dongle.pairingList != nil {
			cDevices := make([]C.Jabra_PairedDevice, len(dongle.pairingList.pairedDevices))

			for i, device := range dongle.pairingList.pairedDevices {
				cDevices[i].deviceName = C.CString(device.deviceName)
				cDevices[i].isConnected = C.bool(device.isConnected)

				for j := 0; j < 6; j++ {
					cDevices[i].deviceBTAddr[j] = C.uint8_t(device.deviceBTAddr[j])
				}
			}

			if err := returnCode(int(C.Jabra_ClearPairedDevice(C.ushort(pairingID), (*C.Jabra_PairedDevice)(unsafe.Pointer(&cDevices[0]))))); err != nil {
				reterr = err
			}

			for _, cDevice := range cDevices {
				C.free(unsafe.Pointer(cDevice.deviceName))
			}
		}
	} else {
		return fmt.Errorf("no dongle found")
	}

	return reterr
}

// Connect from pairingList where deviceListType is pairedDevices
func connectDeviceFromPairedlist(pairingID uint16) error {

	var reterr error

	if dongle, exists := deviceManager[selectedDongle]; exists {
		if dongle.pairingList != nil {
			cDevices := make([]C.Jabra_PairedDevice, len(dongle.pairingList.pairedDevices))

			for i, device := range dongle.pairingList.pairedDevices {
				cDevices[i].deviceName = C.CString(device.deviceName)
				cDevices[i].isConnected = C.bool(device.isConnected)

				for j := 0; j < 6; j++ {
					cDevices[i].deviceBTAddr[j] = C.uint8_t(device.deviceBTAddr[j])
				}
			}

			if err := returnCode(int(C.Jabra_ConnectPairedDevice(C.ushort(pairingID), (*C.Jabra_PairedDevice)(unsafe.Pointer(&cDevices[0]))))); err != nil {
				reterr = err
			}

			for _, cDevice := range cDevices {
				C.free(unsafe.Pointer(cDevice.deviceName))
			}
		}
	} else {
		return fmt.Errorf("no dongle found")
	}

	return reterr
}

// Disconnect from pairingList where deviceListType is pairedDevices
func disconnectDeviceFromPairedlist(pairingID uint16) error {

	var returnErr error

	if dongle, exists := deviceManager[selectedDongle]; exists {
		if dongle.pairingList != nil {
			cDevices := make([]C.Jabra_PairedDevice, len(dongle.pairingList.pairedDevices))

			for i, device := range dongle.pairingList.pairedDevices {
				cDevices[i].deviceName = C.CString(device.deviceName)
				cDevices[i].isConnected = C.bool(device.isConnected)

				for j := 0; j < 6; j++ {
					cDevices[i].deviceBTAddr[j] = C.uint8_t(device.deviceBTAddr[j])
				}
			}

			if err := returnCode(int(C.Jabra_DisConnectPairedDevice(C.ushort(pairingID), (*C.Jabra_PairedDevice)(unsafe.Pointer(&cDevices[0]))))); err != nil {
				returnErr = err
			}

			for _, cDevice := range cDevices {
				C.free(unsafe.Pointer(cDevice.deviceName))
			}
		}
	} else {
		return fmt.Errorf("no dongle found")
	}

	return returnErr
}

func reconnectToDevice() error {
	if dongle, exists := deviceManager[selectedDongle]; exists {
		if err := returnCode(int(C.Jabra_ConnectBTDevice(C.ushort(dongle.deviceID)))); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("no dongle found")
	}

	return nil
}

func disconnectBTDeviceFromDongle() error {

	if dongle, exists := deviceManager[selectedDongle]; exists {
		if err := returnCode(int(C.Jabra_DisconnectBTDevice(C.ushort(dongle.deviceID)))); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("no dongle found")
	}

	return nil
}

func getAutoPairing() (bool, error) {
	if dongle, exists := deviceManager[selectedDongle]; exists {
		return bool(C.Jabra_GetAutoPairing(C.ushort(dongle.deviceID))), nil
	}

	return false, fmt.Errorf("no dongle")
}

func setAutoPairing(autoPairing bool) error {
	if dongle, exists := deviceManager[selectedDongle]; exists {
		if err := returnCode(int(C.Jabra_SetAutoPairing(C.ushort(dongle.deviceID), C.bool(autoPairing)))); err != nil {
			return err
		}
	}
	return nil
}

/****************************************************************************/
/*                             BATTERY STATUS                               */
/****************************************************************************/

func getBatteryStatus(deviceID uint16) (*batteryStatus, error) {
	var cBatteryStatus *C.Jabra_BatteryStatus

	if err := returnCode(int(C.Jabra_GetBatteryStatusV2(C.ushort(deviceID), &cBatteryStatus))); err != nil {
		return nil, err
	}

	goBatteryStatus := &batteryStatus{
		levelInPercent:  uint8(cBatteryStatus.levelInPercent),
		charging:        bool(cBatteryStatus.charging),
		batteryLow:      bool(cBatteryStatus.batteryLow),
		component:       batteryComponent(cBatteryStatus.component),
		extraUnitsCount: uint32(cBatteryStatus.extraUnitsCount),
	}

	if cBatteryStatus.extraUnitsCount > 0 && cBatteryStatus.extraUnits != nil {
		extraUnits := (*[1 << 30]C.Jabra_BatteryStatusUnit)(unsafe.Pointer(cBatteryStatus.extraUnits))[:cBatteryStatus.extraUnitsCount:cBatteryStatus.extraUnitsCount]
		for _, unit := range extraUnits {
			goBatteryStatus.extraUnits = append(goBatteryStatus.extraUnits, batteryStatusUnit{
				levelInPercent: uint8(unit.levelInPercent),
				component:      batteryComponent(unit.component),
			})
		}
	}

	C.Jabra_FreeBatteryStatus(cBatteryStatus)

	return goBatteryStatus, nil
}

/****************************************************************************/
/*                            HEADSET SETTINGS                              */
/****************************************************************************/

func updateHeadsetSettingsMenu() {
	headsetSettingsMenu = []menuItem{}

	device, exists := deviceManager[selectedHeadset]
	if !exists {
		return
	}

	// Try the API directly — some devices support ambience modes without
	// reporting the feature flag (e.g. Evolve2 85).
	if supportedModes, err := getSupportedAmbienceModes(device.deviceID); err == nil && len(supportedModes) > 0 {
		mode, err := getAmbienceMode(device.deviceID)
		modeLabel := "ANC Mode: OFF"
		if err == nil {
			switch mode {
			case 1:
				modeLabel = "ANC Mode: HearThrough"
			case 2:
				modeLabel = "ANC Mode: ANC"
			}
		}
		headsetSettingsMenu = append(headsetSettingsMenu, menuItem{id: 0, label: modeLabel})
	}

	if device.featureFlags.musicEqualizer && isEqualizerSupported(device.deviceID) {
		headsetSettingsMenu = append(headsetSettingsMenu, menuItem{id: 1, label: "Equalizer"})
	}

	sidetone := findDeviceSetting(device.deviceID, "sidetone")
	if sidetone != nil && len(sidetone.options) > 0 {
		currentLabel := "Unknown"
		if sidetone.current >= 0 && sidetone.current < len(sidetone.options) {
			currentLabel = sidetone.options[sidetone.current]
		}
		headsetSettingsMenu = append(headsetSettingsMenu, menuItem{id: 2, label: fmt.Sprintf("Sidetone: %s", currentLabel)})
	}

	if device.featureFlags.busyLight || device.featureFlags.manualBusyLight {
		status := getBusyLightStatus(device.deviceID, device.featureFlags)
		label := "Busy Light: OFF"
		if status {
			label = "Busy Light: ON"
		}
		headsetSettingsMenu = append(headsetSettingsMenu, menuItem{id: 3, label: label})
	}
}

/****************************************************************************/
/*                              BUSY LIGHT                                  */
/****************************************************************************/

func getBusyLightStatus(deviceID uint16, flags *featureFlags) bool {
	if flags.manualBusyLight {
		return bool(C.Jabra_GetManualBusylightStatus(C.ushort(deviceID)))
	}
	return bool(C.Jabra_GetBusylightStatus(C.ushort(deviceID)))
}

func setBusyLightStatus(deviceID uint16, on bool, flags *featureFlags) error {
	if flags.manualBusyLight {
		val := C.BUSYLIGHT_OFF
		if on {
			val = C.BUSYLIGHT_ON
		}
		return returnCode(int(C.Jabra_SetManualBusylightStatus(C.ushort(deviceID), C.BusyLightValue(val))))
	}
	return returnCode(int(C.Jabra_SetBusylightStatus(C.ushort(deviceID), C.bool(on))))
}

/****************************************************************************/
/*                           AMBIENCE MODES (ANC)                           */
/****************************************************************************/

func getAmbienceMode(deviceID uint16) (int, error) {
	var mode C.Jabra_AmbienceMode
	if err := returnCode(int(C.Jabra_GetAmbienceMode(C.ushort(deviceID), &mode))); err != nil {
		return 0, err
	}
	return int(mode), nil
}

func setAmbienceMode(deviceID uint16, mode int) error {
	return returnCode(int(C.Jabra_SetAmbienceMode(C.ushort(deviceID), C.Jabra_AmbienceMode(mode))))
}

func getSupportedAmbienceModes(deviceID uint16) ([]int, error) {
	var length C.size_t = 10
	modes := make([]C.Jabra_AmbienceMode, length)
	if err := returnCode(int(C.Jabra_GetSupportedAmbienceModes(C.ushort(deviceID), &modes[0], &length))); err != nil {
		return nil, err
	}
	result := make([]int, int(length))
	for i := 0; i < int(length); i++ {
		result[i] = int(modes[i])
	}
	return result, nil
}

func getSupportedAmbienceModeLevels(deviceID uint16, mode int) (uint8, error) {
	var levels C.uint8_t
	if err := returnCode(int(C.Jabra_GetSupportedAmbienceModeLevels(C.ushort(deviceID), C.Jabra_AmbienceMode(mode), &levels))); err != nil {
		return 0, err
	}
	return uint8(levels), nil
}

func getAmbienceModeLevel(deviceID uint16, mode int) (uint8, error) {
	var level C.uint8_t
	if err := returnCode(int(C.Jabra_GetAmbienceModeLevel(C.ushort(deviceID), C.Jabra_AmbienceMode(mode), &level))); err != nil {
		return 0, err
	}
	return uint8(level), nil
}

func setAmbienceModeLevel(deviceID uint16, mode int, level uint8) error {
	return returnCode(int(C.Jabra_SetAmbienceModeLevel(C.ushort(deviceID), C.Jabra_AmbienceMode(mode), C.uint8_t(level))))
}

func getAmbienceModeLoop(deviceID uint16) ([]int, error) {
	var length C.size_t = 10
	modes := make([]C.Jabra_AmbienceMode, length)
	if err := returnCode(int(C.Jabra_GetAmbienceModeLoop(C.ushort(deviceID), &modes[0], &length))); err != nil {
		return nil, err
	}
	result := make([]int, int(length))
	for i := 0; i < int(length); i++ {
		result[i] = int(modes[i])
	}
	return result, nil
}

func setAmbienceModeLoop(deviceID uint16, modes []int) error {
	if len(modes) == 0 {
		return returnCode(int(C.Jabra_SetAmbienceModeLoop(C.ushort(deviceID), nil, 0)))
	}
	cModes := make([]C.Jabra_AmbienceMode, len(modes))
	for i, m := range modes {
		cModes[i] = C.Jabra_AmbienceMode(m)
	}
	return returnCode(int(C.Jabra_SetAmbienceModeLoop(C.ushort(deviceID), &cModes[0], C.size_t(len(cModes)))))
}

type ancScreenState struct {
	supportedModes []int
	currentMode    int
	maxLevels      map[int]uint8 // mode -> max level (0 = no levels)
	currentLevels  map[int]uint8 // mode -> current level
	loopModes      []int
	loopSupported  bool
}

var currentANCState *ancScreenState

func initANCScreenState(deviceID uint16) {
	supportedModes, err := getSupportedAmbienceModes(deviceID)
	if err != nil {
		return
	}
	currentMode, _ := getAmbienceMode(deviceID)

	state := &ancScreenState{
		supportedModes: supportedModes,
		currentMode:    currentMode,
		maxLevels:      make(map[int]uint8),
		currentLevels:  make(map[int]uint8),
	}

	for _, m := range supportedModes {
		maxLvl, err := getSupportedAmbienceModeLevels(deviceID, m)
		if err == nil && maxLvl > 0 {
			state.maxLevels[m] = maxLvl
			curLvl, err := getAmbienceModeLevel(deviceID, m)
			if err == nil {
				state.currentLevels[m] = curLvl
			}
		}
	}

	// Try the loop API directly — don't rely only on the feature flag
	loopModes, err := getAmbienceModeLoop(deviceID)
	if err == nil {
		state.loopSupported = true
		state.loopModes = loopModes
	}

	currentANCState = state
}

func toggleModeInLoop(mode int) {
	if currentANCState == nil {
		return
	}
	for i, m := range currentANCState.loopModes {
		if m == mode {
			currentANCState.loopModes = append(currentANCState.loopModes[:i], currentANCState.loopModes[i+1:]...)
			return
		}
	}
	currentANCState.loopModes = append(currentANCState.loopModes, mode)
}

func ambienceModeName(mode int) string {
	switch mode {
	case 0:
		return "OFF"
	case 1:
		return "HearThrough"
	case 2:
		return "ANC"
	default:
		return fmt.Sprintf("Mode %d", mode)
	}
}

/****************************************************************************/
/*                               EQUALIZER                                  */
/****************************************************************************/

func isEqualizerSupported(deviceID uint16) bool {
	return bool(C.Jabra_IsEqualizerSupported(C.ushort(deviceID)))
}

func getEqualizerParameters(deviceID uint16) ([]equalizerBand, error) {
	var nbands C.uint = 10
	bands := make([]C.Jabra_EqualizerBand, nbands)
	if err := returnCode(int(C.Jabra_GetEqualizerParameters(C.ushort(deviceID), &bands[0], &nbands))); err != nil {
		return nil, err
	}
	result := make([]equalizerBand, int(nbands))
	for i := 0; i < int(nbands); i++ {
		result[i] = equalizerBand{
			maxGain:         float32(bands[i].max_gain),
			centerFrequency: int(bands[i].centerFrequency),
			currentGain:     float32(bands[i].currentGain),
		}
	}
	return result, nil
}

func setEqualizerParameters(deviceID uint16, gains []float32) error {
	cGains := make([]C.float, len(gains))
	for i, g := range gains {
		cGains[i] = C.float(g)
	}
	return returnCode(int(C.Jabra_SetEqualizerParameters(C.ushort(deviceID), &cGains[0], C.uint(len(cGains)))))
}

/****************************************************************************/
/*                             DEVICE INFO                                  */
/****************************************************************************/

func getConnectedBTDeviceName(deviceID uint16) string {
	cName := C.Jabra_GetConnectedBTDeviceName(C.ushort(deviceID))
	if cName == nil {
		return ""
	}
	name := C.GoString(cName)
	C.Jabra_FreeString(cName)
	return name
}

func getSecureConnectionMode(deviceID uint16) string {
	var mode C.Jabra_SecureConnectionMode
	if err := returnCode(int(C.Jabra_GetSecureConnectionMode(C.ushort(deviceID), &mode))); err != nil {
		return ""
	}
	switch int(mode) {
	case 0:
		return "Legacy"
	case 1:
		return "Secure"
	case 2:
		return "Restricted"
	default:
		return fmt.Sprintf("Unknown (%d)", int(mode))
	}
}

func getDeviceConstantLines(deviceID uint16) []string {
	var lines []string
	constants := C.Jabra_GetConstants(C.ushort(deviceID))
	if constants == nil {
		return lines
	}
	defer C.Jabra_ReleaseConst(constants)

	keys := []struct {
		key   string
		label string
	}{
		{"bluetooth_address", "BT Address"},
		{"model_name", "Model"},
		{"vendor_name", "Vendor"},
		{"product_name", "Product"},
	}

	for _, k := range keys {
		cKey := C.CString(k.key)
		val := C.Jabra_GetConst(constants, cKey)
		C.free(unsafe.Pointer(cKey))
		if val == nil {
			continue
		}
		if C.Jabra_IsString(val) {
			str := C.GoString(C.Jabra_AsString(val))
			if str != "" {
				lines = append(lines, fmt.Sprintf("  %-10s %s", k.label+":", str))
			}
		}
	}

	return lines
}

func checkFirmwareUpdate(deviceID uint16) (bool, string) {
	rc := int(C.Jabra_CheckForFirmwareUpdate(C.ushort(deviceID), nil))
	if rc != 17 { // 17 = Firmware_Available
		return false, ""
	}
	fwInfo := C.Jabra_GetLatestFirmwareInformation(C.ushort(deviceID), nil)
	if fwInfo == nil {
		return true, ""
	}
	version := C.GoString(fwInfo.version)
	C.Jabra_FreeFirmwareInfo(fwInfo)
	return true, version
}

func getFirmwareVersion(deviceID uint16) string {
	const bufferSize = 64
	buffer := make([]byte, bufferSize)
	cBuffer := (*C.char)(unsafe.Pointer(&buffer[0]))
	if err := returnCode(int(C.Jabra_GetFirmwareVersion(C.ushort(deviceID), cBuffer, C.int(bufferSize)))); err != nil {
		return ""
	}
	return C.GoString(cBuffer)
}

func getESN(deviceID uint16) string {
	const bufferSize = 64
	buffer := make([]byte, bufferSize)
	cBuffer := (*C.char)(unsafe.Pointer(&buffer[0]))
	if err := returnCode(int(C.Jabra_GetESN(C.ushort(deviceID), cBuffer, C.int(bufferSize)))); err != nil {
		return ""
	}
	return C.GoString(cBuffer)
}

func getSku(deviceID uint16) string {
	const bufferSize = 64
	buffer := make([]byte, bufferSize)
	cBuffer := (*C.char)(unsafe.Pointer(&buffer[0]))
	if err := returnCode(int(C.Jabra_GetSku(C.ushort(deviceID), cBuffer, C.uint(bufferSize)))); err != nil {
		return ""
	}
	return C.GoString(cBuffer)
}

type fwUpdatePhase int

const (
	fwPhaseIdle fwUpdatePhase = iota
	fwPhaseDownload
	fwPhaseUpdate
	fwPhaseCompleted
	fwPhaseCancelled
	fwPhaseError
)

type firmwareUpdateState struct {
	deviceID   uint16
	deviceName string
	version    string
	phase      fwUpdatePhase
	percentage int
	statusMsg  string
	errorMsg   string
}

var fwUpdateState *firmwareUpdateState

type fwUpdateAvailability struct {
	deviceID   uint16
	deviceName string
	version    string
	lineIndex  int
}

var fwUpdatesAvailable []fwUpdateAvailability

//export firmwareProgressFunc
func firmwareProgressFunc(deviceID C.ushort, eventType C.Jabra_FirmwareEventType, status C.Jabra_FirmwareEventStatus, percentage C.ushort) {
	if fwUpdateState == nil {
		return
	}

	switch eventType {
	case C.Firmware_Download:
		fwUpdateState.phase = fwPhaseDownload
	case C.Firmware_Update:
		fwUpdateState.phase = fwPhaseUpdate
	}

	fwUpdateState.percentage = int(percentage)

	switch status {
	case C.Initiating:
		fwUpdateState.statusMsg = "Initiating..."
	case C.InProgress:
		if eventType == C.Firmware_Download {
			fwUpdateState.statusMsg = fmt.Sprintf("Downloading... %d%%", int(percentage))
		} else {
			fwUpdateState.statusMsg = fmt.Sprintf("Updating... %d%%", int(percentage))
		}
	case C.Completed:
		if eventType == C.Firmware_Download {
			fwUpdateState.statusMsg = "Download complete, starting update..."
			go startFirmwareUpdate(fwUpdateState.deviceID, fwUpdateState.version)
		} else {
			fwUpdateState.phase = fwPhaseCompleted
			fwUpdateState.statusMsg = "Firmware update completed!"
		}
	case C.Cancelled:
		fwUpdateState.phase = fwPhaseCancelled
		fwUpdateState.statusMsg = "Cancelled"
	default:
		fwUpdateState.phase = fwPhaseError
		fwUpdateState.errorMsg = firmwareEventStatusToString(int(status))
		fwUpdateState.statusMsg = fwUpdateState.errorMsg
	}

	select {
	case chFirmwareProgress <- firmwareProgressEvent{
		deviceID:   uint16(deviceID),
		eventType:  int(eventType),
		status:     int(status),
		percentage: int(percentage),
	}:
	default:
	}
}

func firmwareEventStatusToString(status int) string {
	switch status {
	case 4:
		return "File not available"
	case 5:
		return "File not accessible"
	case 6:
		return "File already present"
	case 7:
		return "Network error"
	case 8:
		return "SSL error"
	case 9:
		return "Download error"
	case 10:
		return "Update error"
	case 11:
		return "Invalid authentication"
	case 12:
		return "File under download"
	case 13:
		return "Not allowed"
	case 14:
		return "SDK too old for update"
	default:
		return fmt.Sprintf("Unknown error (%d)", status)
	}
}

func registerFirmwareProgressCallback() {
	C.Jabra_RegisterFirmwareProgressCallBack((*[0]byte)(C.firmwareProgressFunc))
}

func startFirmwareDownload(deviceID uint16, version string) {
	fwUpdateState = &firmwareUpdateState{
		deviceID:  deviceID,
		version:   version,
		phase:     fwPhaseDownload,
		statusMsg: "Starting download...",
	}
	if device, exists := deviceManager[selectedDongle]; exists && device.deviceID == deviceID {
		fwUpdateState.deviceName = device.deviceName
	} else if device, exists := deviceManager[selectedHeadset]; exists && device.deviceID == deviceID {
		fwUpdateState.deviceName = device.deviceName
	}

	cVersion := C.CString(version)
	defer C.free(unsafe.Pointer(cVersion))
	rc := returnCode(int(C.Jabra_DownloadFirmware(C.ushort(deviceID), cVersion, nil)))
	if rc != nil {
		// Return_Async (29) is expected for async operation
		rcStr := rc.Error()
		if rcStr != "Return_Async" {
			fwUpdateState.phase = fwPhaseError
			fwUpdateState.errorMsg = rcStr
			fwUpdateState.statusMsg = rcStr
		}
	}
}

func startFirmwareUpdate(deviceID uint16, version string) {
	cVersion := C.CString(version)
	filePath := C.Jabra_GetFirmwareFilePath(C.ushort(deviceID), cVersion)
	C.free(unsafe.Pointer(cVersion))
	if filePath == nil {
		fwUpdateState.phase = fwPhaseError
		fwUpdateState.errorMsg = "Could not get firmware file path"
		fwUpdateState.statusMsg = fwUpdateState.errorMsg
		return
	}
	defer C.Jabra_FreeString(filePath)

	rc := returnCode(int(C.Jabra_UpdateFirmware(C.ushort(deviceID), filePath)))
	if rc != nil {
		rcStr := rc.Error()
		if rcStr != "Return_Async" {
			fwUpdateState.phase = fwPhaseError
			fwUpdateState.errorMsg = rcStr
			fwUpdateState.statusMsg = rcStr
		}
	}
}

func cancelFirmwareDownload(deviceID uint16) error {
	return returnCode(int(C.Jabra_CancelFirmwareDownload(C.ushort(deviceID))))
}

/****************************************************************************/
/*                          DEVICE SETTINGS (Generic)                       */
/****************************************************************************/

func findDeviceSetting(deviceID uint16, name string) *deviceSetting {
	cSettings := C.Jabra_GetSettings(C.ushort(deviceID))
	if cSettings == nil {
		return nil
	}
	defer C.Jabra_FreeDeviceSettings(cSettings)

	settingsArr := (*[1 << 30]C.SettingInfo)(unsafe.Pointer(cSettings.settingInfo))[:cSettings.settingCount:cSettings.settingCount]

	for _, s := range settingsArr {
		settingName := C.GoString(s.name)
		if !strings.Contains(strings.ToLower(settingName), strings.ToLower(name)) {
			continue
		}

		ds := &deviceSetting{
			guid: C.GoString(s.guid),
			name: settingName,
		}

		if s.settingDataType == C.settingByte && s.currValue != nil {
			ds.current = int(*(*C.ushort)(s.currValue))
		}

		if s.listSize > 0 && s.listKeyValue != nil {
			kvArr := (*[1 << 30]C.ListKeyValue)(unsafe.Pointer(s.listKeyValue))[:s.listSize:s.listSize]
			ds.options = make([]string, 0, s.listSize)
			for _, kv := range kvArr {
				ds.options = append(ds.options, C.GoString(kv.value))
			}
		}

		return ds
	}

	return nil
}

func setDeviceSetting(deviceID uint16, guid string, key int) error {
	cGuid := C.CString(guid)
	defer C.free(unsafe.Pointer(cGuid))

	cSettings := C.Jabra_GetSetting(C.ushort(deviceID), cGuid)
	if cSettings == nil {
		return fmt.Errorf("setting not found")
	}

	if cSettings.settingCount > 0 {
		settingsArr := (*[1 << 30]C.SettingInfo)(unsafe.Pointer(cSettings.settingInfo))[:1:1]
		if settingsArr[0].settingDataType == C.settingByte && settingsArr[0].currValue != nil {
			*(*C.ushort)(settingsArr[0].currValue) = C.ushort(key)
		}
	}

	err := returnCode(int(C.Jabra_SetSettings(C.ushort(deviceID), cSettings)))
	C.Jabra_FreeDeviceSettings(cSettings)
	return err
}

/****************************************************************************/
/*                          PIPEWIRE AUDIO                                  */
/****************************************************************************/

func discoverPipeWireDevice() *audioState {
	out, err := exec.Command("wpctl", "status").Output()
	if err != nil {
		return nil
	}

	var dongleName string
	if dongle, exists := deviceManager[selectedDongle]; exists {
		dongleName = dongle.deviceName
	} else {
		return nil
	}

	lines := strings.Split(string(out), "\n")
	state := &audioState{deviceID: -1, sinkID: -1, sourceID: -1}

	inAudio := false
	subsection := "" // "devices", "sinks", "sources"
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Top-level section headers
		if trimmed == "Audio" {
			inAudio = true
			subsection = ""
			continue
		}
		if trimmed == "Video" || trimmed == "Settings" {
			inAudio = false
			continue
		}
		if !inAudio {
			continue
		}

		// Sub-section headers (use exact suffixes to avoid "Sink endpoints:" matching "Sinks:")
		stripped := strings.TrimLeft(trimmed, "│├└─ ")
		switch {
		case stripped == "Devices:":
			subsection = "devices"
			continue
		case stripped == "Sinks:":
			subsection = "sinks"
			continue
		case stripped == "Sources:":
			subsection = "sources"
			continue
		case strings.HasSuffix(stripped, "endpoints:") || stripped == "Streams:" || stripped == "Filters:":
			subsection = ""
			continue
		}

		if subsection == "" || !strings.Contains(trimmed, dongleName) {
			continue
		}

		id := parsePipeWireID(trimmed)
		if id < 0 {
			continue
		}

		switch subsection {
		case "devices":
			state.deviceID = id
		case "sinks":
			state.sinkID = id
		case "sources":
			state.sourceID = id
		}
	}

	if state.deviceID < 0 {
		return nil
	}

	return state
}

func parsePipeWireID(line string) int {
	// Lines look like: "  *  49. Jabra Link 390 [vol: 1.00]"
	// or "     49. Jabra Link 390 [vol: 1.00]"
	cleaned := strings.TrimLeft(line, " *│├└─")
	dotIdx := strings.Index(cleaned, ".")
	if dotIdx < 0 {
		return -1
	}
	numStr := strings.TrimSpace(cleaned[:dotIdx])
	id, err := strconv.Atoi(numStr)
	if err != nil {
		return -1
	}
	return id
}

func getPipeWireProfiles(deviceID int) []audioProfile {
	out, err := exec.Command("pw-cli", "enum-params", strconv.Itoa(deviceID), "EnumProfile").Output()
	if err != nil {
		return nil
	}

	var profiles []audioProfile
	lines := strings.Split(string(out), "\n")
	var current *audioProfile
	expectField := "" // "index", "name", "description"

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "Object:") {
			if current != nil {
				profiles = append(profiles, *current)
			}
			current = &audioProfile{}
			expectField = ""
			continue
		}

		if strings.Contains(trimmed, "Profile:index") {
			expectField = "index"
			continue
		}
		if strings.Contains(trimmed, "Profile:name") {
			expectField = "name"
			continue
		}
		if strings.Contains(trimmed, "Profile:description") {
			expectField = "description"
			continue
		}

		if current != nil && expectField != "" {
			switch expectField {
			case "index":
				if strings.HasPrefix(trimmed, "Int ") {
					val := strings.TrimPrefix(trimmed, "Int ")
					current.index, _ = strconv.Atoi(val)
				}
			case "name":
				if strings.HasPrefix(trimmed, "String ") {
					current.name = strings.Trim(strings.TrimPrefix(trimmed, "String "), "\"")
				}
			case "description":
				if strings.HasPrefix(trimmed, "String ") {
					current.description = strings.Trim(strings.TrimPrefix(trimmed, "String "), "\"")
				}
			}
			expectField = ""
		}
	}
	if current != nil {
		profiles = append(profiles, *current)
	}

	return profiles
}

func getActiveProfile(deviceID int) int {
	out, err := exec.Command("pw-cli", "enum-params", strconv.Itoa(deviceID), "Profile").Output()
	if err != nil {
		return -1
	}

	expectIndex := false
	for _, line := range strings.Split(string(out), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "Profile:index") {
			expectIndex = true
			continue
		}
		if expectIndex && strings.HasPrefix(trimmed, "Int ") {
			val := strings.TrimPrefix(trimmed, "Int ")
			idx, err := strconv.Atoi(val)
			if err == nil {
				return idx
			}
		}
	}
	return -1
}

func setAudioProfile(deviceID, index int) error {
	err := exec.Command("pw-cli", "set-param", strconv.Itoa(deviceID), "Profile",
		fmt.Sprintf("{ index: %d }", index)).Run()
	if err != nil {
		return err
	}

	// After profile change, PipeWire recreates sink/source nodes with new IDs.
	// Re-discover and set as default so audio doesn't fall back to built-in speakers.
	time.Sleep(500 * time.Millisecond)
	state := discoverPipeWireDevice()
	if state != nil {
		if state.sinkID >= 0 {
			exec.Command("wpctl", "set-default", strconv.Itoa(state.sinkID)).Run()
		}
		if state.sourceID >= 0 {
			exec.Command("wpctl", "set-default", strconv.Itoa(state.sourceID)).Run()
		}
	}
	return nil
}

func getVolume(nodeID int) float64 {
	out, err := exec.Command("wpctl", "get-volume", strconv.Itoa(nodeID)).Output()
	if err != nil {
		return -1
	}
	// Output: "Volume: 0.58" or "Volume: 0.58 [MUTED]"
	parts := strings.Fields(string(out))
	if len(parts) >= 2 {
		vol, err := strconv.ParseFloat(parts[1], 64)
		if err == nil {
			return vol
		}
	}
	return -1
}

func setVolume(nodeID int, vol float64) error {
	if vol < 0 {
		vol = 0
	}
	if vol > 1.5 {
		vol = 1.5
	}
	return exec.Command("wpctl", "set-volume", strconv.Itoa(nodeID),
		fmt.Sprintf("%.2f", vol)).Run()
}

func refreshAudioState() {
	if currentAudioState == nil {
		return
	}
	if currentAudioState.deviceID >= 0 {
		currentAudioState.profiles = getPipeWireProfiles(currentAudioState.deviceID)
		currentAudioState.activeProfile = getActiveProfile(currentAudioState.deviceID)
	}
	if currentAudioState.sinkID >= 0 {
		currentAudioState.outputVolume = getVolume(currentAudioState.sinkID)
	}
	if currentAudioState.sourceID >= 0 {
		currentAudioState.inputVolume = getVolume(currentAudioState.sourceID)
	}
}
