//go:build windows

package policy

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unicode/utf16"
)

// POL file format: Registry.pol (PReg format)
// Header: PReg\x01\x00\x00\x00
// Her entry: [key;value;type;size;data]

const (
	PolHeader    = "PReg\x01\x00\x00\x00"
	PolEntryOpen = '['
	PolEntryEnd  = ']'
	PolSemicolon = ';'
)

// PolEntry represents a single entry in a Registry.pol file
type PolEntry struct {
	KeyName   string
	ValueName string
	Type      uint32
	Data      []byte
}

// PolFile represents a Registry.pol file
type PolFile struct {
	Entries []PolEntry
	path    string
}

// LoadPolFile loads a .pol file
func LoadPolFile(path string) (*PolFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// If the file does not exist, return an empty pol file
			return &PolFile{path: path, Entries: []PolEntry{}}, nil
		}
		return nil, fmt.Errorf("pol file could not be loaded: %w", err)
	}

	if len(data) < 8 {
		return nil, fmt.Errorf("invalid pol file: too short")
	}

	header := string(data[:8])
	if header != PolHeader {
		return nil, fmt.Errorf("invalid pol header: %s", header)
	}

	pol := &PolFile{path: path}
	buf := bytes.NewReader(data[8:])

	for {
		var openBracket byte
		if err := binary.Read(buf, binary.LittleEndian, &openBracket); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if openBracket != PolEntryOpen {
			return nil, fmt.Errorf("expected '[' character instead of: %c", openBracket)
		}

		// Read KeyName (null-terminated UTF-16LE)
		keyName, err := readUTF16String(buf)
		if err != nil {
			return nil, fmt.Errorf("key could not be read: %w", err)
		}

		// Semicolon
		var semi1 byte
		binary.Read(buf, binary.LittleEndian, &semi1)

		// Read ValueName
		valueName, err := readUTF16String(buf)
		if err != nil {
			return nil, fmt.Errorf("value could not be read: %w", err)
		}

		// Semicolon
		var semi2 byte
		binary.Read(buf, binary.LittleEndian, &semi2)

		// Type (DWORD)
		var regType uint32
		if err := binary.Read(buf, binary.LittleEndian, &regType); err != nil {
			return nil, err
		}

		// Semicolon
		var semi3 byte
		binary.Read(buf, binary.LittleEndian, &semi3)

		// Size (DWORD)
		var dataSize uint32
		if err := binary.Read(buf, binary.LittleEndian, &dataSize); err != nil {
			return nil, err
		}

		// Semicolon
		var semi4 byte
		binary.Read(buf, binary.LittleEndian, &semi4)

		// Data
		entryData := make([]byte, dataSize)
		if dataSize > 0 {
			if _, err := io.ReadFull(buf, entryData); err != nil {
				return nil, err
			}
		}

		// ']'
		var closeBracket byte
		binary.Read(buf, binary.LittleEndian, &closeBracket)

		pol.Entries = append(pol.Entries, PolEntry{
			KeyName:   keyName,
			ValueName: valueName,
			Type:      regType,
			Data:      entryData,
		})
	}

	return pol, nil
}

// Save writes the pol file to disk
func (p *PolFile) Save() error {
	// Create parent directory
	dir := filepath.Dir(p.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("pol directory could not be created: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString(PolHeader)

	for _, entry := range p.Entries {
		buf.WriteByte(PolEntryOpen)

		// KeyName (UTF-16LE + null)
		writeUTF16String(&buf, entry.KeyName)
		buf.WriteByte(PolSemicolon)
		buf.WriteByte(0)

		// ValueName
		writeUTF16String(&buf, entry.ValueName)
		buf.WriteByte(PolSemicolon)
		buf.WriteByte(0)

		// Type
		binary.Write(&buf, binary.LittleEndian, entry.Type)
		buf.WriteByte(PolSemicolon)
		buf.WriteByte(0)

		// Size
		dataSize := uint32(len(entry.Data))
		binary.Write(&buf, binary.LittleEndian, dataSize)
		buf.WriteByte(PolSemicolon)
		buf.WriteByte(0)

		// Data
		if dataSize > 0 {
			buf.Write(entry.Data)
		}

		buf.WriteByte(PolEntryEnd)
		buf.WriteByte(0)
	}

	// Use a temporary file for atomic write
	tmpPath := p.path + ".tmp"
	if err := os.WriteFile(tmpPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("pol file could not be written: %w", err)
	}

	if err := os.Rename(tmpPath, p.path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("pol file could not be saved: %w", err)
	}

	return nil
}

// SetValue adds/updates a registry value to the pol file
func (p *PolFile) SetValue(keyPath, valueName string, data interface{}, valueType RegistryValueKind) error {
	// First find and delete the same key+value
	p.DeleteValue(keyPath, valueName)

	var regType uint32
	var rawData []byte

	switch valueType {
	case RegString:
		regType = 1 // REG_SZ (string)
		str, ok := data.(string)
		if !ok {
			str = fmt.Sprintf("%v", data)
		}
		rawData = encodeUTF16String(str)

	case RegExpandString:
		regType = 2 // REG_EXPAND_SZ (expandable string)
		str, ok := data.(string)
		if !ok {
			str = fmt.Sprintf("%v", data)
		}
		rawData = encodeUTF16String(str)

	case RegDWord:
		regType = 4 // REG_DWORD (32-bit integer)
		var dword uint32
		switch v := data.(type) {
		case uint32:
			dword = v
		case uint64:
			dword = uint32(v)
		case int:
			dword = uint32(v)
		case int64:
			dword = uint32(v)
		case bool:
			if v {
				dword = 1
			} else {
				dword = 0
			}
		default:
			return fmt.Errorf("invalid data type for DWORD: %T", data)
		}
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.LittleEndian, dword)
		rawData = buf.Bytes()

	case RegMultiString:
		regType = 7 // REG_MULTI_SZ (multiple strings)
		strs, ok := data.([]string)
		if !ok {
			return fmt.Errorf("invalid data type for MultiString: %T", data)
		}
		var buf bytes.Buffer
		for _, s := range strs {
			writeUTF16String(&buf, s)
		}
		// Null terminator
		buf.WriteByte(0)
		buf.WriteByte(0)
		rawData = buf.Bytes()

	default:
		return fmt.Errorf("unsupported registry type: %d", valueType)
	}

	p.Entries = append(p.Entries, PolEntry{
		KeyName:   keyPath,
		ValueName: valueName,
		Type:      regType,
		Data:      rawData,
	})

	return nil
}

// DeleteValue deletes a value
func (p *PolFile) DeleteValue(keyPath, valueName string) {
	var filtered []PolEntry
	for _, entry := range p.Entries {
		if entry.KeyName == keyPath && entry.ValueName == valueName {
			continue
		}
		filtered = append(filtered, entry)
	}
	p.Entries = filtered
}

// GetValue reads a value
func (p *PolFile) GetValue(keyPath, valueName string) (interface{}, uint32, bool) {
	for _, entry := range p.Entries {
		if entry.KeyName == keyPath && entry.ValueName == valueName {
			return decodePolData(entry.Data, entry.Type), entry.Type, true
		}
	}
	return nil, 0, false
}

// ContainsValue checks if a value exists
func (p *PolFile) ContainsValue(keyPath, valueName string) bool {
	for _, entry := range p.Entries {
		if entry.KeyName == keyPath && entry.ValueName == valueName {
			return true
		}
	}
	return false
}

// GetValueNames returns all value names under a key
func (p *PolFile) GetValueNames(keyPath string) []string {
	var names []string
	seen := make(map[string]bool)
	for _, entry := range p.Entries {
		if entry.KeyName == keyPath && !seen[entry.ValueName] {
			names = append(names, entry.ValueName)
			seen[entry.ValueName] = true
		}
	}
	return names
}

// ClearKey clears all values under a key
func (p *PolFile) ClearKey(keyPath string) {
	var filtered []PolEntry
	for _, entry := range p.Entries {
		if entry.KeyName != keyPath {
			filtered = append(filtered, entry)
		}
	}
	p.Entries = filtered
}

// Helper functions

func readUTF16String(r io.Reader) (string, error) {
	var chars []uint16
	for {
		var c uint16
		if err := binary.Read(r, binary.LittleEndian, &c); err != nil {
			return "", err
		}
		if c == 0 {
			break
		}
		chars = append(chars, c)
	}
	return string(utf16.Decode(chars)), nil
}

// writeUTF16String writes a string to a writer in UTF-16LE
func writeUTF16String(w io.Writer, s string) {
	encoded := utf16.Encode([]rune(s))
	for _, c := range encoded {
		binary.Write(w, binary.LittleEndian, c)
	}
	// Null terminator
	binary.Write(w, binary.LittleEndian, uint16(0))
}

// encodeUTF16String encodes a string to UTF-16LE
func encodeUTF16String(s string) []byte {
	encoded := utf16.Encode([]rune(s))
	buf := new(bytes.Buffer)
	for _, c := range encoded {
		binary.Write(buf, binary.LittleEndian, c)
	}
	// Null terminator
	binary.Write(buf, binary.LittleEndian, uint16(0))
	return buf.Bytes()
}

// decodePolData decodes pol data to a Go value
func decodePolData(data []byte, regType uint32) interface{} {
	switch regType {
	case 1, 2: // REG_SZ, REG_EXPAND_SZ (string, expandable string)
		if len(data) < 2 {
			return ""
		}
		// UTF-16LE decode (string, expandable string)
		u16 := make([]uint16, len(data)/2)
		for i := 0; i < len(u16); i++ {
			u16[i] = binary.LittleEndian.Uint16(data[i*2 : i*2+2])
		}
		// Remove null terminator
		for i, c := range u16 {
			if c == 0 {
				u16 = u16[:i]
				break
			}
		}
		return string(utf16.Decode(u16))

	case 4: // REG_DWORD (32-bit integer)
		if len(data) >= 4 {
			return binary.LittleEndian.Uint32(data[:4])
		}
		return uint32(0)

	case 7: // REG_MULTI_SZ (multiple strings)
		var strs []string
		buf := bytes.NewReader(data)
		for {
			str, err := readUTF16String(buf)
			if err != nil || str == "" {
				break
			}
			strs = append(strs, str)
		}
		return strs

	default:
		return data
	}
}

// GetPolPath returns the path to the pol file for a given section
func GetPolPath(section AdmxPolicySection) (string, error) {
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		systemRoot = "C:\\Windows"
	}

	basePath := filepath.Join(systemRoot, "System32", "GroupPolicy")

	switch section {
	case User:
		return filepath.Join(basePath, "User", "Registry.pol"), nil
	case Machine:
		return filepath.Join(basePath, "Machine", "Registry.pol"), nil
	default:
		return "", fmt.Errorf("invalid section: %d", section)
	}
}
