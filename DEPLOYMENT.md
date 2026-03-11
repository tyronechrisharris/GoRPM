# Deployment Guide for PyRPM (Go)

The Go migration of the PyRPM project has significantly simplified the deployment process for remote field hardware (such as Linux-based Portal Monitors or Windows CAS Stations). Because it operates as a statically linked, standalone executable, there are no dependencies to install on the target machine (i.e., no Python environment, GStreamer, or `gobject` required).

## Deploying to Linux-Based Portal Monitors

1. **Obtain the Binary**:
   Copy the `dist/pyrpm-linux-x64` executable to the target remote system. You can use `scp`, a USB drive, or an automated configuration management tool (like Ansible).

   ```bash
   scp dist/pyrpm-linux-x64 user@remote-ip:/opt/pyrpm/
   ```

2. **Configuration**:
   Ensure `settings.json` is located in the same directory as the executable on the target machine. Modify the IP bindings in the `settings.json` (e.g., `0.0.0.0`) so the network interfaces can be reached by other systems.

3. **Running the Application**:
   Make the binary executable if it isn't already:
   ```bash
   chmod +x /opt/pyrpm/pyrpm-linux-x64
   ```

4. **Service Persistence (Systemd)**:
   For long-running deployments, set up a Systemd service to ensure the simulator starts on boot and restarts if it crashes.
   Create `/etc/systemd/system/pyrpm.service`:
   ```ini
   [Unit]
   Description=PyRPM Go Simulator
   After=network.target

   [Service]
   Type=simple
   WorkingDirectory=/opt/pyrpm
   ExecStart=/opt/pyrpm/pyrpm-linux-x64
   Restart=always
   RestartSec=5

   [Install]
   WantedBy=multi-user.target
   ```
   Enable and start the service:
   ```bash
   systemctl daemon-reload
   systemctl enable pyrpm
   systemctl start pyrpm
   ```

## Deploying to Windows CAS Stations

1. **Obtain the Binary**:
   Copy the `dist/pyrpm-windows-x64.exe` executable to the desired directory on the CAS Station (e.g., `C:\PyRPM\`).

2. **Configuration**:
   Copy the appropriate `settings.json` to the same folder. Adjust firewall rules on Windows to allow inbound connections to the TCP data ports and RTSP streaming ports specified in your configuration.

3. **Running the Application**:
   You can run the executable directly from the Command Prompt or PowerShell:
   ```cmd
   cd C:\PyRPM\
   pyrpm-windows-x64.exe
   ```

4. **Service Persistence**:
   To run this tool in the background on a Windows machine, consider using tools such as [NSSM (Non-Sucking Service Manager)](https://nssm.cc/) to wrap the executable into a proper Windows Service.
