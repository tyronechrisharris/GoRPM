# Adding New Architectures

Because the PyRPM migration uses Pure Go and natively manages RTSP streams via `gortsplib/v4` and `image/jpeg`—completely eliminating the need for `CGO` compilation—building for a new target OS or Architecture is simple and straightforward.

## Cross-Platform Build System

The default project architecture provides a script `/build.sh` (or `make build` via Makefile) that generates three pre-configured binaries in the `/dist` directory.

### Updating the `build.sh` script
If you want to compile PyRPM for a Raspberry Pi (or similar Linux-based ARM hardware), you simply need to define a new build command by appending a line to `build.sh` with the appropriate `GOOS` and `GOARCH` flags.

For example, to add `linux-arm`:
```bash
echo "Building for Linux (arm32)..."
GOOS=linux GOARCH=arm go build -ldflags="-s -w" -o "dist/gorpm-linux-arm" .
```

To add `linux-arm64` (e.g., Raspberry Pi 4 running 64-bit OS):
```bash
echo "Building for Linux (arm64)..."
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o "dist/gorpm-linux-arm64" .
```

Make sure you do the same inside the `Makefile` under the `build` target so both build systems remain consistent.

### Compiling Without Dependencies
Since `CGO_ENABLED=0` is implicitly supported due to the pure Go libraries used in this project, there are no C toolchains required on the build machine for cross-compilation.

You only need the standard Go compiler (`go build`).

### Minifying the Binaries
Always use the `-ldflags="-s -w"` flag to omit symbol tables and debugging information when adding new targets to your build script. This significantly minimizes binary size, which is critical for edge deployments on field hardware with limited storage.
