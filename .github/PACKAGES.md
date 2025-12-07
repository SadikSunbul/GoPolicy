# GitHub Packages Usage

This project automatically builds and uploads the GoPolicy binary to GitHub Packages. Every time a tag is pushed, it automatically builds and uploads to both GitHub Releases and GitHub Packages.

## 📦 Package Information

- **Registry**: `ghcr.io`
- **Package**: `ghcr.io/[USERNAME]/gopolicy`
- **Versions**: A separate version is created for each tag (e.g., `v1.0.1`, `latest`)

## 🚀 Usage

### Download from GitHub Releases

1. Go to the [Releases page](https://github.com/[USERNAME]/GoPolicy/releases)
2. Select the version you want
3. Download the `gopolicy.exe` file

### Download from GitHub Packages

You can use ORAS CLI to download the binary from GitHub Packages:

```powershell
# Download ORAS CLI (if not already installed)
# https://github.com/oras-project/oras/releases

# Create a GitHub Personal Access Token with packages:read permission

# Download the binary
oras pull ghcr.io/[USERNAME]/gopolicy:v1.0.1 --output gopolicy.exe
```

### Docker/Container Usage

You can use the OCI artifact from GitHub Packages as a container:

```bash
# Login
echo $GITHUB_TOKEN | docker login ghcr.io -u [USERNAME] --password-stdin

# Pull
docker pull ghcr.io/[USERNAME]/gopolicy:v1.0.1
```

## 🔐 Authentication

A GitHub Personal Access Token (PAT) is required to access GitHub Packages:

1. GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Create a token with `read:packages` permission
3. Use the token to login

## 📝 Versioning

- Automatic build and release is created when a tag is pushed
- Tag format: `v1.0.1`, `v1.0.2`, etc.
- The `latest` tag always points to the most recent version

## 🔄 Automatic Build

The workflow is triggered in the following cases:

1. **Tag Push**: When tags matching the `v*.*.*` format are pushed
2. **Manual Trigger**: Can be run manually from the GitHub Actions UI

## 📋 Release Contents

Each release includes:

- `gopolicy.exe` - Main executable file
- `gopolicy-windows-amd64.zip` - Zip archive containing all files
- `gopolicy.exe.sha256` - SHA256 checksum file
- `checksums.txt` - Text file containing all checksums
- `README.md` - Documentation
- `LICENSE` - License file

## 🛠️ Local Build

To build locally:

```powershell
# Build for Windows
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -o gopolicy.exe .
```

## 📚 More Information

- [GitHub Packages Documentation](https://docs.github.com/en/packages)
- [ORAS CLI Documentation](https://oras.land/docs/)
- [GoPolicy README](../README.md)
