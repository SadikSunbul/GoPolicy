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

// notifyWindowsSettingChange sends a registry change notification to Windows.
func notifyWindowsSettingChange() {
	// Send WM_SETTINGCHANGE message to all windows
	procSendMessageTimeoutW.Call(
		uintptr(HWND_BROADCAST),
		uintptr(WM_SETTINGCHANGE),
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Policy"))),
		SMTO_ABORTIFHUNG|SMTO_NORMAL,
		5000, // 5 second timeout
		0,
	)
}

// refreshPolicyEx refreshes the Group Policy (if it exists)
func refreshPolicyEx(isMachine bool) {
	// RefreshPolicyEx is not available in some Windows versions; continue silently if there is an error
	if procRefreshPolicyEx == nil {
		return
	}
	if err := procRefreshPolicyEx.Find(); err != nil {
		return
	}

	machineFlag := uintptr(0)
	if isMachine {
		machineFlag = 1
	}
	procRefreshPolicyEx.Call(machineFlag, 0)
}

// restartExplorer restarts Explorer (required for some registry changes)
func restartExplorer() {
	// Close Explorer
	cmd := exec.Command("taskkill", "/F", "/IM", "explorer.exe")
	cmd.Run() // Continue even if there is an error

	// Short wait
	time.Sleep(500 * time.Millisecond)

	// Restart Explorer
	cmd = exec.Command("explorer.exe")
	cmd.Start() // Start in the background, no error checking
}

// PolicySource interface for accessing the Windows Registry
type PolicySource interface {
	ContainsValue(key, value string) bool
	GetValue(key, value string) (interface{}, error)
	SetValue(key, value string, data interface{}, valueType RegistryValueKind) error
	DeleteValue(key, value string) error
	GetValueNames(key string) ([]string, error)
	ClearKey(key string) error
}

// RegistryValueKind Windows registry data types
type RegistryValueKind int

const (
	RegString RegistryValueKind = iota
	RegExpandString
	RegDWord
	RegMultiString
)

// RegistryPolicySource real Windows Registry access
type RegistryPolicySource struct {
	RootKey registry.Key // HKEY_LOCAL_MACHINE or HKEY_CURRENT_USER
}

// NewRegistrySource creates a PolicySource for User or Machine
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

// ContainsValue checks if a value exists in the registry
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

// GetValue gets a value from the registry
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

// SetValue sets a value in the registry
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

	// Notify Windows of the registry change
	notifyWindowsSettingChange()

	// Refresh the Group Policy (if it exists)
	isMachine := (r.RootKey == registry.LOCAL_MACHINE)
	refreshPolicyEx(isMachine)

	// Restart Explorer (required for some changes)
	restartExplorer()

	return nil
}

// DeleteValue deletes a value from the registry
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

	// Notify Windows of the registry change
	notifyWindowsSettingChange()

	// Refresh the Group Policy (if it exists)
	isMachine := (r.RootKey == registry.LOCAL_MACHINE)
	refreshPolicyEx(isMachine)

	// Restart Explorer (required for some changes)
	restartExplorer()

	return nil
}

// GetValueNames gets the names of the values in the registry
func (r *RegistryPolicySource) GetValueNames(keyPath string) ([]string, error) {
	k, err := registry.OpenKey(r.RootKey, keyPath, registry.QUERY_VALUE|registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return nil, err
	}
	defer k.Close()

	names, err := k.ReadValueNames(0)
	if err != nil {
		return nil, err
	}
	return names, nil
}

// ClearKey clears a key in the registry
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

	// Notify Windows of the registry change
	if hasChanges {
		notifyWindowsSettingChange()
		isMachine := (r.RootKey == registry.LOCAL_MACHINE)
		refreshPolicyEx(isMachine)
		restartExplorer()
	}

	return nil
}

// GetPolicyState reads the current policy state (.pol file takes precedence)
func GetPolicyState(source PolicySource, policy *AdmxPolicy) (PolicyState, map[string]interface{}, error) {
	// Try reading from the .pol file first
	if regSource, ok := source.(*RegistryPolicySource); ok {
		var section AdmxPolicySection
		if regSource.RootKey == registry.CURRENT_USER {
			section = User
		} else if regSource.RootKey == registry.LOCAL_MACHINE {
			section = Machine
		}

		if polPath, err := GetPolPath(section); err == nil {
			if pol, err := LoadPolFile(polPath); err == nil && len(pol.Entries) > 0 {
				state, options := getPolicyStateFromPol(pol, policy)
				if state != PolicyStateNotConfigured {
					return state, options, nil
				}
			}
		}
	}

	// If .pol file does not exist or cannot be found, read from the registry
	if policy.RegistryValue != "" {
		if !source.ContainsValue(policy.RegistryKey, policy.RegistryValue) {
			return PolicyStateNotConfigured, nil, nil
		}
	}

	// Determine the state based on the AffectedValues
	if policy.AffectedValues != nil {
		if isRegistryListPresent(source, policy.AffectedValues, policy.RegistryKey, policy.RegistryValue, true) {
			options := readPolicyElements(source, policy)
			return PolicyStateEnabled, options, nil
		}
		if isRegistryListPresent(source, policy.AffectedValues, policy.RegistryKey, policy.RegistryValue, false) {
			return PolicyStateDisabled, nil, nil
		}
	} else if policy.RegistryValue != "" {
		// Simple case: just check the registry value
		val, err := source.GetValue(policy.RegistryKey, policy.RegistryValue)
		if err == nil {
			if dw, ok := val.(uint32); ok && dw == 1 {
				options := readPolicyElements(source, policy)
				return PolicyStateEnabled, options, nil
			}
		}
	}

	return PolicyStateNotConfigured, nil, nil
}

// getPolicyStateFromPol reads the policy state from the .pol file
func getPolicyStateFromPol(pol *PolFile, policy *AdmxPolicy) (PolicyState, map[string]interface{}) {
	if policy.RegistryValue != "" {
		if !pol.ContainsValue(policy.RegistryKey, policy.RegistryValue) {
			return PolicyStateNotConfigured, nil
		}
	}

	// Determine the state based on the AffectedValues
	if policy.AffectedValues != nil {
		if isPolListPresent(pol, policy.AffectedValues, policy.RegistryKey, policy.RegistryValue, true) {
			options := readPolicyElementsFromPol(pol, policy)
			return PolicyStateEnabled, options
		}
		if isPolListPresent(pol, policy.AffectedValues, policy.RegistryKey, policy.RegistryValue, false) {
			return PolicyStateDisabled, nil
		}
	} else if policy.RegistryValue != "" {
		// Simple case: just check the registry value
		val, _, ok := pol.GetValue(policy.RegistryKey, policy.RegistryValue)
		if ok {
			if dw, ok := val.(uint32); ok && dw == 1 {
				options := readPolicyElementsFromPol(pol, policy)
				return PolicyStateEnabled, options
			}
			if dw, ok := val.(uint32); ok && dw == 0 {
				return PolicyStateDisabled, nil
			}
		}
	}

	return PolicyStateNotConfigured, nil
}

// isPolListPresent checks if a list is present in the .pol file
func isPolListPresent(pol *PolFile, regList *PolicyRegistryList, defaultKey, defaultValue string, checkOn bool) bool {
	var value *PolicyRegistryValue
	var valueList *PolicyRegistrySingleList

	if checkOn {
		value = regList.OnValue
		valueList = regList.OnValueList
	} else {
		value = regList.OffValue
		valueList = regList.OffValueList
	}

	if value != nil {
		return isPolValuePresent(pol, value, defaultKey, defaultValue)
	}
	if valueList != nil {
		return isPolListAllPresent(pol, valueList, defaultKey)
	}
	return false
}

// isPolValuePresent checks if a value is present in the .pol file
func isPolValuePresent(pol *PolFile, value *PolicyRegistryValue, key, valueName string) bool {
	if !pol.ContainsValue(key, valueName) {
		return false
	}
	data, _, ok := pol.GetValue(key, valueName)
	if !ok {
		return false
	}

	switch value.RegistryType {
	case Delete:
		return false
	case Numeric:
		if dw, ok := data.(uint32); ok {
			return dw == uint32(value.NumberValue)
		}
	default:
		if str, ok := data.(string); ok {
			return str == value.StringValue
		}
	}
	return false
}

// isPolListAllPresent checks if all values in a list are present in the .pol file
func isPolListAllPresent(pol *PolFile, list *PolicyRegistrySingleList, defaultKey string) bool {
	listKey := defaultKey
	if list.DefaultRegistryKey != "" {
		listKey = list.DefaultRegistryKey
	}

	for _, entry := range list.AffectedValues {
		entryKey := listKey
		if entry.RegistryKey != "" {
			entryKey = entry.RegistryKey
		}
		if !isPolValuePresent(pol, entry.Value, entryKey, entry.RegistryValue) {
			return false
		}
	}
	return true
}

// readPolicyElementsFromPol reads the policy elements from the .pol file
func readPolicyElementsFromPol(pol *PolFile, policy *AdmxPolicy) map[string]interface{} {
	options := make(map[string]interface{})
	if policy.Elements == nil {
		return options
	}

	for _, element := range policy.Elements {
		base := element.GetBase()
		elemKey := policy.RegistryKey
		if base.RegistryKey != "" {
			elemKey = base.RegistryKey
		}

		if !pol.ContainsValue(elemKey, base.RegistryValue) {
			continue
		}

		val, _, ok := pol.GetValue(elemKey, base.RegistryValue)
		if !ok {
			continue
		}

		switch e := element.(type) {
		case *DecimalPolicyElement:
			if e.StoreAsText {
				if str, ok := val.(string); ok {
					options[base.ID] = str
				}
			} else {
				if dw, ok := val.(uint32); ok {
					options[base.ID] = dw
				}
			}
		case *TextPolicyElement:
			if str, ok := val.(string); ok {
				options[base.ID] = str
			}
		case *BooleanPolicyElement:
			if dw, ok := val.(uint32); ok {
				options[base.ID] = (dw == 1)
			}
		case *EnumPolicyElement:
			// Find the selected enum item
			for idx, item := range e.Items {
				if item.Value != nil && isPolValuePresent(pol, item.Value, elemKey, base.RegistryValue) {
					options[base.ID] = idx
					break
				}
			}
		case *MultiTextPolicyElement:
			if strs, ok := val.([]string); ok {
				options[base.ID] = strs
			}
		case *ListPolicyElement:
			names := pol.GetValueNames(elemKey)
			if e.UserProvidesNames {
				dict := make(map[string]string)
				for _, name := range names {
					if v, _, ok := pol.GetValue(elemKey, name); ok {
						if str, ok := v.(string); ok {
							dict[name] = str
						}
					}
				}
				options[base.ID] = dict
			} else {
				var items []string
				for _, name := range names {
					if v, _, ok := pol.GetValue(elemKey, name); ok {
						if str, ok := v.(string); ok {
							items = append(items, str)
						}
					}
				}
				options[base.ID] = items
			}
		}
	}

	return options
}

// isRegistryListPresent checks if a list is present in the registry
func isRegistryListPresent(source PolicySource, regList *PolicyRegistryList, defaultKey, defaultValue string, checkOn bool) bool {
	var value *PolicyRegistryValue
	var valueList *PolicyRegistrySingleList

	if checkOn {
		value = regList.OnValue
		valueList = regList.OnValueList
	} else {
		value = regList.OffValue
		valueList = regList.OffValueList
	}

	if value != nil {
		return isValuePresent(source, value, defaultKey, defaultValue)
	}
	if valueList != nil {
		return isListAllPresent(source, valueList, defaultKey)
	}
	return false
}

// isValuePresent checks if a value is present in the registry
func isValuePresent(source PolicySource, value *PolicyRegistryValue, key, valueName string) bool {
	if !source.ContainsValue(key, valueName) {
		return false
	}
	data, err := source.GetValue(key, valueName)
	if err != nil {
		return false
	}

	switch value.RegistryType {
	case Delete:
		return false
	case Numeric:
		if dw, ok := data.(uint32); ok {
			return dw == uint32(value.NumberValue)
		}
	default:
		if str, ok := data.(string); ok {
			return str == value.StringValue
		}
	}
	return false
}

// isListAllPresent checks if all values in a list are present in the registry
func isListAllPresent(source PolicySource, list *PolicyRegistrySingleList, defaultKey string) bool {
	listKey := defaultKey
	if list.DefaultRegistryKey != "" {
		listKey = list.DefaultRegistryKey
	}

	for _, entry := range list.AffectedValues {
		entryKey := listKey
		if entry.RegistryKey != "" {
			entryKey = entry.RegistryKey
		}
		if !isValuePresent(source, entry.Value, entryKey, entry.RegistryValue) {
			return false
		}
	}
	return true
}

// readPolicyElements reads the policy elements from the registry
func readPolicyElements(source PolicySource, policy *AdmxPolicy) map[string]interface{} {
	options := make(map[string]interface{})
	if policy.Elements == nil {
		return options
	}

	for _, element := range policy.Elements {
		base := element.GetBase()
		elemKey := policy.RegistryKey
		if base.RegistryKey != "" {
			elemKey = base.RegistryKey
		}

		if !source.ContainsValue(elemKey, base.RegistryValue) {
			continue
		}

		val, err := source.GetValue(elemKey, base.RegistryValue)
		if err != nil {
			continue
		}

		switch e := element.(type) {
		case *DecimalPolicyElement:
			if e.StoreAsText {
				if str, ok := val.(string); ok {
					options[base.ID] = str
				}
			} else {
				if dw, ok := val.(uint32); ok {
					options[base.ID] = dw
				}
			}
		case *TextPolicyElement:
			if str, ok := val.(string); ok {
				options[base.ID] = str
			}
		case *BooleanPolicyElement:
			if dw, ok := val.(uint32); ok {
				options[base.ID] = (dw == 1)
			}
		case *EnumPolicyElement:
			// Find the selected enum item
			for idx, item := range e.Items {
				if item.Value != nil && isValuePresent(source, item.Value, elemKey, base.RegistryValue) {
					options[base.ID] = idx
					break
				}
			}
		case *MultiTextPolicyElement:
			if strs, ok := val.([]string); ok {
				options[base.ID] = strs
			}
		case *ListPolicyElement:
			names, err := source.GetValueNames(elemKey)
			if err == nil {
				if e.UserProvidesNames {
					dict := make(map[string]string)
					for _, name := range names {
						if v, err := source.GetValue(elemKey, name); err == nil {
							if str, ok := v.(string); ok {
								dict[name] = str
							}
						}
					}
					options[base.ID] = dict
				} else {
					var items []string
					for _, name := range names {
						if v, err := source.GetValue(elemKey, name); err == nil {
							if str, ok := v.(string); ok {
								items = append(items, str)
							}
						}
					}
					options[base.ID] = items
				}
			}
		}
	}

	return options
}

// SetPolicyState writes the policy state to the registry and .pol file
func SetPolicyState(source PolicySource, policy *AdmxPolicy, state PolicyState, options map[string]interface{}) error {
	if policy == nil {
		return fmt.Errorf("policy is nil")
	}

	// First write to the registry
	var err error
	switch state {
	case PolicyStateEnabled:
		err = setPolicyEnabled(source, policy, options)
	case PolicyStateDisabled:
		err = setPolicyDisabled(source, policy)
	case PolicyStateNotConfigured:
		err = setPolicyNotConfigured(source, policy)
	default:
		return fmt.Errorf("invalid state: %d", state)
	}

	if err != nil {
		return err
	}

	// Then update the .pol file (only for RegistryPolicySource)
	if regSource, ok := source.(*RegistryPolicySource); ok {
		var section AdmxPolicySection
		if regSource.RootKey == registry.CURRENT_USER {
			section = User
		} else if regSource.RootKey == registry.LOCAL_MACHINE {
			section = Machine
		} else {
			return nil // Unknown root key, do not update the pol file
		}

		if err := updatePolFile(section, policy, state, options); err != nil {
			// Pol update failed, but registry was written, only warn
			fmt.Printf("⚠ Warning: .pol file could not be updated (%v), but registry was written\n", err)
		}
	}

	return nil
}

// updatePolFile updates the .pol file
func updatePolFile(section AdmxPolicySection, policy *AdmxPolicy, state PolicyState, options map[string]interface{}) error {
	polPath, err := GetPolPath(section)
	if err != nil {
		return err
	}

	pol, err := LoadPolFile(polPath)
	if err != nil {
		return fmt.Errorf("pol file could not be loaded: %w", err)
	}

	switch state {
	case PolicyStateEnabled:
		return updatePolEnabled(pol, policy, options)
	case PolicyStateDisabled:
		return updatePolDisabled(pol, policy)
	case PolicyStateNotConfigured:
		return updatePolNotConfigured(pol, policy)
	}

	return nil
}

// updatePolEnabled writes the main value to the .pol file
func updatePolEnabled(pol *PolFile, policy *AdmxPolicy, options map[string]interface{}) error {
	// Write the main value
	if policy.AffectedValues == nil {
		if policy.RegistryValue != "" {
			if err := pol.SetValue(policy.RegistryKey, policy.RegistryValue, uint32(1), RegDWord); err != nil {
				return err
			}
		}
	} else {
		if policy.AffectedValues.OnValue == nil && policy.RegistryValue != "" {
			if err := pol.SetValue(policy.RegistryKey, policy.RegistryValue, uint32(1), RegDWord); err != nil {
				return err
			}
		}
		if err := applyPolRegistryList(pol, policy.AffectedValues, policy.RegistryKey, policy.RegistryValue, true); err != nil {
			return err
		}
	}

	// Write the elements
	if policy.Elements != nil {
		for _, element := range policy.Elements {
			base := element.GetBase()
			elemKey := policy.RegistryKey
			if base.RegistryKey != "" {
				elemKey = base.RegistryKey
			}

			optionData, hasOption := options[base.ID]
			if !hasOption {
				continue
			}

			switch e := element.(type) {
			case *DecimalPolicyElement:
				if e.StoreAsText {
					pol.SetValue(elemKey, base.RegistryValue, fmt.Sprintf("%v", optionData), RegString)
				} else {
					var dword uint32
					switch v := optionData.(type) {
					case uint32:
						dword = v
					case int:
						dword = uint32(v)
					case float64:
						dword = uint32(v)
					}
					pol.SetValue(elemKey, base.RegistryValue, dword, RegDWord)
				}
			case *BooleanPolicyElement:
				checkState, _ := optionData.(bool)
				if e.AffectedRegistry != nil && e.AffectedRegistry.OnValue == nil && checkState {
					pol.SetValue(elemKey, base.RegistryValue, uint32(1), RegDWord)
				}
				if e.AffectedRegistry != nil && e.AffectedRegistry.OffValue == nil && !checkState {
					pol.DeleteValue(elemKey, base.RegistryValue)
				}
				if e.AffectedRegistry != nil {
					applyPolRegistryList(pol, e.AffectedRegistry, elemKey, base.RegistryValue, checkState)
				}
			case *TextPolicyElement:
				str, _ := optionData.(string)
				regType := RegString
				if e.RegExpandSz {
					regType = RegExpandString
				}
				pol.SetValue(elemKey, base.RegistryValue, str, regType)
			case *ListPolicyElement:
				if !e.NoPurgeOthers {
					pol.ClearKey(elemKey)
				}
				regType := RegString
				if e.RegExpandSz {
					regType = RegExpandString
				}
				if e.UserProvidesNames {
					if dict, ok := optionData.(map[string]string); ok {
						for k, v := range dict {
							pol.SetValue(elemKey, k, v, regType)
						}
					}
				} else {
					if items, ok := optionData.([]string); ok {
						for idx, item := range items {
							valueName := fmt.Sprintf("%d", idx+1)
							if e.HasPrefix && base.RegistryValue != "" {
								valueName = base.RegistryValue + valueName
							}
							pol.SetValue(elemKey, valueName, item, regType)
						}
					}
				}
			case *EnumPolicyElement:
				if idx, ok := optionData.(int); ok && idx >= 0 && idx < len(e.Items) {
					item := e.Items[idx]
					if item.Value != nil {
						writePolValue(pol, item.Value, elemKey, base.RegistryValue)
					}
					if item.ValueList != nil {
						applyPolValueList(pol, item.ValueList, elemKey)
					}
				}
			case *MultiTextPolicyElement:
				if strs, ok := optionData.([]string); ok {
					pol.SetValue(elemKey, base.RegistryValue, strs, RegMultiString)
				}
			}
		}
	}

	return pol.Save()
}

// updatePolDisabled writes the main value to the .pol file
func updatePolDisabled(pol *PolFile, policy *AdmxPolicy) error {
	// Write the main value
	if policy.AffectedValues != nil {
		if err := applyPolRegistryList(pol, policy.AffectedValues, policy.RegistryKey, policy.RegistryValue, false); err != nil {
			return err
		}
	} else if policy.RegistryValue != "" {
		if err := pol.SetValue(policy.RegistryKey, policy.RegistryValue, uint32(0), RegDWord); err != nil {
			return err
		}
	}

	// Clear the elements
	if policy.Elements != nil {
		for _, element := range policy.Elements {
			base := element.GetBase()
			elemKey := policy.RegistryKey
			if base.RegistryKey != "" {
				elemKey = base.RegistryKey
			}
			pol.DeleteValue(elemKey, base.RegistryValue)
		}
	}

	return pol.Save()
}

// updatePolNotConfigured clears all relevant values from the .pol file
func updatePolNotConfigured(pol *PolFile, policy *AdmxPolicy) error {
	// Clear all relevant values
	if policy.RegistryValue != "" {
		pol.DeleteValue(policy.RegistryKey, policy.RegistryValue)
	}

	if policy.Elements != nil {
		for _, element := range policy.Elements {
			base := element.GetBase()
			elemKey := policy.RegistryKey
			if base.RegistryKey != "" {
				elemKey = base.RegistryKey
			}
			pol.DeleteValue(elemKey, base.RegistryValue)
		}
	}

	return pol.Save()
}

// applyPolRegistryList applies a registry list to the .pol file
func applyPolRegistryList(pol *PolFile, regList *PolicyRegistryList, defaultKey, defaultValue string, isOn bool) error {
	var value *PolicyRegistryValue
	var valueList *PolicyRegistrySingleList

	if isOn {
		value = regList.OnValue
		valueList = regList.OnValueList
	} else {
		value = regList.OffValue
		valueList = regList.OffValueList
	}

	if value != nil {
		return writePolValue(pol, value, defaultKey, defaultValue)
	}
	if valueList != nil {
		return applyPolValueList(pol, valueList, defaultKey)
	}
	return nil
}

// writePolValue writes a value to the .pol file
func writePolValue(pol *PolFile, value *PolicyRegistryValue, key, valueName string) error {
	switch value.RegistryType {
	case Delete:
		pol.DeleteValue(key, valueName)
	case Numeric:
		pol.SetValue(key, valueName, uint32(value.NumberValue), RegDWord)
	default:
		pol.SetValue(key, valueName, value.StringValue, RegString)
	}
	return nil
}

// applyPolValueList applies a value list to the .pol file
func applyPolValueList(pol *PolFile, list *PolicyRegistrySingleList, defaultKey string) error {
	listKey := defaultKey
	if list.DefaultRegistryKey != "" {
		listKey = list.DefaultRegistryKey
	}

	for _, entry := range list.AffectedValues {
		entryKey := listKey
		if entry.RegistryKey != "" {
			entryKey = entry.RegistryKey
		}
		writePolValue(pol, entry.Value, entryKey, entry.RegistryValue)
	}
	return nil
}

// setPolicyEnabled sets the policy state to enabled
func setPolicyEnabled(source PolicySource, policy *AdmxPolicy, options map[string]interface{}) error {
	// Set the main value - if AffectedValues is nil or OnValue is explicitly not set
	if policy.AffectedValues == nil {
		// If AffectedValues is nil, simple enable: write 1 to the registry value
		if policy.RegistryValue != "" {
			if err := source.SetValue(policy.RegistryKey, policy.RegistryValue, uint32(1), RegDWord); err != nil {
				return fmt.Errorf("enable value could not be written: %w", err)
			}
		}
	} else {
		// If AffectedValues is set, write the default value of 1 if OnValue is not explicitly set
		if policy.AffectedValues.OnValue == nil && policy.RegistryValue != "" {
			if err := source.SetValue(policy.RegistryKey, policy.RegistryValue, uint32(1), RegDWord); err != nil {
				return fmt.Errorf("enable value could not be written: %w", err)
			}
		}
		// Process the AffectedValues
		if err := applyRegistryList(source, policy.AffectedValues, policy.RegistryKey, policy.RegistryValue, true); err != nil {
			return fmt.Errorf("affected values could not be written: %w", err)
		}
	}

	// Write the elements
	if policy.Elements != nil {
		for _, element := range policy.Elements {
			base := element.GetBase()
			elemKey := policy.RegistryKey
			if base.RegistryKey != "" {
				elemKey = base.RegistryKey
			}

			optionData, hasOption := options[base.ID]
			if !hasOption {
				continue
			}

			switch e := element.(type) {
			case *DecimalPolicyElement:
				if e.StoreAsText {
					if err := source.SetValue(elemKey, base.RegistryValue, fmt.Sprintf("%v", optionData), RegString); err != nil {
						return err
					}
				} else {
					var dword uint32
					switch v := optionData.(type) {
					case uint32:
						dword = v
					case int:
						dword = uint32(v)
					case float64:
						dword = uint32(v)
					}
					if err := source.SetValue(elemKey, base.RegistryValue, dword, RegDWord); err != nil {
						return err
					}
				}
			case *BooleanPolicyElement:
				checkState, _ := optionData.(bool)
				if e.AffectedRegistry != nil && e.AffectedRegistry.OnValue == nil && checkState {
					if err := source.SetValue(elemKey, base.RegistryValue, uint32(1), RegDWord); err != nil {
						return err
					}
				}
				if e.AffectedRegistry != nil && e.AffectedRegistry.OffValue == nil && !checkState {
					if err := source.DeleteValue(elemKey, base.RegistryValue); err != nil {
						return err
					}
				}
				if e.AffectedRegistry != nil {
					if err := applyRegistryList(source, e.AffectedRegistry, elemKey, base.RegistryValue, checkState); err != nil {
						return err
					}
				}
			case *TextPolicyElement:
				str, _ := optionData.(string)
				regType := RegString
				if e.RegExpandSz {
					regType = RegExpandString
				}
				if err := source.SetValue(elemKey, base.RegistryValue, str, regType); err != nil {
					return err
				}
			case *ListPolicyElement:
				if !e.NoPurgeOthers {
					if err := source.ClearKey(elemKey); err != nil {
						return err
					}
				}
				regType := RegString
				if e.RegExpandSz {
					regType = RegExpandString
				}
				if e.UserProvidesNames {
					if dict, ok := optionData.(map[string]string); ok {
						for k, v := range dict {
							if err := source.SetValue(elemKey, k, v, regType); err != nil {
								return err
							}
						}
					}
				} else {
					if items, ok := optionData.([]string); ok {
						for idx, item := range items {
							valueName := item
							if e.HasPrefix {
								valueName = fmt.Sprintf("%s%d", base.RegistryValue, idx+1)
							}
							if err := source.SetValue(elemKey, valueName, item, regType); err != nil {
								return err
							}
						}
					}
				}
			case *EnumPolicyElement:
				selIdx, _ := optionData.(int)
				if selIdx >= 0 && selIdx < len(e.Items) {
					item := e.Items[selIdx]
					if item.Value != nil {
						if err := applyRegistryValue(source, elemKey, base.RegistryValue, item.Value); err != nil {
							return err
						}
					}
					if item.ValueList != nil {
						if err := applySingleList(source, item.ValueList, elemKey); err != nil {
							return err
						}
					}
				}
			case *MultiTextPolicyElement:
				if strs, ok := optionData.([]string); ok {
					if err := source.SetValue(elemKey, base.RegistryValue, strs, RegMultiString); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// setPolicyDisabled sets the policy state to disabled
func setPolicyDisabled(source PolicySource, policy *AdmxPolicy) error {
	// Clear the main value or set the off value
	if policy.AffectedValues != nil && policy.AffectedValues.OffValue == nil && policy.RegistryValue != "" {
		if err := source.DeleteValue(policy.RegistryKey, policy.RegistryValue); err != nil {
			return err
		}
	}

	// Set the AffectedValues to the off state
	if err := applyRegistryList(source, policy.AffectedValues, policy.RegistryKey, policy.RegistryValue, false); err != nil {
		return err
	}

	// Clear the elements
	if policy.Elements != nil {
		for _, element := range policy.Elements {
			base := element.GetBase()
			elemKey := policy.RegistryKey
			if base.RegistryKey != "" {
				elemKey = base.RegistryKey
			}

			switch e := element.(type) {
			case *ListPolicyElement:
				if err := source.ClearKey(elemKey); err != nil {
					return err
				}
			case *BooleanPolicyElement:
				if e.AffectedRegistry != nil && (e.AffectedRegistry.OffValue != nil || e.AffectedRegistry.OffValueList != nil) {
					if err := applyRegistryList(source, e.AffectedRegistry, elemKey, base.RegistryValue, false); err != nil {
						return err
					}
				} else {
					if err := source.DeleteValue(elemKey, base.RegistryValue); err != nil {
						return err
					}
				}
			default:
				if err := source.DeleteValue(elemKey, base.RegistryValue); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// setPolicyNotConfigured clears all relevant values from the registry
func setPolicyNotConfigured(source PolicySource, policy *AdmxPolicy) error {
	// Clear all relevant values from the registry
	if policy.RegistryValue != "" {
		if err := source.DeleteValue(policy.RegistryKey, policy.RegistryValue); err != nil {
			return err
		}
	}

	if policy.Elements != nil {
		for _, element := range policy.Elements {
			base := element.GetBase()
			elemKey := policy.RegistryKey
			if base.RegistryKey != "" {
				elemKey = base.RegistryKey
			}

			if _, ok := element.(*ListPolicyElement); ok {
				if err := source.ClearKey(elemKey); err != nil {
					return err
				}
			} else {
				if err := source.DeleteValue(elemKey, base.RegistryValue); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// applyRegistryList applies a registry list to the registry
func applyRegistryList(source PolicySource, regList *PolicyRegistryList, defaultKey, defaultValue string, isOn bool) error {
	if regList == nil {
		return nil
	}

	var value *PolicyRegistryValue
	var valueList *PolicyRegistrySingleList

	if isOn {
		value = regList.OnValue
		valueList = regList.OnValueList
	} else {
		value = regList.OffValue
		valueList = regList.OffValueList
	}

	if value != nil {
		if err := applyRegistryValue(source, defaultKey, defaultValue, value); err != nil {
			return err
		}
	}
	if valueList != nil {
		if err := applySingleList(source, valueList, defaultKey); err != nil {
			return err
		}
	}

	return nil
}

// applyRegistryValue applies a registry value to the registry
func applyRegistryValue(source PolicySource, key, valueName string, value *PolicyRegistryValue) error {
	if value == nil {
		return nil
	}

	switch value.RegistryType {
	case Delete:
		return source.DeleteValue(key, valueName)
	case Numeric:
		return source.SetValue(key, valueName, uint32(value.NumberValue), RegDWord)
	default:
		return source.SetValue(key, valueName, value.StringValue, RegString)
	}
	return nil
}

// applySingleList applies a single list to the registry
func applySingleList(source PolicySource, list *PolicyRegistrySingleList, defaultKey string) error {
	if list == nil {
		return nil
	}

	listKey := defaultKey
	if list.DefaultRegistryKey != "" {
		listKey = list.DefaultRegistryKey
	}

	for _, entry := range list.AffectedValues {
		entryKey := listKey
		if entry.RegistryKey != "" {
			entryKey = entry.RegistryKey
		}
		if err := applyRegistryValue(source, entryKey, entry.RegistryValue, entry.Value); err != nil {
			return err
		}
	}

	return nil
}

// PolicyState represents the state of a policy
type PolicyState int

const (
	PolicyStateNotConfigured PolicyState = 0
	PolicyStateDisabled      PolicyState = 1
	PolicyStateEnabled       PolicyState = 2
	PolicyStateUnknown       PolicyState = 3
)

// String returns the string representation of a PolicyState
func (ps PolicyState) String() string {
	switch ps {
	case PolicyStateNotConfigured:
		return "Not Configured"
	case PolicyStateDisabled:
		return "Disabled"
	case PolicyStateEnabled:
		return "Enabled"
	default:
		return "Unknown"
	}
}
