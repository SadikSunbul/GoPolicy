package policy

import (
	"fmt"
)

// PolicySource policy source interface
type PolicySource interface {
	ContainsValue(key, value string) bool
	GetValue(key, value string) (interface{}, error)
	GetValueNames(key string) ([]string, error)
	SetValue(key, value string, data interface{}, dataType int) error
	DeleteValue(key, value string) error
	ForgetValue(key, value string) error
	ClearKey(key string) error
	ForgetKeyClearance(key string) error
	WillDeleteValue(key, value string) bool
}

// GetPolicyState determines the state of a policy
func GetPolicyState(source PolicySource, policy *PolicyPlusPolicy) PolicyState {
	enabledEvidence := 0.0
	disabledEvidence := 0.0
	rawpol := policy.RawPolicy

	checkOneVal := func(value *PolicyRegistryValue, key, valueName string, evidenceVar *float64) {
		if value == nil {
			return
		}
		if valuePresent(value, source, key, valueName) {
			*evidenceVar += 1.0
		}
	}

	checkValList := func(valList *PolicyRegistrySingleList, defaultKey string, evidenceVar *float64) {
		if valList == nil {
			return
		}
		listKey := defaultKey
		if valList.DefaultRegistryKey != "" {
			listKey = valList.DefaultRegistryKey
		}
		for _, regVal := range valList.AffectedValues {
			entryKey := listKey
			if regVal.RegistryKey != "" {
				entryKey = regVal.RegistryKey
			}
			checkOneVal(regVal.Value, entryKey, regVal.RegistryValue, evidenceVar)
		}
	}

	// Check policy's standard Registry values
	if rawpol.RegistryValue != "" {
		if rawpol.AffectedValues.OnValue == nil {
			checkOneVal(&PolicyRegistryValue{NumberValue: 1, RegistryType: Numeric},
				rawpol.RegistryKey, rawpol.RegistryValue, &enabledEvidence)
		} else {
			checkOneVal(rawpol.AffectedValues.OnValue,
				rawpol.RegistryKey, rawpol.RegistryValue, &enabledEvidence)
		}

		if rawpol.AffectedValues.OffValue == nil {
			checkOneVal(&PolicyRegistryValue{RegistryType: Delete},
				rawpol.RegistryKey, rawpol.RegistryValue, &disabledEvidence)
		} else {
			checkOneVal(rawpol.AffectedValues.OffValue,
				rawpol.RegistryKey, rawpol.RegistryValue, &disabledEvidence)
		}
	}

	checkValList(rawpol.AffectedValues.OnValueList, rawpol.RegistryKey, &enabledEvidence)
	checkValList(rawpol.AffectedValues.OffValueList, rawpol.RegistryKey, &disabledEvidence)

	// Check policy elements
	if rawpol.Elements != nil && len(rawpol.Elements) > 0 {
		deletedElements := 0.0
		presentElements := 0.0

		for _, elem := range rawpol.Elements {
			elemKey := rawpol.RegistryKey
			if elem.GetRegistryKey() != "" {
				elemKey = elem.GetRegistryKey()
			}

			if elem.GetElementType() == "list" {
				neededValues := 0
				if source.WillDeleteValue(elemKey, "") {
					deletedElements += 1.0
					neededValues = 1
				}
				names, _ := source.GetValueNames(elemKey)
				if len(names) > 0 {
					deletedElements -= float64(neededValues)
					presentElements += 1.0
				}
			} else if elem.GetElementType() == "boolean" {
				booleanElem := elem.(*BooleanPolicyElement)
				if source.WillDeleteValue(elemKey, elem.GetRegistryValue()) {
					deletedElements += 1.0
				} else {
					checkboxDisabled := 0.0
					checkOneVal(booleanElem.AffectedRegistry.OffValue, elemKey, elem.GetRegistryValue(), &checkboxDisabled)
					checkValList(booleanElem.AffectedRegistry.OffValueList, elemKey, &checkboxDisabled)
					deletedElements += checkboxDisabled * 0.1
					checkOneVal(booleanElem.AffectedRegistry.OnValue, elemKey, elem.GetRegistryValue(), &presentElements)
					checkValList(booleanElem.AffectedRegistry.OnValueList, elemKey, &presentElements)
				}
			} else {
				if source.WillDeleteValue(elemKey, elem.GetRegistryValue()) {
					deletedElements += 1.0
				} else if source.ContainsValue(elemKey, elem.GetRegistryValue()) {
					presentElements += 1.0
				}
			}
		}

		if presentElements > 0 {
			enabledEvidence += presentElements
		} else if deletedElements > 0 {
			disabledEvidence += deletedElements
		}
	}

	// Evaluate evidence
	if enabledEvidence > disabledEvidence {
		return Enabled
	} else if disabledEvidence > enabledEvidence {
		return Disabled
	} else if enabledEvidence == 0 {
		return NotConfigured
	}
	return Unknown
}

func valuePresent(value *PolicyRegistryValue, source PolicySource, key, valueName string) bool {
	switch value.RegistryType {
	case Delete:
		return source.WillDeleteValue(key, valueName)
	case Numeric:
		if !source.ContainsValue(key, valueName) {
			return false
		}
		sourceVal, err := source.GetValue(key, valueName)
		if err != nil {
			return false
		}
		// can be uint32 or int
		var numVal uint32
		switch v := sourceVal.(type) {
		case uint32:
			numVal = v
		case int:
			numVal = uint32(v)
		case int32:
			numVal = uint32(v)
		default:
			return false
		}
		return numVal == value.NumberValue
	case Text:
		if !source.ContainsValue(key, valueName) {
			return false
		}
		sourceVal, err := source.GetValue(key, valueName)
		if err != nil {
			return false
		}
		strVal, ok := sourceVal.(string)
		if !ok {
			return false
		}
		return strVal == value.StringValue
	}
	return false
}

// GetPolicyOptionStates gets element states of an enabled policy
func GetPolicyOptionStates(source PolicySource, policy *PolicyPlusPolicy) (map[string]interface{}, error) {
	state := make(map[string]interface{})
	if policy.RawPolicy.Elements == nil {
		return state, nil
	}

	for _, elem := range policy.RawPolicy.Elements {
		elemKey := policy.RawPolicy.RegistryKey
		if elem.GetRegistryKey() != "" {
			elemKey = elem.GetRegistryKey()
		}

		switch elem.GetElementType() {
		case "decimal":
			val, err := source.GetValue(elemKey, elem.GetRegistryValue())
			if err == nil {
				if numVal, ok := val.(uint32); ok {
					state[elem.GetID()] = numVal
				}
			}

		case "boolean":
			// Implementation simplified
			state[elem.GetID()] = source.ContainsValue(elemKey, elem.GetRegistryValue())

		case "text":
			val, err := source.GetValue(elemKey, elem.GetRegistryValue())
			if err == nil {
				state[elem.GetID()] = val
			}

		case "list":
			listElem := elem.(*ListPolicyElement)
			if listElem.UserProvidesNames {
				entries := make(map[string]string)
				names, _ := source.GetValueNames(elemKey)
				for _, name := range names {
					val, err := source.GetValue(elemKey, name)
					if err == nil {
						if strVal, ok := val.(string); ok {
							entries[name] = strVal
						}
					}
				}
				state[elem.GetID()] = entries
			} else {
				var entries []string
				if listElem.HasPrefix {
					n := 1
					for {
						valName := fmt.Sprintf("%s%d", elem.GetRegistryValue(), n)
						if !source.ContainsValue(elemKey, valName) {
							break
						}
						val, err := source.GetValue(elemKey, valName)
						if err == nil {
							if strVal, ok := val.(string); ok {
								entries = append(entries, strVal)
							}
						}
						n++
					}
				} else {
					names, _ := source.GetValueNames(elemKey)
					entries = names
				}
				state[elem.GetID()] = entries
			}

		case "enum":
			enumElem := elem.(*EnumPolicyElement)
			selectedIndex := -1
			for i, item := range enumElem.Items {
				if valuePresent(item.Value, source, elemKey, elem.GetRegistryValue()) {
					selectedIndex = i
					break
				}
			}
			state[elem.GetID()] = selectedIndex

		case "multiText":
			val, err := source.GetValue(elemKey, elem.GetRegistryValue())
			if err == nil {
				state[elem.GetID()] = val
			}
		}
	}

	return state, nil
}

// SetPolicyState sets the state of a policy
func SetPolicyState(source PolicySource, policy *PolicyPlusPolicy, policyState PolicyState, options map[string]interface{}) error {
	rawpol := policy.RawPolicy

	setValue := func(key, valueName string, value *PolicyRegistryValue) error {
		if value == nil {
			return nil
		}
		switch value.RegistryType {
		case Delete:
			return source.DeleteValue(key, valueName)
		case Numeric:
			return source.SetValue(key, valueName, value.NumberValue, 4) // REG_DWORD
		case Text:
			return source.SetValue(key, valueName, value.StringValue, 1) // REG_SZ
		}
		return nil
	}

	setSingleList := func(singleList *PolicyRegistrySingleList, defaultKey string) error {
		if singleList == nil {
			return nil
		}
		listKey := defaultKey
		if singleList.DefaultRegistryKey != "" {
			listKey = singleList.DefaultRegistryKey
		}
		for _, e := range singleList.AffectedValues {
			itemKey := listKey
			if e.RegistryKey != "" {
				itemKey = e.RegistryKey
			}
			if err := setValue(itemKey, e.RegistryValue, e.Value); err != nil {
				return err
			}
		}
		return nil
	}

	setList := func(list *PolicyRegistryList, defaultKey, defaultValue string, isOn bool) error {
		if list == nil {
			return nil
		}
		if isOn {
			if err := setValue(defaultKey, defaultValue, list.OnValue); err != nil {
				return err
			}
			return setSingleList(list.OnValueList, defaultKey)
		} else {
			if err := setValue(defaultKey, defaultValue, list.OffValue); err != nil {
				return err
			}
			return setSingleList(list.OffValueList, defaultKey)
		}
	}

	switch policyState {
	case Enabled:
		// Set main value
		if rawpol.AffectedValues.OnValue == nil && rawpol.RegistryValue != "" {
			if err := source.SetValue(rawpol.RegistryKey, rawpol.RegistryValue, uint32(1), 4); err != nil {
				return err
			}
		}
		if err := setList(rawpol.AffectedValues, rawpol.RegistryKey, rawpol.RegistryValue, true); err != nil {
			return err
		}

		// Set elements
		if rawpol.Elements != nil {
			for _, elem := range rawpol.Elements {
				elemKey := rawpol.RegistryKey
				if elem.GetRegistryKey() != "" {
					elemKey = elem.GetRegistryKey()
				}

				optionData, hasOption := options[elem.GetID()]
				if !hasOption {
					continue
				}

				switch elem.GetElementType() {
				case "decimal":
					decElem := elem.(*DecimalPolicyElement)
					numVal := optionData.(uint32)
					regType := 4 // REG_DWORD
					if decElem.StoreAsText {
						regType = 1 // REG_SZ
						optionData = fmt.Sprint(numVal)
					}
					source.SetValue(elemKey, elem.GetRegistryValue(), optionData, regType)

				case "text":
					textElem := elem.(*TextPolicyElement)
					regType := 1 // REG_SZ
					if textElem.RegExpandSz {
						regType = 2 // REG_EXPAND_SZ
					}
					source.SetValue(elemKey, elem.GetRegistryValue(), optionData, regType)

				case "list":
					listElem := elem.(*ListPolicyElement)
					if !listElem.NoPurgeOthers {
						source.ClearKey(elemKey)
					}
					// List writing implementation
					// Simplified

				case "enum":
					enumElem := elem.(*EnumPolicyElement)
					idx := optionData.(int)
					if idx >= 0 && idx < len(enumElem.Items) {
						selItem := enumElem.Items[idx]
						setValue(elemKey, elem.GetRegistryValue(), selItem.Value)
						setSingleList(selItem.ValueList, elemKey)
					}
				}
			}
		}

	case Disabled:
		// Delete main value or set disabled value
		if rawpol.AffectedValues.OffValue == nil && rawpol.RegistryValue != "" {
			source.DeleteValue(rawpol.RegistryKey, rawpol.RegistryValue)
		}
		setList(rawpol.AffectedValues, rawpol.RegistryKey, rawpol.RegistryValue, false)

		// Clear elements
		if rawpol.Elements != nil {
			for _, elem := range rawpol.Elements {
				elemKey := rawpol.RegistryKey
				if elem.GetRegistryKey() != "" {
					elemKey = elem.GetRegistryKey()
				}

				if elem.GetElementType() == "list" {
					source.ClearKey(elemKey)
				} else {
					source.DeleteValue(elemKey, elem.GetRegistryValue())
				}
			}
		}

	case NotConfigured:
		// Clear all values
		if rawpol.RegistryValue != "" {
			source.ForgetValue(rawpol.RegistryKey, rawpol.RegistryValue)
		}
	}

	return nil
}

// DeduplicatePolicies merges user and computer policies with the same properties
func DeduplicatePolicies(workspace *AdmxBundle) int {
	dedupeCount := 0

	// Group by category
	categoryGroups := make(map[*PolicyPlusCategory][]*PolicyPlusPolicy)
	for _, pol := range workspace.Policies {
		if pol.Category != nil {
			categoryGroups[pol.Category] = append(categoryGroups[pol.Category], pol)
		}
	}

	// Check policies in each category
	for _, policies := range categoryGroups {
		// Group by DisplayName
		nameGroups := make(map[string][]*PolicyPlusPolicy)
		for _, pol := range policies {
			nameGroups[pol.DisplayName] = append(nameGroups[pol.DisplayName], pol)
		}

		for _, nameGroup := range nameGroups {
			if len(nameGroup) != 2 {
				continue
			}

			a := nameGroup[0]
			b := nameGroup[1]

			// One must be User, one must be Machine
			if a.RawPolicy.Section+b.RawPolicy.Section != Both {
				continue
			}

			// Other properties must be the same
			if a.DisplayExplanation != b.DisplayExplanation {
				continue
			}
			if a.RawPolicy.RegistryKey != b.RawPolicy.RegistryKey {
				continue
			}

			// Merge - remove a, make b Both
			a.Category.Policies = removePolicy(a.Category.Policies, a)
			delete(workspace.Policies, a.UniqueID)
			b.RawPolicy.Section = Both
			dedupeCount++
		}
	}

	return dedupeCount
}

func removePolicy(slice []*PolicyPlusPolicy, item *PolicyPlusPolicy) []*PolicyPlusPolicy {
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
