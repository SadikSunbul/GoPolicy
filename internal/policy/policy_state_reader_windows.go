package policy

import (
	"golang.org/x/sys/windows/registry"
)

// GetPolicyState reads the current policy from the .pol file or registry.
func GetPolicyState(source PolicySource, policy *AdmxPolicy) (PolicyState, map[string]interface{}, error) {
	if regSource, ok := source.(*RegistryPolicySource); ok {
		var section AdmxPolicySection
		if regSource.RootKey == registry.CURRENT_USER {
			section = User
		} else if regSource.RootKey == registry.LOCAL_MACHINE {
			section = Machine
		}

		if polPath, err := GetPolPath(section); err == nil {
			if pol, err := Load(polPath); err == nil {
				state, options := getPolicyStateFromPolFile(pol, policy)
				if state != PolicyStateNotConfigured {
					return state, options, nil
				}
			}
		}
	}

	if policy.RegistryValue != "" && !source.ContainsValue(policy.RegistryKey, policy.RegistryValue) {
		return PolicyStateNotConfigured, nil, nil
	}

	if policy.AffectedValues != nil {
		if isRegistryListPresent(source, policy.AffectedValues, policy.RegistryKey, policy.RegistryValue, true) {
			options := readPolicyElements(source, policy)
			return PolicyStateEnabled, options, nil
		}
		if isRegistryListPresent(source, policy.AffectedValues, policy.RegistryKey, policy.RegistryValue, false) {
			return PolicyStateDisabled, nil, nil
		}
	} else if policy.RegistryValue != "" {
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

// GetPolicyStateFromPolFilePublic is public wrapper for getPolicyStateFromPolFile
func GetPolicyStateFromPolFilePublic(pol *PolFile, policy *AdmxPolicy) (PolicyState, map[string]interface{}) {
	return getPolicyStateFromPolFile(pol, policy)
}

func getPolicyStateFromPolFile(pol *PolFile, rawPolicy *AdmxPolicy) (PolicyState, map[string]interface{}) {
	options := readPolicyElementsFromPolFile(pol, rawPolicy)

	if len(options) > 0 {
		return PolicyStateEnabled, options
	}

	if rawPolicy.RegistryValue != "" {
		if !pol.ContainsValue(rawPolicy.RegistryKey, rawPolicy.RegistryValue) {
			return PolicyStateNotConfigured, nil
		}

		val, _, err := pol.GetValue(rawPolicy.RegistryKey, rawPolicy.RegistryValue)
		if err == nil {
			if dw, ok := val.(uint32); ok {
				if dw == 1 {
					return PolicyStateEnabled, options
				}
				if dw == 0 {
					return PolicyStateDisabled, nil
				}
			}
			return PolicyStateEnabled, options
		}
	}

	if rawPolicy.AffectedValues != nil {
		if isPolFileListPresent(pol, rawPolicy.AffectedValues, rawPolicy.RegistryKey, rawPolicy.RegistryValue, true) {
			return PolicyStateEnabled, options
		}
		if isPolFileListPresent(pol, rawPolicy.AffectedValues, rawPolicy.RegistryKey, rawPolicy.RegistryValue, false) {
			return PolicyStateDisabled, nil
		}
	}

	return PolicyStateNotConfigured, nil
}

func isPolFileListPresent(pol *PolFile, regList *PolicyRegistryList, defaultKey, defaultValue string, checkOn bool) bool {
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
		return isPolFileValuePresent(pol, value, defaultKey, defaultValue)
	}
	if valueList != nil {
		return isPolFileListAllPresent(pol, valueList, defaultKey)
	}
	return false
}

func isPolFileValuePresent(pol *PolFile, value *PolicyRegistryValue, key, valueName string) bool {
	data, _, err := pol.GetValue(key, valueName)
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

func isPolFileListAllPresent(pol *PolFile, list *PolicyRegistrySingleList, defaultKey string) bool {
	listKey := defaultKey
	if list.DefaultRegistryKey != "" {
		listKey = list.DefaultRegistryKey
	}

	for _, entry := range list.AffectedValues {
		entryKey := listKey
		if entry.RegistryKey != "" {
			entryKey = entry.RegistryKey
		}
		if !isPolFileValuePresent(pol, entry.Value, entryKey, entry.RegistryValue) {
			return false
		}
	}
	return true
}

func readPolicyElementsFromPolFile(pol *PolFile, rawPolicy *AdmxPolicy) map[string]interface{} {
	options := make(map[string]interface{})
	if rawPolicy.Elements == nil {
		return options
	}

	for _, element := range rawPolicy.Elements {
		base := element.GetBase()
		elemKey := rawPolicy.RegistryKey
		if base.RegistryKey != "" {
			elemKey = base.RegistryKey
		}

		// skip values starting with **del. or **
		if base.RegistryValue != "" {
			if base.RegistryValue[0] == '*' && len(base.RegistryValue) > 1 && base.RegistryValue[1] == '*' {
				continue
			}
		}

		if !pol.ContainsValue(elemKey, base.RegistryValue) {
			continue
		}

		val, _, err := pol.GetValue(elemKey, base.RegistryValue)
		if err != nil {
			continue
		}

		switch e := element.(type) {
		case *DecimalPolicyElement:
			if e.StoreAsText {
				if str, ok := val.(string); ok {
					options[base.ID] = str
				}
			} else if dw, ok := val.(uint32); ok {
				options[base.ID] = dw
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
			for idx, item := range e.Items {
				if item.Value != nil && isPolFileValuePresent(pol, item.Value, elemKey, base.RegistryValue) {
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
					if v, _, err := pol.GetValue(elemKey, name); err == nil {
						if str, ok := v.(string); ok {
							dict[name] = str
						}
					}
				}
				options[base.ID] = dict
			} else {
				var items []string
				for _, name := range names {
					if v, _, err := pol.GetValue(elemKey, name); err == nil {
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
			} else if dw, ok := val.(uint32); ok {
				options[base.ID] = dw
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
