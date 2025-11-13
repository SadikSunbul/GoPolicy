package policy

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// AdmxFile ADMX file
type AdmxFile struct {
	SourceFile             string
	AdmxNamespace          string
	SupersededAdm          string
	MinAdmlVersion         float64
	Prefixes               map[string]string
	Products               []*AdmxProduct
	SupportedOnDefinitions []*AdmxSupportDefinition
	Categories             []*AdmxCategory
	Policies               []*AdmxPolicy
}

// ADMX XML structures
type admxPolicyDefinitions struct {
	XMLName          xml.Name              `xml:"policyDefinitions"`
	PolicyNamespaces *admxPolicyNamespaces `xml:"policyNamespaces"`
	SupersededAdm    *admxSupersededAdm    `xml:"supersededAdm"`
	Resources        *admxResources        `xml:"resources"`
	SupportedOn      *admxSupportedOn      `xml:"supportedOn"`
	Categories       *admxCategories       `xml:"categories"`
	Policies         *admxPolicies         `xml:"policies"`
}

type admxPolicyNamespaces struct {
	Target admxNamespace   `xml:"target"`
	Usings []admxNamespace `xml:"using"`
}

type admxNamespace struct {
	Prefix    string `xml:"prefix,attr"`
	Namespace string `xml:"namespace,attr"`
}

type admxSupersededAdm struct {
	FileName string `xml:"fileName,attr"`
}

type admxResources struct {
	MinRequiredRevision string `xml:"minRequiredRevision,attr"`
}

type admxSupportedOn struct {
	Definitions *admxSupportDefinitions `xml:"definitions"`
	Products    *admxProducts           `xml:"products"`
}

type admxSupportDefinitions struct {
	Definitions []admxSupportDefinition `xml:"definition"`
}

type admxSupportDefinition struct {
	Name        string            `xml:"name,attr"`
	DisplayName string            `xml:"displayName,attr"`
	Or          *admxSupportLogic `xml:"or"`
	And         *admxSupportLogic `xml:"and"`
}

type admxSupportLogic struct {
	References []admxSupportReference `xml:"reference"`
	Ranges     []admxSupportRange     `xml:"range"`
}

type admxSupportReference struct {
	Ref string `xml:"ref,attr"`
}

type admxSupportRange struct {
	Ref             string `xml:"ref,attr"`
	MinVersionIndex string `xml:"minVersionIndex,attr"`
	MaxVersionIndex string `xml:"maxVersionIndex,attr"`
}

type admxProducts struct {
	Products []admxProductDef `xml:"product"`
}

type admxProductDef struct {
	Name          string             `xml:"name,attr"`
	DisplayName   string             `xml:"displayName,attr"`
	MajorVersions []admxMajorVersion `xml:"majorVersion"`
}

type admxMajorVersion struct {
	Name          string             `xml:"name,attr"`
	DisplayName   string             `xml:"displayName,attr"`
	VersionIndex  string             `xml:"versionIndex,attr"`
	MinorVersions []admxMinorVersion `xml:"minorVersion"`
}

type admxMinorVersion struct {
	Name         string `xml:"name,attr"`
	DisplayName  string `xml:"displayName,attr"`
	VersionIndex string `xml:"versionIndex,attr"`
}

type admxCategories struct {
	Categories []admxCategoryDef `xml:"category"`
}

type admxCategoryDef struct {
	Name           string              `xml:"name,attr"`
	DisplayName    string              `xml:"displayName,attr"`
	ExplainText    string              `xml:"explainText,attr"`
	ParentCategory *admxParentCategory `xml:"parentCategory"`
}

type admxParentCategory struct {
	Ref string `xml:"ref,attr"`
}

type admxPolicies struct {
	Policies []admxPolicyDef `xml:"policy"`
}

type admxPolicyDef struct {
	Name            string              `xml:"name,attr"`
	Class           string              `xml:"class,attr"`
	DisplayName     string              `xml:"displayName,attr"`
	ExplainText     string              `xml:"explainText,attr"`
	Key             string              `xml:"key,attr"`
	ValueName       string              `xml:"valueName,attr"`
	Presentation    string              `xml:"presentation,attr"`
	ClientExtension string              `xml:"clientExtension,attr"`
	ParentCategory  admxParentCategory  `xml:"parentCategory"`
	SupportedOn     *admxSupportedOnRef `xml:"supportedOn"`
	EnabledValue    *admxValue          `xml:"enabledValue"`
	DisabledValue   *admxValue          `xml:"disabledValue"`
	EnabledList     *admxValueList      `xml:"enabledList"`
	DisabledList    *admxValueList      `xml:"disabledList"`
	Elements        *admxElements       `xml:"elements"`
}

type admxSupportedOnRef struct {
	Ref string `xml:"ref,attr"`
}

type admxValue struct {
	Decimal *admxDecimalValue `xml:"decimal"`
	String  *admxStringValue  `xml:"string"`
	Delete  *struct{}         `xml:"delete"`
}

type admxDecimalValue struct {
	Value string `xml:"value,attr"`
}

type admxStringValue struct {
	Value string `xml:",chardata"`
}

type admxValueList struct {
	DefaultKey string          `xml:"defaultKey,attr"`
	Items      []admxValueItem `xml:"item"`
}

type admxValueItem struct {
	ValueName string     `xml:"valueName,attr"`
	Key       string     `xml:"key,attr"`
	Value     *admxValue `xml:"value"`
}

type admxElements struct {
	Decimals   []admxDecimalElement   `xml:"decimal"`
	Booleans   []admxBooleanElement   `xml:"boolean"`
	Texts      []admxTextElement      `xml:"text"`
	Lists      []admxListElement      `xml:"list"`
	Enums      []admxEnumElement      `xml:"enum"`
	MultiTexts []admxMultiTextElement `xml:"multiText"`
}

type admxDecimalElement struct {
	ID              string `xml:"id,attr"`
	ValueName       string `xml:"valueName,attr"`
	Key             string `xml:"key,attr"`
	MinValue        string `xml:"minValue,attr"`
	MaxValue        string `xml:"maxValue,attr"`
	Soft            string `xml:"soft,attr"`
	StoreAsText     string `xml:"storeAsText,attr"`
	ClientExtension string `xml:"clientExtension,attr"`
}

type admxBooleanElement struct {
	ID              string         `xml:"id,attr"`
	ValueName       string         `xml:"valueName,attr"`
	Key             string         `xml:"key,attr"`
	ClientExtension string         `xml:"clientExtension,attr"`
	TrueValue       *admxValue     `xml:"trueValue"`
	FalseValue      *admxValue     `xml:"falseValue"`
	TrueList        *admxValueList `xml:"trueList"`
	FalseList       *admxValueList `xml:"falseList"`
}

type admxTextElement struct {
	ID              string `xml:"id,attr"`
	ValueName       string `xml:"valueName,attr"`
	Key             string `xml:"key,attr"`
	MaxLength       string `xml:"maxLength,attr"`
	Required        string `xml:"required,attr"`
	Expandable      string `xml:"expandable,attr"`
	Soft            string `xml:"soft,attr"`
	ClientExtension string `xml:"clientExtension,attr"`
}

type admxListElement struct {
	ID              string `xml:"id,attr"`
	Key             string `xml:"key,attr"`
	ValuePrefix     string `xml:"valuePrefix,attr"`
	Additive        string `xml:"additive,attr"`
	Expandable      string `xml:"expandable,attr"`
	ExplicitValue   string `xml:"explicitValue,attr"`
	ClientExtension string `xml:"clientExtension,attr"`
}

type admxEnumElement struct {
	ID              string         `xml:"id,attr"`
	ValueName       string         `xml:"valueName,attr"`
	Key             string         `xml:"key,attr"`
	Required        string         `xml:"required,attr"`
	ClientExtension string         `xml:"clientExtension,attr"`
	Items           []admxEnumItem `xml:"item"`
}

type admxEnumItem struct {
	DisplayName string         `xml:"displayName,attr"`
	Value       *admxValue     `xml:"value"`
	ValueList   *admxValueList `xml:"valueList"`
}

type admxMultiTextElement struct {
	ID              string `xml:"id,attr"`
	ValueName       string `xml:"valueName,attr"`
	Key             string `xml:"key,attr"`
	ClientExtension string `xml:"clientExtension,attr"`
}

// LoadAdmxFile loads ADMX file
func LoadAdmxFile(path string) (*AdmxFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var policyDefs admxPolicyDefinitions
	if err := xml.Unmarshal(data, &policyDefs); err != nil {
		return nil, fmt.Errorf("XML parse error: %w", err)
	}

	admx := &AdmxFile{
		SourceFile:             path,
		Prefixes:               make(map[string]string),
		Products:               []*AdmxProduct{},
		SupportedOnDefinitions: []*AdmxSupportDefinition{},
		Categories:             []*AdmxCategory{},
		Policies:               []*AdmxPolicy{},
	}

	// Namespace information
	if policyDefs.PolicyNamespaces != nil {
		admx.AdmxNamespace = policyDefs.PolicyNamespaces.Target.Namespace
		admx.Prefixes[policyDefs.PolicyNamespaces.Target.Prefix] = policyDefs.PolicyNamespaces.Target.Namespace
		for _, using := range policyDefs.PolicyNamespaces.Usings {
			admx.Prefixes[using.Prefix] = using.Namespace
		}
	}

	// Superseded ADM
	if policyDefs.SupersededAdm != nil {
		admx.SupersededAdm = policyDefs.SupersededAdm.FileName
	}

	// Resources
	if policyDefs.Resources != nil && policyDefs.Resources.MinRequiredRevision != "" {
		admx.MinAdmlVersion, _ = strconv.ParseFloat(policyDefs.Resources.MinRequiredRevision, 64)
	}

	// Categories
	if policyDefs.Categories != nil {
		for _, cat := range policyDefs.Categories.Categories {
			category := &AdmxCategory{
				ID:          cat.Name,
				DisplayCode: cat.DisplayName,
				ExplainCode: cat.ExplainText,
				DefinedIn:   admx,
			}
			if cat.ParentCategory != nil {
				category.ParentID = cat.ParentCategory.Ref
			}
			admx.Categories = append(admx.Categories, category)
		}
	}

	// Products
	if policyDefs.SupportedOn != nil && policyDefs.SupportedOn.Products != nil {
		for _, prod := range policyDefs.SupportedOn.Products.Products {
			product := &AdmxProduct{
				ID:          prod.Name,
				DisplayCode: prod.DisplayName,
				Type:        Product,
				DefinedIn:   admx,
			}
			admx.Products = append(admx.Products, product)

			// Major versions
			for _, major := range prod.MajorVersions {
				majorProd := &AdmxProduct{
					ID:          major.Name,
					DisplayCode: major.DisplayName,
					Type:        MajorRevision,
					Parent:      product,
					DefinedIn:   admx,
				}
				if major.VersionIndex != "" {
					majorProd.Version, _ = strconv.Atoi(major.VersionIndex)
				}
				admx.Products = append(admx.Products, majorProd)

				// Minor versions
				for _, minor := range major.MinorVersions {
					minorProd := &AdmxProduct{
						ID:          minor.Name,
						DisplayCode: minor.DisplayName,
						Type:        MinorRevision,
						Parent:      majorProd,
						DefinedIn:   admx,
					}
					if minor.VersionIndex != "" {
						minorProd.Version, _ = strconv.Atoi(minor.VersionIndex)
					}
					admx.Products = append(admx.Products, minorProd)
				}
			}
		}
	}

	// Support Definitions
	if policyDefs.SupportedOn != nil && policyDefs.SupportedOn.Definitions != nil {
		for _, supDef := range policyDefs.SupportedOn.Definitions.Definitions {
			support := &AdmxSupportDefinition{
				ID:          supDef.Name,
				DisplayCode: supDef.DisplayName,
				Logic:       Blank,
				Entries:     []*AdmxSupportEntry{},
				DefinedIn:   admx,
			}

			var logic *admxSupportLogic
			if supDef.Or != nil {
				support.Logic = AnyOf
				logic = supDef.Or
			} else if supDef.And != nil {
				support.Logic = AllOf
				logic = supDef.And
			}

			if logic != nil {
				for _, ref := range logic.References {
					entry := &AdmxSupportEntry{
						ProductID: ref.Ref,
						IsRange:   false,
					}
					support.Entries = append(support.Entries, entry)
				}
				for _, rng := range logic.Ranges {
					entry := &AdmxSupportEntry{
						ProductID: rng.Ref,
						IsRange:   true,
					}
					if rng.MinVersionIndex != "" {
						min, _ := strconv.Atoi(rng.MinVersionIndex)
						entry.MinVersion = &min
					}
					if rng.MaxVersionIndex != "" {
						max, _ := strconv.Atoi(rng.MaxVersionIndex)
						entry.MaxVersion = &max
					}
					support.Entries = append(support.Entries, entry)
				}
			}

			admx.SupportedOnDefinitions = append(admx.SupportedOnDefinitions, support)
		}
	}

	// Policies
	if policyDefs.Policies != nil {
		for _, polDef := range policyDefs.Policies.Policies {
			policy := &AdmxPolicy{
				ID:              polDef.Name,
				DisplayCode:     polDef.DisplayName,
				ExplainCode:     polDef.ExplainText,
				CategoryID:      polDef.ParentCategory.Ref,
				RegistryKey:     polDef.Key,
				RegistryValue:   polDef.ValueName,
				PresentationID:  polDef.Presentation,
				ClientExtension: polDef.ClientExtension,
				DefinedIn:       admx,
				AffectedValues:  &PolicyRegistryList{},
			}

			// Section
			switch strings.ToLower(polDef.Class) {
			case "machine":
				policy.Section = Machine
			case "user":
				policy.Section = User
			default:
				policy.Section = Both
			}

			// Supported On
			if polDef.SupportedOn != nil {
				policy.SupportedCode = polDef.SupportedOn.Ref
			}

			// Enabled/Disabled values
			if polDef.EnabledValue != nil {
				policy.AffectedValues.OnValue = parseAdmxValue(polDef.EnabledValue)
			}
			if polDef.DisabledValue != nil {
				policy.AffectedValues.OffValue = parseAdmxValue(polDef.DisabledValue)
			}
			if polDef.EnabledList != nil {
				policy.AffectedValues.OnValueList = parseAdmxValueList(polDef.EnabledList)
			}
			if polDef.DisabledList != nil {
				policy.AffectedValues.OffValueList = parseAdmxValueList(polDef.DisabledList)
			}

			// Elements
			if polDef.Elements != nil {
				policy.Elements = parseAdmxElements(polDef.Elements)
			}

			admx.Policies = append(admx.Policies, policy)
		}
	}

	return admx, nil
}

func parseAdmxValue(val *admxValue) *PolicyRegistryValue {
	if val.Delete != nil {
		return &PolicyRegistryValue{RegistryType: Delete}
	}
	if val.Decimal != nil {
		num, _ := strconv.ParseUint(val.Decimal.Value, 10, 32)
		return &PolicyRegistryValue{
			RegistryType: Numeric,
			NumberValue:  uint32(num),
		}
	}
	if val.String != nil {
		return &PolicyRegistryValue{
			RegistryType: Text,
			StringValue:  val.String.Value,
		}
	}
	return nil
}

func parseAdmxValueList(list *admxValueList) *PolicyRegistrySingleList {
	result := &PolicyRegistrySingleList{
		DefaultRegistryKey: list.DefaultKey,
		AffectedValues:     []*PolicyRegistryListEntry{},
	}
	for _, item := range list.Items {
		entry := &PolicyRegistryListEntry{
			RegistryValue: item.ValueName,
			RegistryKey:   item.Key,
		}
		if item.Value != nil {
			entry.Value = parseAdmxValue(item.Value)
		}
		result.AffectedValues = append(result.AffectedValues, entry)
	}
	return result
}

func parseAdmxElements(elements *admxElements) []PolicyElement {
	var result []PolicyElement

	// Decimal elements
	for _, dec := range elements.Decimals {
		elem := &DecimalPolicyElement{
			BasePolicyElement: BasePolicyElement{
				ID:              dec.ID,
				RegistryValue:   dec.ValueName,
				RegistryKey:     dec.Key,
				ClientExtension: dec.ClientExtension,
				ElementType:     "decimal",
			},
			Maximum: ^uint32(0), // Max uint32
		}
		if dec.MinValue != "" {
			min, _ := strconv.ParseUint(dec.MinValue, 10, 32)
			elem.Minimum = uint32(min)
		}
		if dec.MaxValue != "" {
			max, _ := strconv.ParseUint(dec.MaxValue, 10, 32)
			elem.Maximum = uint32(max)
		}
		elem.StoreAsText = dec.StoreAsText == "true"
		elem.NoOverwrite = dec.Soft == "true"
		result = append(result, elem)
	}

	// Boolean elements
	for _, boo := range elements.Booleans {
		elem := &BooleanPolicyElement{
			BasePolicyElement: BasePolicyElement{
				ID:              boo.ID,
				RegistryValue:   boo.ValueName,
				RegistryKey:     boo.Key,
				ClientExtension: boo.ClientExtension,
				ElementType:     "boolean",
			},
			AffectedRegistry: &PolicyRegistryList{},
		}
		if boo.TrueValue != nil {
			elem.AffectedRegistry.OnValue = parseAdmxValue(boo.TrueValue)
		}
		if boo.FalseValue != nil {
			elem.AffectedRegistry.OffValue = parseAdmxValue(boo.FalseValue)
		}
		if boo.TrueList != nil {
			elem.AffectedRegistry.OnValueList = parseAdmxValueList(boo.TrueList)
		}
		if boo.FalseList != nil {
			elem.AffectedRegistry.OffValueList = parseAdmxValueList(boo.FalseList)
		}
		result = append(result, elem)
	}

	// Text elements
	for _, txt := range elements.Texts {
		elem := &TextPolicyElement{
			BasePolicyElement: BasePolicyElement{
				ID:              txt.ID,
				RegistryValue:   txt.ValueName,
				RegistryKey:     txt.Key,
				ClientExtension: txt.ClientExtension,
				ElementType:     "text",
			},
			MaxLength: 255,
		}
		if txt.MaxLength != "" {
			maxLen, _ := strconv.Atoi(txt.MaxLength)
			elem.MaxLength = maxLen
		}
		elem.Required = txt.Required == "true"
		elem.RegExpandSz = txt.Expandable == "true"
		elem.NoOverwrite = txt.Soft == "true"
		result = append(result, elem)
	}

	// List elements
	for _, lst := range elements.Lists {
		elem := &ListPolicyElement{
			BasePolicyElement: BasePolicyElement{
				ID:              lst.ID,
				RegistryValue:   lst.ValuePrefix,
				RegistryKey:     lst.Key,
				ClientExtension: lst.ClientExtension,
				ElementType:     "list",
			},
			HasPrefix:         lst.ValuePrefix != "",
			NoPurgeOthers:     lst.Additive == "true",
			RegExpandSz:       lst.Expandable == "true",
			UserProvidesNames: lst.ExplicitValue == "true",
		}
		result = append(result, elem)
	}

	// Enum elements
	for _, enm := range elements.Enums {
		elem := &EnumPolicyElement{
			BasePolicyElement: BasePolicyElement{
				ID:              enm.ID,
				RegistryValue:   enm.ValueName,
				RegistryKey:     enm.Key,
				ClientExtension: enm.ClientExtension,
				ElementType:     "enum",
			},
			Required: enm.Required == "true",
			Items:    []*EnumPolicyElementItem{},
		}
		for _, item := range enm.Items {
			enumItem := &EnumPolicyElementItem{
				DisplayCode: item.DisplayName,
			}
			if item.Value != nil {
				enumItem.Value = parseAdmxValue(item.Value)
			}
			if item.ValueList != nil {
				enumItem.ValueList = parseAdmxValueList(item.ValueList)
			}
			elem.Items = append(elem.Items, enumItem)
		}
		result = append(result, elem)
	}

	// MultiText elements
	for _, mt := range elements.MultiTexts {
		elem := &MultiTextPolicyElement{
			BasePolicyElement: BasePolicyElement{
				ID:              mt.ID,
				RegistryValue:   mt.ValueName,
				RegistryKey:     mt.Key,
				ClientExtension: mt.ClientExtension,
				ElementType:     "multiText",
			},
		}
		result = append(result, elem)
	}

	return result
}
