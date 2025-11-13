# Policy Plus - Go Edition

<div align="center">

🛡️ **Windows Group Policy Editor - Web-Based Interface for All Windows Versions**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows-0078D6?style=for-the-badge&logo=windows)](https://www.microsoft.com/windows)

</div>

---

## 📖 About

**Policy Plus - Go Edition** is a version of the original [PolicyPlus](https://github.com/Fleex255/PolicyPlus) project where the Visual Basic .NET code has been **directly translated to Go** and a **modern web interface** has been added.

### ✨ Features

- ✅ **Works on all Windows versions** (Home, Pro, Enterprise)
- 🌐 **Modern web-based interface** - Access from browser
- 📝 **ADMX/ADML file support** - Reads all Windows policy definitions
- 💾 **POL file read/write** - Group Policy Object management
- 🔧 **Registry editing** - Direct registry manipulation
- 🎨 **Visual and user-friendly** - Modern, responsive design
- 🚀 **Fast and lightweight** - Go language performance

## 🎯 Main Goals

1. **Universal Access**: Works on all versions including Windows Home
2. **License Compliance**: Works without shipping Windows components
3. **Full-Featured**: Local GPO, per-user GPO, POL files, Registry editing
4. **Easy to Use**: Web-based modern interface
5. **Direct Port**: Provides all functionality of original VB.NET code in Go

## 📦 Installation

### Requirements

- Go 1.21 or higher
- Windows Vista or higher (Windows Server 2008+ supported)
- Web browser (Chrome, Firefox, Edge, etc.)

### Installation with Binary

```bash
# Download binary from releases page and run
policy-plus.exe
```

### Build from Source

```bash
# Clone the project
git clone https://github.com/yourusername/go-PolicyPlus.git
cd go-PolicyPlus

# Install dependencies
go mod download

# Build and run
go build -o policy-plus.exe
policy-plus.exe
```

## 🚀 Usage

### 1. Start the Application

```bash
policy-plus.exe
```

When the application starts, you will see this output:

```
Policy Plus - Go Edition
Local Group Policy Editor for all Windows editions
========================================
Loading ADMX files: C:\Windows\PolicyDefinitions
Starting web interface: http://localhost:8080
Open in your browser and start using!
```

### 2. Open Web Interface

Open the following address in your browser:
```
http://localhost:8080
```

### 3. Manage Policies

1. **Select category from left panel**: Browse categories
2. **Select from policy list**: Find the policy you want
3. **Edit**: Double-click on the policy
4. **Set state**: Enabled / Disabled / Not Configured
5. **Configure settings**: Enter element values
6. **Save**: Apply changes

## 📁 Project Structure

```
go-PolicyPlus/
├── main.go                          # Main application entry point
├── go.mod                           # Go module definition
├── internal/
│   ├── policy/                      # Policy processing logic
│   │   ├── structures.go           # ADMX data structures
│   │   ├── compiled_structures.go  # Compiled policy structures
│   │   ├── presentation.go         # UI presentation structures
│   │   ├── admx_file.go           # ADMX XML reading
│   │   ├── adml_file.go           # ADML localization reading
│   │   ├── admx_bundle.go         # ADMX collection management
│   │   └── policy_processing.go   # Policy state management
│   ├── polfile/                    # POL file processing
│   │   └── pol_file.go            # Binary POL read/write
│   ├── registry/                   # Windows Registry interface
│   │   └── registry.go            # Registry manipulation
│   └── handlers/                   # HTTP handlers
│       └── handlers.go            # Web API endpoints
├── web/                            # Web interface
│   ├── static/
│   │   ├── style.css             # CSS styles
│   │   └── app.js                # JavaScript logic
│   └── templates/
│       └── index.html            # Main HTML template
└── README.md                       # This file
```

## 🔧 API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Main page |
| `/api/categories` | GET | List all categories |
| `/api/policies` | GET | List policies for a category |
| `/api/policy/{id}` | GET | Get policy details |
| `/api/policy/set` | POST | Set policy state |
| `/api/sources` | GET | List policy sources |
| `/api/save` | POST | Save changes |

## 🎨 Customization

### Change Port

You can change the port number in the `main.go` file:

```go
port := ":8080"  // Change to your desired port
```

### ADMX Folder

By default, `C:\Windows\PolicyDefinitions` is used. To use a different folder:

```go
admxPath := "C:\\YourCustomPath\\PolicyDefinitions"
```

## 🐛 Troubleshooting

### ADMX Files Cannot Be Loaded

Default ADMX files may be missing on Windows Home editions:

1. [Download ADMX files from Microsoft](https://www.microsoft.com/en-us/download/details.aspx?id=104593)
2. Extract to `C:\Windows\PolicyDefinitions` folder

### Port Already in Use Error

If another application is using port 8080, change the port or close the conflicting application.

### Access Denied

For registry write operations, **run as Administrator**.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **Original PolicyPlus**: [Fleex255/PolicyPlus](https://github.com/Fleex255/PolicyPlus)
- Translation of Visual Basic .NET code to Go and web interface addition

## 📞 Contact

- 🐛 For bug reports: [Issues](https://github.com/yourusername/go-PolicyPlus/issues)
- 💡 For feature suggestions: [Discussions](https://github.com/yourusername/go-PolicyPlus/discussions)

---

<div align="center">

**⭐ If you liked the project, don't forget to give it a star! ⭐**

Made with ❤️ using Go

</div>

