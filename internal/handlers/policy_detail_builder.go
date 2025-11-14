package handlers

import (
	"gopolicy/internal/policy"
)

type PolicyDetailBuilder struct {
	workspace *policy.AdmxBundle
}

func NewPolicyDetailBuilder(workspace *policy.AdmxBundle) *PolicyDetailBuilder {
	return &PolicyDetailBuilder{workspace: workspace}
}

func (b *PolicyDetailBuilder) Build(pol *policy.PolicyPlusPolicy, state policy.PolicyState, options map[string]interface{}) PolicyDetail {
	detail := PolicyDetail{
		ID:          pol.UniqueID,
		Name:        pol.DisplayName,
		Description: pol.DisplayExplanation,
		State:       state.String(),
		Section:     sectionName(pol.RawPolicy.Section),
		Elements:    []ElementInfo{},
		RegistryKey: pol.RawPolicy.RegistryKey,
	}

	if pol.RawPolicy.Elements == nil {
		return detail
	}

	for _, elem := range pol.RawPolicy.Elements {
		elemInfo := b.buildElementInfo(pol, elem, options)
		detail.Elements = append(detail.Elements, elemInfo)
	}

	return detail
}

func (b *PolicyDetailBuilder) buildElementInfo(pol *policy.PolicyPlusPolicy, elem policy.PolicyElement, options map[string]interface{}) ElementInfo {
	elemInfo := ElementInfo{
		ID:       elem.GetID(),
		Type:     elem.GetElementType(),
		Label:    elem.GetID(),
		Metadata: map[string]interface{}{},
	}

	if pol.Presentation != nil {
		for _, presElem := range pol.Presentation.Elements {
			if presElem.GetID() != elem.GetID() {
				continue
			}
			b.applyPresentation(elemInfo.Metadata, &elemInfo, presElem, pol)
		}
	}

	b.applyElementType(elemInfo.Metadata, &elemInfo, elem, options, pol)

	return elemInfo
}

func (b *PolicyDetailBuilder) applyPresentation(metadata map[string]interface{}, elemInfo *ElementInfo, pres policy.PresentationElement, pol *policy.PolicyPlusPolicy) {
	switch pe := pres.(type) {
	case *policy.TextBoxPresentationElement:
		elemInfo.Label = b.resolveString(pe.Label, pol)
		if pe.DefaultValue != "" {
			elemInfo.DefaultValue = b.resolveString(pe.DefaultValue, pol)
		}
	case *policy.NumericBoxPresentationElement:
		elemInfo.Label = b.resolveString(pe.Label, pol)
		if pe.DefaultValue != 0 {
			elemInfo.DefaultValue = pe.DefaultValue
		}
	case *policy.CheckBoxPresentationElement:
		elemInfo.Label = b.resolveString(pe.Text, pol)
		elemInfo.DefaultValue = pe.DefaultState
	case *policy.ComboBoxPresentationElement:
		elemInfo.Label = b.resolveString(pe.Label, pol)
		if pe.DefaultText != "" {
			elemInfo.DefaultValue = b.resolveString(pe.DefaultText, pol)
		}
	case *policy.DropDownPresentationElement:
		elemInfo.Label = b.resolveString(pe.Label, pol)
	case *policy.ListPresentationElement:
		elemInfo.Label = b.resolveString(pe.Label, pol)
	case *policy.MultiTextPresentationElement:
		elemInfo.Label = b.resolveString(pe.Label, pol)
	}
}

func (b *PolicyDetailBuilder) applyElementType(metadata map[string]interface{}, elemInfo *ElementInfo, elem policy.PolicyElement, options map[string]interface{}, pol *policy.PolicyPlusPolicy) {
	switch elem.GetElementType() {
	case "text":
		textElem := elem.(*policy.TextPolicyElement)
		elemInfo.Required = textElem.Required
		if textElem.MaxLength > 0 {
			elemInfo.MaxLength = &textElem.MaxLength
		}
		metadata["expandable"] = textElem.RegExpandSz
		if val, ok := options[elemInfo.ID]; ok {
			if str, ok := val.(string); ok {
				elemInfo.DefaultValue = str
			}
		}
	case "decimal":
		decElem := elem.(*policy.DecimalPolicyElement)
		elemInfo.Required = decElem.Required
		if decElem.Minimum > 0 || decElem.Maximum < ^uint32(0) {
			elemInfo.MinValue = &decElem.Minimum
			if decElem.Maximum < ^uint32(0) {
				elemInfo.MaxValue = &decElem.Maximum
			}
		}
		metadata["storeAsText"] = decElem.StoreAsText
		if val, ok := options[elemInfo.ID]; ok {
			switch num := val.(type) {
			case uint32:
				elemInfo.DefaultValue = num
			case int:
				elemInfo.DefaultValue = uint32(num)
			}
		}
	case "boolean":
		boolElem := elem.(*policy.BooleanPolicyElement)
		metadata["hasAffectedRegistry"] = boolElem.AffectedRegistry != nil
		if val, ok := options[elemInfo.ID]; ok {
			if b, ok := val.(bool); ok {
				elemInfo.DefaultValue = b
			}
		}
	case "enum":
		enumElem := elem.(*policy.EnumPolicyElement)
		elemInfo.Required = enumElem.Required
		elemInfo.Options = []EnumOptionInfo{}
		for idx, item := range enumElem.Items {
			optName := b.resolveString(item.DisplayCode, pol)
			elemInfo.Options = append(elemInfo.Options, EnumOptionInfo{
				Index:       idx,
				DisplayName: optName,
			})
		}
		if val, ok := options[elemInfo.ID]; ok {
			if idx, ok := val.(int); ok {
				elemInfo.DefaultValue = idx
			}
		}
	case "list", "multiText":
		if val, ok := options[elemInfo.ID]; ok {
			elemInfo.DefaultValue = val
		}
		if elem.GetElementType() == "list" {
			listElem := elem.(*policy.ListPolicyElement)
			metadata["hasPrefix"] = listElem.HasPrefix
			metadata["userProvidesNames"] = listElem.UserProvidesNames
		} else {
			metadata["multiline"] = true
		}
	}
}

func (b *PolicyDetailBuilder) resolveString(code string, pol *policy.PolicyPlusPolicy) string {
	return b.workspace.ResolveString(code, pol.RawPolicy.DefinedIn)
}

func sectionName(section policy.AdmxPolicySection) string {
	switch section {
	case policy.Machine:
		return "Computer"
	case policy.User:
		return "User"
	default:
		return "Both"
	}
}
