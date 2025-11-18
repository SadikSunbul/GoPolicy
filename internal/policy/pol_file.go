package policy

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"unicode/utf16"

	"golang.org/x/sys/windows/registry"
)

const (
	// POL file signature
	polSignature = 0x67655250 // "PReg"
	polVersion   = 1
)

// ValueType represents Windows registry value type
type ValueType uint32

// Windows registry value types (taken from registry package)
const (
	NONE      ValueType = registry.NONE
	SZ        ValueType = registry.SZ
	EXPAND_SZ ValueType = registry.EXPAND_SZ
	BINARY    ValueType = registry.BINARY
	DWORD     ValueType = registry.DWORD
	MULTI_SZ  ValueType = registry.MULTI_SZ
	QWORD     ValueType = registry.QWORD
)

// PolFile POL file
type PolFile struct {
	entries          map[string]*polEntryData
	casePreservation map[string]string
}

type polEntryData struct {
	Kind ValueType
	Data []byte
}

// NewPolFile creates a new POL file
func NewPolFile() *PolFile {
	return &PolFile{
		entries:          make(map[string]*polEntryData),
		casePreservation: make(map[string]string),
	}
}

// Load loads POL file
func Load(path string) (*PolFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return LoadFromReader(file)
}

// LoadFromReader reads POL file from reader
func LoadFromReader(reader io.Reader) (*PolFile, error) {
	pol := NewPolFile()

	// Check signature
	var sig uint32
	if err := binary.Read(reader, binary.LittleEndian, &sig); err != nil {
		return nil, fmt.Errorf("failed to read signature: %w", err)
	}
	if sig != polSignature {
		return nil, fmt.Errorf("invalid POL signature: %08x", sig)
	}

	// Check version
	var ver uint32
	if err := binary.Read(reader, binary.LittleEndian, &ver); err != nil {
		return nil, fmt.Errorf("failed to read version: %w", err)
	}
	if ver != polVersion {
		return nil, fmt.Errorf("unsupported POL version: %d", ver)
	}

	// Read entries
	for {
		entry, err := readEntry(reader)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		dictKey := pol.getDictKey(entry.key, entry.value)
		pol.entries[dictKey] = entry.data
	}

	return pol, nil
}

type polEntry struct {
	key   string
	value string
	data  *polEntryData
}

func readEntry(reader io.Reader) (*polEntry, error) {
	// Read "[" character
	var bracket uint16
	if err := binary.Read(reader, binary.LittleEndian, &bracket); err != nil {
		return nil, err
	}
	if bracket != '[' {
		return nil, fmt.Errorf("expected '[', got: %c", rune(bracket))
	}

	// Key name
	keyName, err := readNullTerminatedUTF16(reader)
	if err != nil {
		return nil, err
	}

	// ";"
	if err := expectChar(reader, ';'); err != nil {
		return nil, err
	}

	// Value name
	valueName, err := readNullTerminatedUTF16(reader)
	if err != nil {
		return nil, err
	}

	// ";" (sometimes there may be an extra null)
	var semi uint16
	if err := binary.Read(reader, binary.LittleEndian, &semi); err != nil {
		return nil, err
	}
	if semi == 0 {
		// Extra null, read again
		if err := binary.Read(reader, binary.LittleEndian, &semi); err != nil {
			return nil, err
		}
	}
	if semi != ';' {
		return nil, fmt.Errorf("expected ';', got: %c", rune(semi))
	}

	// Type
	var regType uint32
	if err := binary.Read(reader, binary.LittleEndian, &regType); err != nil {
		return nil, err
	}

	// ";"
	if err := expectChar(reader, ';'); err != nil {
		return nil, err
	}

	// Data length
	var length uint32
	if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
		return nil, err
	}

	// ";"
	if err := expectChar(reader, ';'); err != nil {
		return nil, err
	}

	// Data
	data := make([]byte, length)
	if _, err := io.ReadFull(reader, data); err != nil {
		return nil, err
	}

	// "]"
	if err := expectChar(reader, ']'); err != nil {
		return nil, err
	}

	return &polEntry{
		key:   keyName,
		value: valueName,
		data: &polEntryData{
			Kind: ValueType(regType),
			Data: data,
		},
	}, nil
}

func readNullTerminatedUTF16(reader io.Reader) (string, error) {
	var chars []uint16
	for {
		var char uint16
		if err := binary.Read(reader, binary.LittleEndian, &char); err != nil {
			return "", err
		}
		if char == 0 {
			break
		}
		chars = append(chars, char)
	}
	return string(utf16.Decode(chars)), nil
}

func expectChar(reader io.Reader, expected rune) error {
	var char uint16
	if err := binary.Read(reader, binary.LittleEndian, &char); err != nil {
		return err
	}
	if rune(char) != expected {
		return fmt.Errorf("expected '%c', got: '%c'", expected, rune(char))
	}
	return nil
}

// Save saves POL file
func (p *PolFile) Save(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return p.SaveToWriter(file)
}

// SaveToWriter writes POL file to writer
func (p *PolFile) SaveToWriter(writer io.Writer) error {
	// Signature
	if err := binary.Write(writer, binary.LittleEndian, uint32(polSignature)); err != nil {
		return err
	}

	// Version
	if err := binary.Write(writer, binary.LittleEndian, uint32(polVersion)); err != nil {
		return err
	}

	// Sort entries
	keys := make([]string, 0, len(p.entries))
	for k := range p.entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// For each entry
	for _, dictKey := range keys {
		entry := p.entries[dictKey]
		casedKey := p.casePreservation[dictKey]
		parts := strings.SplitN(casedKey, "\\\\", 2)
		keyName := parts[0]
		valueName := ""
		if len(parts) > 1 {
			valueName = parts[1]
		}

		if err := writeEntry(writer, keyName, valueName, entry); err != nil {
			return err
		}
	}

	return nil
}

func writeEntry(writer io.Writer, keyName, valueName string, entry *polEntryData) error {
	// "["
	if err := binary.Write(writer, binary.LittleEndian, uint16('[')); err != nil {
		return err
	}

	// Key name
	if err := writeNullTerminatedUTF16(writer, keyName); err != nil {
		return err
	}

	// ";"
	if err := binary.Write(writer, binary.LittleEndian, uint16(';')); err != nil {
		return err
	}

	// Value name
	if err := writeNullTerminatedUTF16(writer, valueName); err != nil {
		return err
	}

	// ";"
	if err := binary.Write(writer, binary.LittleEndian, uint16(';')); err != nil {
		return err
	}

	// Type
	if err := binary.Write(writer, binary.LittleEndian, uint32(entry.Kind)); err != nil {
		return err
	}

	// ";"
	if err := binary.Write(writer, binary.LittleEndian, uint16(';')); err != nil {
		return err
	}

	// Data length
	if err := binary.Write(writer, binary.LittleEndian, uint32(len(entry.Data))); err != nil {
		return err
	}

	// ";"
	if err := binary.Write(writer, binary.LittleEndian, uint16(';')); err != nil {
		return err
	}

	// Data
	if _, err := writer.Write(entry.Data); err != nil {
		return err
	}

	// "]"
	if err := binary.Write(writer, binary.LittleEndian, uint16(']')); err != nil {
		return err
	}

	return nil
}

func writeNullTerminatedUTF16(writer io.Writer, str string) error {
	chars := utf16.Encode([]rune(str))
	for _, char := range chars {
		if err := binary.Write(writer, binary.LittleEndian, char); err != nil {
			return err
		}
	}
	// Null terminator
	return binary.Write(writer, binary.LittleEndian, uint16(0))
}

func (p *PolFile) getDictKey(key, value string) string {
	origCase := key + "\\\\" + value
	lowerCase := strings.ToLower(origCase)
	if _, exists := p.casePreservation[lowerCase]; !exists {
		p.casePreservation[lowerCase] = origCase
	}
	return lowerCase
}

// SetValue sets a value
func (p *PolFile) SetValue(key, value string, data interface{}, dataType ValueType) error {
	dictKey := p.getDictKey(key, value)

	entry, err := fromArbitrary(data, dataType)
	if err != nil {
		return err
	}

	p.entries[dictKey] = entry
	return nil
}

// GetValue reads a value
func (p *PolFile) GetValue(key, value string) (interface{}, ValueType, error) {
	dictKey := p.getDictKey(key, value)
	entry, ok := p.entries[dictKey]
	if !ok {
		return nil, 0, fmt.Errorf("value not found")
	}

	data, err := entry.asArbitrary()
	return data, entry.Kind, err
}

// ContainsValue checks if a value exists
func (p *PolFile) ContainsValue(key, value string) bool {
	dictKey := p.getDictKey(key, value)
	_, ok := p.entries[dictKey]
	return ok
}

// DeleteValue deletes a value
func (p *PolFile) DeleteValue(key, value string) {
	p.ForgetValue(key, value)
	dictKey := p.getDictKey(key, "**del."+value)
	p.entries[dictKey] = &polEntryData{
		Kind: DWORD,
		Data: []byte{32, 0, 0, 0}, // DWORD 32
	}
}

// ForgetValue completely forgets a value
func (p *PolFile) ForgetValue(key, value string) {
	dictKey := p.getDictKey(key, value)
	delete(p.entries, dictKey)

	deleterKey := p.getDictKey(key, "**del."+value)
	delete(p.entries, deleterKey)
}

// ClearKey clears a key
func (p *PolFile) ClearKey(key string) {
	// Forget all values
	for dictKey := range p.entries {
		casedKey := p.casePreservation[dictKey]
		parts := strings.SplitN(casedKey, "\\\\", 2)
		if len(parts) > 0 && strings.EqualFold(parts[0], key) {
			delete(p.entries, dictKey)
		}
	}

	// Add clear marker
	dictKey := p.getDictKey(key, "**delvals.")
	entry, _ := fromString(" ", false)
	p.entries[dictKey] = entry
}

// GetValueNames returns all value names in a key
func (p *PolFile) GetValueNames(key string) []string {
	var names []string
	keyLower := strings.ToLower(key)

	for dictKey := range p.entries {
		casedKey := p.casePreservation[dictKey]
		parts := strings.SplitN(casedKey, "\\\\", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], keyLower) {
			if !strings.HasPrefix(parts[1], "**") {
				names = append(names, parts[1])
			}
		}
	}

	return names
}

// Data conversion functions

func (e *polEntryData) asArbitrary() (interface{}, error) {
	switch e.Kind {
	case SZ, EXPAND_SZ:
		return e.asString(), nil
	case DWORD:
		return e.asDword(), nil
	case QWORD:
		return e.asQword(), nil
	case MULTI_SZ:
		return e.asMultiString(), nil
	default:
		return e.Data, nil
	}
}

func (e *polEntryData) asString() string {
	if len(e.Data) < 2 {
		return ""
	}
	// UTF-16LE decode
	chars := make([]uint16, len(e.Data)/2)
	buf := bytes.NewReader(e.Data)
	binary.Read(buf, binary.LittleEndian, &chars)

	// Take until null terminator
	for i, c := range chars {
		if c == 0 {
			chars = chars[:i]
			break
		}
	}
	return string(utf16.Decode(chars))
}

func (e *polEntryData) asDword() uint32 {
	if len(e.Data) < 4 {
		return 0
	}
	return binary.LittleEndian.Uint32(e.Data[:4])
}

func (e *polEntryData) asQword() uint64 {
	if len(e.Data) < 8 {
		return 0
	}
	return binary.LittleEndian.Uint64(e.Data[:8])
}

func (e *polEntryData) asMultiString() []string {
	if len(e.Data) < 2 {
		return []string{}
	}

	chars := make([]uint16, len(e.Data)/2)
	buf := bytes.NewReader(e.Data)
	binary.Read(buf, binary.LittleEndian, &chars)

	var strings []string
	var current []uint16

	for _, c := range chars {
		if c == 0 {
			if len(current) == 0 {
				break
			}
			strings = append(strings, string(utf16.Decode(current)))
			current = []uint16{}
		} else {
			current = append(current, c)
		}
	}

	return strings
}

func fromArbitrary(data interface{}, kind ValueType) (*polEntryData, error) {
	switch kind {
	case SZ:
		return fromString(data.(string), false)
	case EXPAND_SZ:
		return fromString(data.(string), true)
	case DWORD:
		return fromDword(data.(uint32)), nil
	case QWORD:
		return fromQword(data.(uint64)), nil
	case MULTI_SZ:
		return fromMultiString(data.([]string))
	default:
		return &polEntryData{Kind: kind, Data: data.([]byte)}, nil
	}
}

func fromString(text string, expand bool) (*polEntryData, error) {
	kind := SZ
	if expand {
		kind = EXPAND_SZ
	}

	chars := utf16.Encode([]rune(text))
	data := make([]byte, (len(chars)+1)*2)

	for i, c := range chars {
		binary.LittleEndian.PutUint16(data[i*2:], c)
	}
	// Null terminator is already zero

	return &polEntryData{Kind: kind, Data: data}, nil
}

func fromDword(value uint32) *polEntryData {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, value)
	return &polEntryData{Kind: DWORD, Data: data}
}

func fromQword(value uint64) *polEntryData {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, value)
	return &polEntryData{Kind: QWORD, Data: data}
}

func fromMultiString(strings []string) (*polEntryData, error) {
	totalLen := 0
	for _, s := range strings {
		totalLen += len(s) + 1 // +1 for null terminator
	}
	totalLen++ // Final null terminator

	data := make([]byte, totalLen*2)
	pos := 0

	for _, s := range strings {
		chars := utf16.Encode([]rune(s))
		for _, c := range chars {
			binary.LittleEndian.PutUint16(data[pos:], c)
			pos += 2
		}
		pos += 2 // Null terminator
	}
	// Final null terminator is already zero

	return &polEntryData{Kind: MULTI_SZ, Data: data}, nil
}
