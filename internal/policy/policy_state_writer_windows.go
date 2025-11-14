//go:build windows

package policy

import (
	"fmt"

	"gopolicy/internal/polfile"

	"golang.org/x/sys/windows/registry"
)

// SetPolicyState updates both registry and .pol file.
func SetPolicyState(source PolicySource, policy *AdmxPolicy, state PolicyState, options map[string]interface{}) error {
	if policy == nil {
		return fmt.Errorf("policy is nil")
	}

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

	if regSource, ok := source.(*RegistryPolicySource); ok {
		var section AdmxPolicySection
		if regSource.RootKey == registry.CURRENT_USER {
			section = User
		} else if regSource.RootKey == registry.LOCAL_MACHINE {
			section = Machine
		} else {
			return nil
		}

		if err := updatePolFile(section, policy, state, options); err != nil {
			fmt.Printf("⚠ Warning: .pol file could not be updated (%v), but registry was written\n", err)
		}
	}

	return nil
}

func updatePolFile(section AdmxPolicySection, policy *AdmxPolicy, state PolicyState, options map[string]interface{}) error {
	polPath, err := GetPolPath(section)
	if err != nil {
		return err
	}

	pol, err := polfile.Load(polPath)
	if err != nil {
		// If POL file is corrupted or can't be loaded, create a new empty one
		pol = polfile.NewPolFile()
	}

	switch state {
	case PolicyStateEnabled:
		return updatePolEnabled(pol, polPath, policy, options)
	case PolicyStateDisabled:
		return updatePolDisabled(pol, polPath, policy)
	case PolicyStateNotConfigured:
		return updatePolNotConfigured(pol, polPath, policy)
	}

	return nil
}

func updatePolEnabled(pol *polfile.PolFile, polPath string, policy *AdmxPolicy, options map[string]interface{}) error {
	if policy.AffectedValues == nil {
		if policy.RegistryValue != "" {
			if err := pol.SetValue(policy.RegistryKey, policy.RegistryValue, uint32(1), polfile.DWORD); err != nil {
				return err
			}
		}
	} else {
		if policy.AffectedValues.OnValue == nil && policy.RegistryValue != "" {
			if err := pol.SetValue(policy.RegistryKey, policy.RegistryValue, uint32(1), polfile.DWORD); err != nil {
				return err
			}
		}
		if err := applyPolFileRegistryList(pol, policy.AffectedValues, policy.RegistryKey, policy.RegistryValue, true); err != nil {
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

			optionData, hasOption := options[base.ID]
			if !hasOption {
				continue
			}

			switch e := element.(type) {
			case *DecimalPolicyElement:
				if e.StoreAsText {
					pol.SetValue(elemKey, base.RegistryValue, fmt.Sprintf("%v", optionData), polfile.SZ)
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
					pol.SetValue(elemKey, base.RegistryValue, dword, polfile.DWORD)
				}
			case *BooleanPolicyElement:
				checkState, _ := optionData.(bool)
				if e.AffectedRegistry != nil && e.AffectedRegistry.OnValue == nil && checkState {
					pol.SetValue(elemKey, base.RegistryValue, uint32(1), polfile.DWORD)
				}
				if e.AffectedRegistry != nil && e.AffectedRegistry.OffValue == nil && !checkState {
					pol.DeleteValue(elemKey, base.RegistryValue)
				}
				if e.AffectedRegistry != nil {
					applyPolFileRegistryList(pol, e.AffectedRegistry, elemKey, base.RegistryValue, checkState)
				}
			case *TextPolicyElement:
				str, _ := optionData.(string)
				regType := polfile.SZ
				if e.RegExpandSz {
					regType = polfile.EXPAND_SZ
				}
				pol.SetValue(elemKey, base.RegistryValue, str, regType)
			case *ListPolicyElement:
				if !e.NoPurgeOthers {
					pol.ClearKey(elemKey)
				}
				regType := polfile.SZ
				if e.RegExpandSz {
					regType = polfile.EXPAND_SZ
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
						writePolFileValue(pol, item.Value, elemKey, base.RegistryValue)
					}
					if item.ValueList != nil {
						applyPolFileValueList(pol, item.ValueList, elemKey)
					}
				}
			case *MultiTextPolicyElement:
				if strs, ok := optionData.([]string); ok {
					pol.SetValue(elemKey, base.RegistryValue, strs, polfile.MULTI_SZ)
				}
			}
		}
	}

	return pol.Save(polPath)
}

func updatePolDisabled(pol *polfile.PolFile, polPath string, policy *AdmxPolicy) error {
	if policy.AffectedValues != nil {
		if err := applyPolFileRegistryList(pol, policy.AffectedValues, policy.RegistryKey, policy.RegistryValue, false); err != nil {
			return err
		}
	} else if policy.RegistryValue != "" {
		if err := pol.SetValue(policy.RegistryKey, policy.RegistryValue, uint32(0), polfile.DWORD); err != nil {
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
			pol.DeleteValue(elemKey, base.RegistryValue)
		}
	}

	return pol.Save(polPath)
}

func updatePolNotConfigured(pol *polfile.PolFile, polPath string, policy *AdmxPolicy) error {
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

	return pol.Save(polPath)
}

func applyPolFileRegistryList(pol *polfile.PolFile, regList *PolicyRegistryList, defaultKey, defaultValue string, isOn bool) error {
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
		return writePolFileValue(pol, value, defaultKey, defaultValue)
	}
	if valueList != nil {
		return applyPolFileValueList(pol, valueList, defaultKey)
	}
	return nil
}

func writePolFileValue(pol *polfile.PolFile, value *PolicyRegistryValue, key, valueName string) error {
	switch value.RegistryType {
	case Delete:
		pol.DeleteValue(key, valueName)
	case Numeric:
		pol.SetValue(key, valueName, uint32(value.NumberValue), polfile.DWORD)
	default:
		pol.SetValue(key, valueName, value.StringValue, polfile.SZ)
	}
	return nil
}

func applyPolFileValueList(pol *polfile.PolFile, list *PolicyRegistrySingleList, defaultKey string) error {
	listKey := defaultKey
	if list.DefaultRegistryKey != "" {
		listKey = list.DefaultRegistryKey
	}

	for _, entry := range list.AffectedValues {
		entryKey := listKey
		if entry.RegistryKey != "" {
			entryKey = entry.RegistryKey
		}
		writePolFileValue(pol, entry.Value, entryKey, entry.RegistryValue)
	}
	return nil
}

func setPolicyEnabled(source PolicySource, policy *AdmxPolicy, options map[string]interface{}) error {
	if policy.AffectedValues == nil {
		if policy.RegistryValue != "" {
			if err := source.SetValue(policy.RegistryKey, policy.RegistryValue, uint32(1), RegDWord); err != nil {
				return fmt.Errorf("enable value could not be written: %w", err)
			}
		}
	} else {
		if policy.AffectedValues.OnValue == nil && policy.RegistryValue != "" {
			if err := source.SetValue(policy.RegistryKey, policy.RegistryValue, uint32(1), RegDWord); err != nil {
				return fmt.Errorf("enable value could not be written: %w", err)
			}
		}
		if err := applyRegistryList(source, policy.AffectedValues, policy.RegistryKey, policy.RegistryValue, true); err != nil {
			return fmt.Errorf("affected values could not be written: %w", err)
		}
	}

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

func setPolicyDisabled(source PolicySource, policy *AdmxPolicy) error {
	if policy.AffectedValues != nil && policy.AffectedValues.OffValue == nil && policy.RegistryValue != "" {
		if err := source.DeleteValue(policy.RegistryKey, policy.RegistryValue); err != nil {
			return err
		}
	}

	if err := applyRegistryList(source, policy.AffectedValues, policy.RegistryKey, policy.RegistryValue, false); err != nil {
		return err
	}

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

func setPolicyNotConfigured(source PolicySource, policy *AdmxPolicy) error {
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
}

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
