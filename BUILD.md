# Build ve Derleme Kılavuzu

## 🔨 Geliştirme Ortamı Kurulumu

### 1. Go Kurulumu

**Windows için:**
```powershell
# Chocolatey ile
choco install golang

# Veya
# https://go.dev/dl/ adresinden installer indir ve kur
```

Kurulumu doğrula:
```bash
go version
# go version go1.21.0 windows/amd64 gibi bir çıktı görmelisin
```

### 2. Proje Klonlama

```bash
git clone https://github.com/yourusername/go-PolicyPlus.git
cd go-PolicyPlus
```

### 3. Bağımlılıkları Yükle

```bash
go mod download
go mod tidy
```

## 🏗️ Derleme

### Development Build (Hızlı)

```bash
go build -o policy-plus.exe
```

### Production Build (Optimize)

```bash
go build -ldflags="-s -w" -o policy-plus.exe
```

Flags açıklaması:
- `-s`: Symbol tablosunu kaldır
- `-w`: DWARF debug bilgisini kaldır
- Sonuç: ~30-40% daha küçük binary

### Windows Subsystem GUI Build

Console penceresi göstermeden çalıştırmak için:

```bash
go build -ldflags="-s -w -H windowsgui" -o policy-plus.exe
```

## 🧪 Test

### Tüm Testleri Çalıştır

```bash
go test ./...
```

### Belirli Paket Testi

```bash
go test ./internal/policy
go test ./internal/polfile
```

### Coverage Raporu

```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 📦 Binary Boyutunu Küçültme

### 1. UPX ile Sıkıştırma

```bash
# UPX indir: https://upx.github.io/
upx --best --lzma policy-plus.exe
```

### 2. Embed Dosyaları Minimize Et

`main.go` içindeki embed direktiflerini kontrol et:
```go
//go:embed web/static/*
//go:embed web/templates/*
```

## 🔧 Cross-Compilation

### Windows için (başka platformdan)

```bash
# Linux/Mac'ten Windows binary derle
GOOS=windows GOARCH=amd64 go build -o policy-plus.exe

# 32-bit Windows
GOOS=windows GOARCH=386 go build -o policy-plus-x86.exe
```

## 🚀 Release Build

### 1. Version Tag

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

### 2. Multi-Platform Build

```bash
# Windows 64-bit
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/policy-plus-windows-amd64.exe

# Windows 32-bit
GOOS=windows GOARCH=386 go build -ldflags="-s -w" -o dist/policy-plus-windows-386.exe

# Windows ARM64
GOOS=windows GOARCH=arm64 go build -ldflags="-s -w" -o dist/policy-plus-windows-arm64.exe
```

### 3. Checksums Oluştur

```bash
cd dist
sha256sum * > checksums.txt
```

## 📋 Build Script (PowerShell)

`build.ps1` dosyası oluştur:

```powershell
# Build Script
$version = "1.0.0"
$ldflags = "-s -w -X main.Version=$version"

Write-Host "Building Policy Plus v$version..." -ForegroundColor Green

# Clean
Remove-Item -Recurse -Force dist -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path dist | Out-Null

# Windows 64-bit
Write-Host "Building Windows x64..." -ForegroundColor Cyan
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -ldflags="$ldflags" -o dist/policy-plus-windows-amd64.exe

# Windows 32-bit
Write-Host "Building Windows x86..." -ForegroundColor Cyan
$env:GOARCH = "386"
go build -ldflags="$ldflags" -o dist/policy-plus-windows-386.exe

# Windows ARM64
Write-Host "Building Windows ARM64..." -ForegroundColor Cyan
$env:GOARCH = "arm64"
go build -ldflags="$ldflags" -o dist/policy-plus-windows-arm64.exe

# Checksums
Write-Host "Generating checksums..." -ForegroundColor Cyan
Get-ChildItem dist/*.exe | ForEach-Object {
    $hash = (Get-FileHash $_.FullName -Algorithm SHA256).Hash
    "$hash  $($_.Name)" | Out-File -Append dist/checksums.txt
}

Write-Host "Build complete!" -ForegroundColor Green
Get-ChildItem dist
```

Çalıştır:
```powershell
.\build.ps1
```

## 🐧 Build Script (Bash - Linux/Mac)

`build.sh` dosyası oluştur:

```bash
#!/bin/bash

VERSION="1.0.0"
LDFLAGS="-s -w -X main.Version=$VERSION"

echo "Building Policy Plus v$VERSION..."

# Clean
rm -rf dist
mkdir -p dist

# Windows 64-bit
echo "Building Windows x64..."
GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" -o dist/policy-plus-windows-amd64.exe

# Windows 32-bit
echo "Building Windows x86..."
GOOS=windows GOARCH=386 go build -ldflags="$LDFLAGS" -o dist/policy-plus-windows-386.exe

# Windows ARM64
echo "Building Windows ARM64..."
GOOS=windows GOARCH=arm64 go build -ldflags="$LDFLAGS" -o dist/policy-plus-windows-arm64.exe

# Checksums
echo "Generating checksums..."
cd dist
sha256sum *.exe > checksums.txt
cd ..

echo "Build complete!"
ls -lh dist/
```

Çalıştır:
```bash
chmod +x build.sh
./build.sh
```

## 🔍 Troubleshooting

### CGO Hatası

Eğer Windows Registry işlemleri için CGO gerekirse:

```bash
# CGO'yu etkinleştir
$env:CGO_ENABLED = "1"

# MinGW-w64 gerekli olabilir
choco install mingw
```

### Embed Hatası

`web` klasörünün doğru konumda olduğundan emin ol:

```bash
# Proje kök dizininde olmalı
go-PolicyPlus/
├── web/
│   ├── static/
│   └── templates/
└── main.go
```

### Module Hatası

```bash
# Module cache'i temizle
go clean -modcache

# Tekrar dene
go mod download
go mod tidy
```

## 📊 Build İstatistikleri

```bash
# Binary boyutu
ls -lh policy-plus.exe

# Module bağımlılıkları
go list -m all

# Build zamanı ölç
time go build -o policy-plus.exe
```

## 🎯 Optimizasyon İpuçları

1. **Gereksiz Bağımlılıkları Kaldır**: `go mod tidy`
2. **Build Cache Kullan**: Go otomatik yapar
3. **Paralel Derleme**: Go varsayılan olarak paralel derler
4. **Static Assets**: Embedding yerine ayrı dosyalar (büyük projeler için)
5. **Profile-Guided Optimization**: Go 1.20+ için PGO kullan

## 📝 Version Management

`main.go` içinde version yönetimi:

```go
package main

var (
    Version   = "dev"
    BuildTime = "unknown"
    GitCommit = "unknown"
)

func main() {
    fmt.Printf("Policy Plus v%s\n", Version)
    fmt.Printf("Built: %s\n", BuildTime)
    fmt.Printf("Commit: %s\n", GitCommit)
    // ...
}
```

Build sırasında inject et:

```bash
go build -ldflags="
    -X main.Version=1.0.0 
    -X 'main.BuildTime=$(date)' 
    -X 'main.GitCommit=$(git rev-parse HEAD)'
" -o policy-plus.exe
```

---

Sorularınız için [Issues](https://github.com/yourusername/go-PolicyPlus/issues) açabilirsiniz!

