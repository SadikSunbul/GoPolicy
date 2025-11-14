package policy

// PolicyPlusCategory compiled category
type PolicyPlusCategory struct {
	UniqueID           string
	Parent             *PolicyPlusCategory
	Children           []*PolicyPlusCategory
	DisplayName        string
	DisplayExplanation string
	Policies           []*PolicyPlusPolicy
	RawCategory        *AdmxCategory
}

// PolicyPlusProduct compiled product
type PolicyPlusProduct struct {
	UniqueID    string
	Parent      *PolicyPlusProduct
	Children    []*PolicyPlusProduct
	DisplayName string
	RawProduct  *AdmxProduct
}

// PolicyPlusSupport compiled support
type PolicyPlusSupport struct {
	UniqueID    string
	DisplayName string
	Elements    []*PolicyPlusSupportEntry
	RawSupport  *AdmxSupportDefinition
}

// PolicyPlusSupportEntry support entry
type PolicyPlusSupportEntry struct {
	Product           *PolicyPlusProduct
	SupportDefinition *PolicyPlusSupport
	RawSupportEntry   *AdmxSupportEntry
}

// PolicyPlusPolicy compiled policy
type PolicyPlusPolicy struct {
	UniqueID           string
	Category           *PolicyPlusCategory
	DisplayName        string
	DisplayExplanation string
	SupportedOn        *PolicyPlusSupport
	Presentation       *Presentation
	RawPolicy          *AdmxPolicy
}
