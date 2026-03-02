# jLink: Jabra Direct for Linux

[![GitHub Release](https://img.shields.io/github/v/release/Watchdog0x/jLink)](https://github.com/Watchdog0x/jLink/releases)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/Watchdog0x/jLink/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/Watchdog0x/jLink)](https://goreportcard.com/report/github.com/Watchdog0x/jLink)


jLink is your go-to tool for managing **Jabra headsets and dongles** on Linux. Think of it as **Jabra Direct for Linux**. <br>
Now you can finally manage your Jabra devices on Linux with ease.
jLink brings the power of Jabra device management to your Linux system.

<div align="center">
  <img src="./src/image.png" alt="How jLink look" style="max-width: 100%; height: auto;">
</div>

## Features

- **Device Discovery**: Search for new Bluetooth devices and manage connections.
- **Paired Devices**: View, connect, disconnect, and remove paired devices.
- **Battery Status**: Real-time battery level with visual indicator in the header.
- **Headset Settings**: ANC mode, equalizer, sidetone, and busy light control.
- **ANC Granular Control**: Dedicated sub-screen for mode selection, intensity level slider, and mode loop configuration (on supported devices).
- **Firmware Management**: View current firmware versions, check for updates, and perform full firmware download & update with progress tracking.
- **Device Info**: Detailed device information including firmware version, ESN, SKU, serial number, connected Bluetooth device name, security mode, and device constants.
- **BT Link Quality**: Real-time Bluetooth link quality indicator (`[BT:High]`/`[BT:Low]`/`[BT:Off]`) in the header (on supported devices).
- **Head Detection**: On-head/off-head status displayed in the header (on supported devices).
- **PipeWire Audio**: Audio profile switching, output/input volume control via PipeWire.
- **Dongle Settings**: Auto-pairing toggle and factory reset.

## Navigation

| Key             | Action                  |
|------------------|-------------------------|
| `w` or `↑`      | Move up                |
| `s` or `↓`      | Move down              |
| `Enter`         | Select / toggle option |
| `a` / `d`       | Adjust slider (volume, equalizer, ANC level) |
| `q`             | Go back                |

### Side Menu

| Key             | Action                  |
|------------------|-------------------------|
| `1`, `2`, `3`, `4` | Select an option      |
| `q`             | Go back                |

### Firmware Update Screen

| Key             | Action                  |
|------------------|-------------------------|
| `c`             | Cancel download        |
| `q`             | Back (when done/error) |

## Installation and update
<div align="center">
  <img src="./src/install.png" alt="How jLink look" style="max-width: 100%; height: auto;">
</div>

### Option 1: Using `curl`
Run the following command in your terminal:
```bash
curl -so- https://raw.githubusercontent.com/Watchdog0x/jLink/main/install.sh | sudo bash
```

### Option 2: Using `wget`
Run the following command in your terminal:

```bash
wget -qO- https://raw.githubusercontent.com/Watchdog0x/jLink/main/install.sh | sudo bash
```

## Tested Devices

- Jabra Link 380 with Jabra Evolve2 85
- Jabra Link 390 with Jabra Evolve2 65

> **Note**: Some features (ANC granular control, BT link quality, head detection) depend on device capabilities. Not all features are available on all headsets.

## TODO

1. Code Cleanup: Improve the current codebase, which is in need of refactoring.
2. Device Switching: Add support for switching between multiple connected devices.
3. Daemon Service: Create a background service using IPC shared memory for seamless operation.

## Contributing

Contributions are welcome! Here are some ways you can help:
- **Refactor and Clean Up**: Improve the existing codebase.
- **Implement New Features**: Tackle items from the TODO list.
- **Report Bugs**: Open an issue if you encounter any problems.
- **Suggest Enhancements**: Share your ideas for improving jLink.

## Keywords
- Jabra Direct Linux
- Jabra headset Linux support
- Jabra Linux command-line tool
- Manage Jabra devices on Linux
- Jabra Link 380 Linux
- Jabra Evolve2 85 Linux
- Jabra Evolve2 65 Linux
- Jabra firmware update Linux
- Jabra ANC Linux

