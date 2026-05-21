# Installation

Whooper is a Go-based CLI tool. You can install it from source or by downloading pre-built binaries.

## Prerequisites

- **Go 1.24.x** or later is required to build from source.

## 1. Install from Source (Go)

You can install Whooper directly using the Go toolchain.

### Versioned Install
To install a specific version (recommended for stability):

```bash
go install git.infra.hisao.org/hisao/whooper@v0.1.0
```

*Note: Replace `v0.1.0` with the latest release tag.*

### Latest Development Version
To install the latest version from the main branch:

```bash
go install git.infra.hisao.org/hisao/whooper@latest
```

## 2. Install from Checkout (Make)

If you have cloned the repository, you can use the provided `Makefile`. This method embeds version information derived from git into the binary.

```bash
git clone https://git.infra.hisao.org/hisao/whooper.git
cd whooper
make install
```

The binary will be installed to your `$GOPATH/bin` directory.

## 3. Pre-built Binaries (GoReleaser)

Pre-built binaries are available for Linux, macOS, and Windows on this repository's releases page.

### Steps:
1.  Download the appropriate archive for your operating system and architecture:
    - **Linux**: `whooper_X.Y.Z_linux_amd64.tar.gz` (or `arm64`)
    - **macOS**: `whooper_X.Y.Z_darwin_amd64.tar.gz` (or `arm64`)
    - **Windows**: `whooper_X.Y.Z_windows_amd64.zip` (or `arm64`)
2.  (Optional but recommended) Verify the integrity of the download using `checksums.txt`:
    ```bash
    sha256sum --check --ignore-missing checksums.txt
    ```
3.  Extract the binary from the archive, substituting the name of the file you downloaded:
    - **Linux/macOS**:
      ```bash
      tar -xzf whooper_X.Y.Z_linux_amd64.tar.gz
      ```
    - **Windows**: Extract the `.zip` file using your preferred tool.
4.  Move the `whooper` binary to a directory in your `PATH` (e.g., `/usr/local/bin` or `~/bin`).

## 4. Verifying Installation

After installation, verify that Whooper is correctly installed and accessible:

```bash
# Check version
whooper version

# Run local readiness checks
whooper doctor --skip-api
```

The `doctor --skip-api` command validates local configuration, token persistence, and database connectivity without requiring an active internet connection or Whoop API credentials.
