# Go Radiation Portal Monitor (RPM) Simulator

## 1. Overview

This project is a high-performance, single-executable Go application that simulates one or multiple Radiation Portal Monitors (RPMs). It is a full migration of the previous Python-based `PyRPM` project. By leveraging Pure Go libraries, it eliminates all external dependencies (such as Python runtimes, C libraries, and GStreamer), simplifying deployment on field hardware.

The simulator runs continuously and independently in one or more "lanes," procedurally generating radiation profiles for background counts and simulated vehicle occupancies. It streams simulated detector data via TCP and broadcasts an MJPEG video overlay (simulating a camera feed) over RTSP.

## 2. Features

* **High Performance**: Rewritten entirely in idiomatic Go with lightweight Goroutines for concurrency.
* **Pure Go RTSP Server**: Uses `gortsplib/v4` to stream MJPEG frames showing a microsecond clock and an "Occupied" overlay. No external GStreamer or OpenCV required.
* **Single Binary**: Easily compiled into a single static binary for Linux, Windows, or macOS, completely removing the need for local package managers or interpreters on target machines.
* **Configuration via JSON**: Controlled using a `settings.json` file.
* **Realistic Data Generation**: Uses statistical probability to generate varied occupancies (Gamma, Neutron, Combined, or Normal).

## 3. Quick Start

### 3.1. Using Pre-Compiled Binaries

Check the `dist` folder after running the build script, or grab the appropriate binary for your system:

| Platform | Architecture | Binary File |
| :--- | :--- | :--- |
| **Linux** | x64 (amd64) | `pyrpm-linux-x64` |
| **Windows** | x64 (amd64) | `pyrpm-windows-x64.exe` |
| **macOS** | Apple Silicon (arm64) | `pyrpm-macos-arm64` |

1. Create a `settings.json` file in the same directory as the executable (an example is included in the project).
2. Run the executable from your terminal:
   ```bash
   ./pyrpm-linux-x64
   ```

### 3.2. Building from Source

Ensure you have [Go](https://go.dev/doc/install) installed (version 1.20+ recommended).

```bash
make build
```
This command generates the stripped binaries inside the `/dist` directory.

## 4. How It Works

Upon starting, the application reads the `settings.json` file. For every lane configured and enabled, a new `Goroutine` sets up a dedicated TCP data server and an RTSP stream.

### Connecting to Data
Connect to a lane's raw TCP stream using a network utility:
```bash
# e.g., for Lane 1 on the default port
nc 127.0.0.1 10001
```

### Viewing the Camera Stream
Open VLC or a compatible RTSP client and connect to the lane's RTSP endpoint:
```
rtsp://127.0.0.1:8554/
```
*(Subsequent lanes increment the port number starting from 8554, based on LaneID)*

### Web UI
The simulator also serves a Web UI to view the live status of the lanes and manually trigger simulated occupancies.
Open your browser and navigate to:
```
http://127.0.0.1:8080/
```

## 5. Third-Party Packages Used

The following Pure Go libraries made this migration possible without CGO:

* **`github.com/bluenviron/gortsplib/v4`**: Used to implement the RTSP streaming server natively in Go.
* **`github.com/spf13/viper`**: A robust configuration management tool for parsing and falling back to default values for `settings.json`.
* **`golang.org/x/image/font` & `image/jpeg`**: Core Go libraries used to programmatically draw text and encode MJPEG frames directly in memory, eliminating the need for OpenCV/GStreamer.
