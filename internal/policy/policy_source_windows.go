//go:build windows

package policy

import (
	"fmt"
	"os/exec"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	advapi32                = windows.NewLazySystemDLL("advapi32.dll")
	procSendMessageTimeoutW = user32.NewProc("SendMessageTimeoutW")
	procRefreshPolicyEx     = advapi32.NewProc("RefreshPolicyEx")
)

const (
	HWND_BROADCAST   = 0xFFFF
	WM_SETTINGCHANGE = 0x001A
	SMTO_ABORTIFHUNG = 0x0002
	SMTO_NORMAL      = 0x0000
)

// notifyWindowsSettingChange performs WM_SETTINGCHANGE broadcast.
func notifyWindowsSettingChange() {
	procSendMessageTimeoutW.Call(
		uintptr(HWND_BROADCAST),
		uintptr(WM_SETTINGCHANGE),
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Policy"))),
		SMTO_ABORTIFHUNG|SMTO_NORMAL,
		5000,
		0,
	)
}

// refreshPolicyEx triggers Group Policy refresh.
func refreshPolicyEx(isMachine bool) {
	if procRefreshPolicyEx == nil {
		return
	}
	if err := procRefreshPolicyEx.Find(); err != nil {
		return
	}

	flag := uintptr(0)
	if isMachine {
		flag = 1
	}
	procRefreshPolicyEx.Call(flag, 0)
}

// restartExplorer restarts Windows Explorer to ensure UI picks up changes.
func restartExplorer() {
	cmd := exec.Command("taskkill", "/F", "/IM", "explorer.exe")
	cmd.Run()
	time.Sleep(500 * time.Millisecond)
	_ = exec.Command("explorer.exe").Start()
}

// PolicySource interface for Windows Registry access.
type PolicySource interface {
	ContainsValue(key, value string) bool
	GetValue(key, value string) (interface{}, error)
	SetValue(key, value string, data interface{}, valueType RegistryValueKind) error
	DeleteValue(key, value string) error
	GetValueNames(key string) ([]string, error)
	ClearKey(key string) error
}

// RegistryValueKind represents Windows Registry data types.
type RegistryValueKind int

const (
	RegString RegistryValueKind = iota
	RegExpandString
	RegDWord
	RegMultiString
)

// RegistryPolicySource implements real registry access.
type RegistryPolicySource struct {
	RootKey registry.Key
}

// NewRegistrySource returns a registry source for user or machine section.
func NewRegistrySource(section AdmxPolicySection) (*RegistryPolicySource, error) {
	var rootKey registry.Key
	switch section {
	case Machine:
		rootKey = registry.LOCAL_MACHINE
	case User:
		rootKey = registry.CURRENT_USER
	default:
		return nil, fmt.Errorf("unknown section: %d", section)
	}
	return &RegistryPolicySource{RootKey: rootKey}, nil
}

func (r *RegistryPolicySource) ContainsValue(keyPath, valueName string) bool {
	k, err := registry.OpenKey(r.RootKey, keyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	if valueName == "" {
		return true
	}

	_, _, err = k.GetValue(valueName, nil)
	return err == nil
}

func (r *RegistryPolicySource) GetValue(keyPath, valueName string) (interface{}, error) {
	k, err := registry.OpenKey(r.RootKey, keyPath, registry.QUERY_VALUE)
	if err != nil {
		return nil, err
	}
	defer k.Close()

	_, valType, err := k.GetValue(valueName, nil)
	if err != nil {
		return nil, err
	}

	switch valType {
	case registry.SZ, registry.EXPAND_SZ:
		str, _, err := k.GetStringValue(valueName)
		return str, err
	case registry.DWORD:
		dw, _, err := k.GetIntegerValue(valueName)
		return uint32(dw), err
	case registry.MULTI_SZ:
		strs, _, err := k.GetStringsValue(valueName)
		return strs, err
	default:
		val, _, err := k.GetValue(valueName, nil)
		return val, err
	}
}

func (r *RegistryPolicySource) SetValue(keyPath, valueName string, data interface{}, valueType RegistryValueKind) error {
	k, _, err := registry.CreateKey(r.RootKey, keyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("key cannot be created (%s): %w (administrator privileges may be required)", keyPath, err)
	}
	defer k.Close()

	var writeErr error
	switch valueType {
	case RegString:
		str, ok := data.(string)
		if !ok {
			str = fmt.Sprintf("%v", data)
		}
		writeErr = k.SetStringValue(valueName, str)
	case RegExpandString:
		str, ok := data.(string)
		if !ok {
			str = fmt.Sprintf("%v", data)
		}
		writeErr = k.SetExpandStringValue(valueName, str)
	case RegDWord:
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
		default:
			return fmt.Errorf("invalid data type for DWORD: %T", data)
		}
		writeErr = k.SetDWordValue(valueName, dword)
	case RegMultiString:
		strs, ok := data.([]string)
		if !ok {
			return fmt.Errorf("invalid data type for MultiString: %T", data)
		}
		writeErr = k.SetStringsValue(valueName, strs)
	default:
		return fmt.Errorf("unsupported registry type: %d", valueType)
	}

	if writeErr != nil {
		return writeErr
	}

	notifyWindowsSettingChange()
	refreshPolicyEx(r.RootKey == registry.LOCAL_MACHINE)
	restartExplorer()
	return nil
}

func (r *RegistryPolicySource) DeleteValue(keyPath, valueName string) error {
	k, err := registry.OpenKey(r.RootKey, keyPath, registry.SET_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil
		}
		return err
	}
	defer k.Close()

	err = k.DeleteValue(valueName)
	if err == registry.ErrNotExist {
		return nil
	}
	if err != nil {
		return err
	}

	notifyWindowsSettingChange()
	refreshPolicyEx(r.RootKey == registry.LOCAL_MACHINE)
	restartExplorer()
	return nil
}

func (r *RegistryPolicySource) GetValueNames(keyPath string) ([]string, error) {
	k, err := registry.OpenKey(r.RootKey, keyPath, registry.QUERY_VALUE|registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return nil, err
	}
	defer k.Close()
	return k.ReadValueNames(0)
}

func (r *RegistryPolicySource) ClearKey(keyPath string) error {
	k, err := registry.OpenKey(r.RootKey, keyPath, registry.SET_VALUE|registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil
		}
		return err
	}
	defer k.Close()

	names, err := k.ReadValueNames(0)
	if err != nil {
		return err
	}

	hasChanges := false
	for _, name := range names {
		if err := k.DeleteValue(name); err != nil && err != registry.ErrNotExist {
			return err
		}
		hasChanges = true
	}

	if hasChanges {
		notifyWindowsSettingChange()
		refreshPolicyEx(r.RootKey == registry.LOCAL_MACHINE)
		restartExplorer()
	}
	return nil
}
