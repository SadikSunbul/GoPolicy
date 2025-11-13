package policy

// AdmxPolicySection represents policy sections
type AdmxPolicySection int

const (
	Machine AdmxPolicySection = 1
	User    AdmxPolicySection = 2
	Both    AdmxPolicySection = 3
)

// AdmxProduct product definition
type AdmxProduct struct {
	ID          string
	DisplayCode string
	Type        AdmxProductType
	Version     int
	Parent      *AdmxProduct
	DefinedIn   *AdmxFile
}

// AdmxProductType product type
type AdmxProductType int

const (
	Product AdmxProductType = iota
	MajorRevision
	MinorRevision
)

// AdmxSupportDefinition support definition
type AdmxSupportDefinition struct {
	ID          string
	DisplayCode string
	Logic       AdmxSupportLogicType
	Entries     []*AdmxSupportEntry
	DefinedIn   *AdmxFile
}

// AdmxSupportLogicType logic type
type AdmxSupportLogicType int

const (
	Blank AdmxSupportLogicType = iota
	AllOf
	AnyOf
)

// AdmxSupportEntry support entry
type AdmxSupportEntry struct {
	ProductID  string
	IsRange    bool
	MinVersion *int
	MaxVersion *int
}

// AdmxCategory category
type AdmxCategory struct {
	ID          string
	DisplayCode string
	ExplainCode string
	ParentID    string
	DefinedIn   *AdmxFile
}

// AdmxPolicy policy definition
type AdmxPolicy struct {
	ID              string
	Section         AdmxPolicySection
	CategoryID      string
	DisplayCode     string
	ExplainCode     string
	SupportedCode   string
	PresentationID  string
	ClientExtension string
	RegistryKey     string
	RegistryValue   string
	AffectedValues  *PolicyRegistryList
	Elements        []PolicyElement
	DefinedIn       *AdmxFile
}

// PolicyRegistryList registry value list
type PolicyRegistryList struct {
	OnValue      *PolicyRegistryValue
	OnValueList  *PolicyRegistrySingleList
	OffValue     *PolicyRegistryValue
	OffValueList *PolicyRegistrySingleList
}

// PolicyRegistrySingleList single registry list
type PolicyRegistrySingleList struct {
	DefaultRegistryKey string
	AffectedValues     []*PolicyRegistryListEntry
}

// PolicyRegistryValue registry value
type PolicyRegistryValue struct {
	RegistryType PolicyRegistryValueType
	StringValue  string
	NumberValue  uint32
}

// PolicyRegistryListEntry registry list entry
type PolicyRegistryListEntry struct {
	RegistryValue string
	RegistryKey   string
	Value         *PolicyRegistryValue
}

// PolicyRegistryValueType value type
type PolicyRegistryValueType int

const (
	Delete PolicyRegistryValueType = iota
	Numeric
	Text
)

// PolicyElement policy element (abstract)
type PolicyElement interface {
	GetID() string
	GetClientExtension() string
	GetRegistryKey() string
	GetRegistryValue() string
	GetElementType() string
}

// BasePolicyElement base policy element
type BasePolicyElement struct {
	ID              string
	ClientExtension string
	RegistryKey     string
	RegistryValue   string
	ElementType     string
}

func (b *BasePolicyElement) GetID() string              { return b.ID }
func (b *BasePolicyElement) GetClientExtension() string { return b.ClientExtension }
func (b *BasePolicyElement) GetRegistryKey() string     { return b.RegistryKey }
func (b *BasePolicyElement) GetRegistryValue() string   { return b.RegistryValue }
func (b *BasePolicyElement) GetElementType() string     { return b.ElementType }

// DecimalPolicyElement decimal element
type DecimalPolicyElement struct {
	BasePolicyElement
	Required    bool
	Minimum     uint32
	Maximum     uint32
	StoreAsText bool
	NoOverwrite bool
}

// BooleanPolicyElement boolean element
type BooleanPolicyElement struct {
	BasePolicyElement
	AffectedRegistry *PolicyRegistryList
}

// TextPolicyElement text element
type TextPolicyElement struct {
	BasePolicyElement
	Required    bool
	MaxLength   int
	RegExpandSz bool
	NoOverwrite bool
}

// ListPolicyElement list element
type ListPolicyElement struct {
	BasePolicyElement
	HasPrefix         bool
	NoPurgeOthers     bool
	RegExpandSz       bool
	UserProvidesNames bool
}

// EnumPolicyElement enum element
type EnumPolicyElement struct {
	BasePolicyElement
	Required bool
	Items    []*EnumPolicyElementItem
}

// EnumPolicyElementItem enum item
type EnumPolicyElementItem struct {
	DisplayCode string
	Value       *PolicyRegistryValue
	ValueList   *PolicyRegistrySingleList
}

// MultiTextPolicyElement multi-text element
type MultiTextPolicyElement struct {
	BasePolicyElement
}
