package main

/*
#cgo CFLAGS: -Iheaders
#cgo LDFLAGS: -Llib -ljabra

#include "Common.h"
#include "GoWrapper.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"log"
	"unsafe"

	tea "github.com/charmbracelet/bubbletea"
)

// sudo apt install libasound2 libcurl4
func main() {
	appId := C.CString("JabraLink")
	C.Jabra_SetAppID(appId)
	defer C.free(unsafe.Pointer(appId))

	// Callback parameters: FirstScanForDevicesDoneFunc, DeviceAttachedFunc, DeviceRemovedFunc,
	// ButtonInDataRawHidFunc, ButtonInDataTranslatedFunc, nonJabraDeviceDetection, configParams
	if init := C.Jabra_InitializeV2(
		nil,                              // FirstScanForDevicesDoneFunc (not used here)
		(*[0]byte)(C.deviceAttachedFunc), // Callback for when a device is attached
		(*[0]byte)(C.deviceRemovedFunc),  // Callback for when a device is removed
		nil,                              // Callback for raw HID button input (not used here)
		nil,                              // Callback for translated button input (not used here)
		false,                            // nonJabraDeviceDetection (not used here)
		nil,                              // Additional configuration parameters (not used here)
	); !init {
		log.Fatalln("Failed to initialize Jabra SDK")
	}
	defer uninitialize()
	registerFirmwareProgressCallback()

	// The current callback behavior is inconsistent. While the charging status updates as expected,
	// the `levelInPercent` callback is sometimes delayed. This causes issues with timely updates.
	// We need to ensure that the callback is triggered in a more predictable and consistent manner.
	// C.Jabra_RegisterBatteryStatusUpdateCallbackV2((*[0]byte)(unsafe.Pointer(C.batteryStatusUpdate)))

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalln(err)
	}

	fmt.Println("\nThank you for using jLink!")
}
