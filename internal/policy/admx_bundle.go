package policy

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// AdmxBundle collection of ADMX files
type AdmxBundle struct {
	sourceFiles        map[*AdmxFile]*AdmlFile
	namespaces         map[string]*AdmxFile
	rawCategories      []*AdmxCategory
	rawProducts        []*AdmxProduct
	rawPolicies        []*AdmxPolicy
	rawSupport         []*AdmxSupportDefinition
	FlatCategories     map[string]*PolicyPlusCategory
	FlatProducts       map[string]*PolicyPlusProduct
	Categories         map[string]*PolicyPlusCategory
	Products           map[string]*PolicyPlusProduct
	Policies           map[string]*PolicyPlusPolicy
	SupportDefinitions map[string]*PolicyPlusSupport
}

// AdmxLoadFailure loading error
type AdmxLoadFailure struct {
	FailType AdmxLoadFailType
	AdmxPath string
	Info     string
}

// AdmxLoadFailType error type
type AdmxLoadFailType int

const (
	BadAdmxParse AdmxLoadFailType = iota
	BadAdmx
	NoAdml
	BadAdmlParse
	BadAdml
	DuplicateNamespace
)

func (f *AdmxLoadFailure) Error() string {
	msg := fmt.Sprintf("'%s' failed to load: ", f.AdmxPath)
	switch f.FailType {
	case BadAdmxParse:
		msg += "ADMX XML could not be parsed: " + f.Info
	case BadAdmx:
		msg += "ADMX invalid: " + f.Info
	case NoAdml:
		msg += "ADML file not found"
	case BadAdmlParse:
		msg += "ADML XML could not be parsed: " + f.Info
	case BadAdml:
		msg += "ADML invalid: " + f.Info
	case DuplicateNamespace:
		msg += f.Info + " namespace already in use"
	default:
		msg += "Unknown error"
	}
	return msg
}

// NewAdmxBundle creates a new bundle
func NewAdmxBundle() *AdmxBundle {
	return &AdmxBundle{
		sourceFiles:        make(map[*AdmxFile]*AdmlFile),
		namespaces:         make(map[string]*AdmxFile),
		rawCategories:      []*AdmxCategory{},
		rawProducts:        []*AdmxProduct{},
		rawPolicies:        []*AdmxPolicy{},
		rawSupport:         []*AdmxSupportDefinition{},
		FlatCategories:     make(map[string]*PolicyPlusCategory),
		FlatProducts:       make(map[string]*PolicyPlusProduct),
		Categories:         make(map[string]*PolicyPlusCategory),
		Products:           make(map[string]*PolicyPlusProduct),
		Policies:           make(map[string]*PolicyPlusPolicy),
		SupportDefinitions: make(map[string]*PolicyPlusSupport),
	}
}

// LoadFolder loads all ADMX files in a folder. languageCodes is a preference
// list; the loader will try each locale (and their base languages) until a
// matching ADML dosyası bulunur.
func (b *AdmxBundle) LoadFolder(path string, languageCodes ...string) ([]*AdmxLoadFailure, error) {
	if len(languageCodes) == 0 {
		languageCodes = []string{"en-US"}
	}

	failures := []*AdmxLoadFailure{}

	err := filepath.WalkDir(path, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continue even if there is an error
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(filePath), ".admx") {
			if fail := b.addSingleAdmx(filePath, languageCodes); fail != nil {
				failures = append(failures, fail)
			}
		}
		return nil
	})

	if err != nil {
		return failures, err
	}

	b.buildStructures()
	return failures, nil
}

// LoadFile loads a single ADMX file using the provided locale preference list.
func (b *AdmxBundle) LoadFile(path string, languageCodes ...string) ([]*AdmxLoadFailure, error) {
	if len(languageCodes) == 0 {
		languageCodes = []string{"en-US"}
	}

	failures := []*AdmxLoadFailure{}
	if fail := b.addSingleAdmx(path, languageCodes); fail != nil {
		failures = append(failures, fail)
	}
	b.buildStructures()
	return failures, nil
}

func (b *AdmxBundle) addSingleAdmx(admxPath string, languageCodes []string) *AdmxLoadFailure {
	// Load ADMX
	admx, err := LoadAdmxFile(admxPath)
	if err != nil {
		return &AdmxLoadFailure{
			FailType: BadAdmxParse,
			AdmxPath: admxPath,
			Info:     err.Error(),
		}
	}

	// Check namespace
	if _, exists := b.namespaces[admx.AdmxNamespace]; exists {
		return &AdmxLoadFailure{
			FailType: DuplicateNamespace,
			AdmxPath: admxPath,
			Info:     admx.AdmxNamespace,
		}
	}

	admlPath, err := resolveAdmlPath(filepath.Dir(admxPath), filepath.Base(admxPath), languageCodes)
	if err != nil {
		return &AdmxLoadFailure{
			FailType: NoAdml,
			AdmxPath: admxPath,
			Info:     err.Error(),
		}
	}

	// Check ADML
	if _, err := os.Stat(admlPath); os.IsNotExist(err) {
		return &AdmxLoadFailure{
			FailType: NoAdml,
			AdmxPath: admxPath,
		}
	}

	// Load ADML
	adml, err := LoadAdmlFile(admlPath)
	if err != nil {
		return &AdmxLoadFailure{
			FailType: BadAdmlParse,
			AdmxPath: admxPath,
			Info:     err.Error(),
		}
	}

	// Stage for building
	b.rawCategories = append(b.rawCategories, admx.Categories...)
	b.rawProducts = append(b.rawProducts, admx.Products...)
	b.rawPolicies = append(b.rawPolicies, admx.Policies...)
	b.rawSupport = append(b.rawSupport, admx.SupportedOnDefinitions...)
	b.sourceFiles[admx] = adml
	b.namespaces[admx.AdmxNamespace] = admx

	return nil
}

func resolveAdmlPath(dir string, admxFileName string, languageCodes []string) (string, error) {
	base := strings.TrimSuffix(admxFileName, ".admx") + ".adml"

	// Some OEM images keep ADMLs next to ADMX files.
	directPath := filepath.Join(dir, base)
	if _, err := os.Stat(directPath); err == nil {
		return directPath, nil
	}

	candidateDirs := []string{dir}
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				candidateDirs = append(candidateDirs, filepath.Join(dir, entry.Name()))
			}
		}
	}

	tried := map[string]struct{}{}
	candidates := expandLocaleCandidates(languageCodes)

	for _, candidateDir := range candidateDirs {
		localPath := filepath.Join(candidateDir, base)
		if _, err := os.Stat(localPath); err == nil {
			return localPath, nil
		}

		for _, locale := range candidates {
			localePath := filepath.Join(candidateDir, locale, base)
			key := strings.ToLower(localePath)
			if _, ok := tried[key]; ok {
				continue
			}
			tried[key] = struct{}{}
			if _, err := os.Stat(localePath); err == nil {
				return localePath, nil
			}
		}
	}

	// final fallback to en-US regardless of duplicates
	fallback := filepath.Join(dir, "en-US", base)
	if _, err := os.Stat(fallback); err == nil {
		return fallback, nil
	}

	return "", fmt.Errorf("adml not found for %s", admxFileName)
}

func expandLocaleCandidates(languageCodes []string) []string {
	seen := map[string]struct{}{}
	var result []string
	add := func(loc string) {
		loc = strings.TrimSpace(loc)
		if loc == "" {
			return
		}
		key := strings.ToLower(loc)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		result = append(result, loc)
	}

	for _, loc := range languageCodes {
		add(loc)
		if idx := strings.Index(loc, "-"); idx > 0 {
			add(loc[:idx])
		}
	}
	add("en-US")
	return result
}

func (b *AdmxBundle) buildStructures() {
	catIds := make(map[string]*PolicyPlusCategory)
	productIds := make(map[string]*PolicyPlusProduct)
	supIds := make(map[string]*PolicyPlusSupport)
	polIds := make(map[string]*PolicyPlusPolicy)

	// First pass: Create structures
	for _, rawCat := range b.rawCategories {
		cat := &PolicyPlusCategory{
			DisplayName:        b.resolveString(rawCat.DisplayCode, rawCat.DefinedIn),
			DisplayExplanation: b.resolveString(rawCat.ExplainCode, rawCat.DefinedIn),
			UniqueID:           b.qualifyName(rawCat.ID, rawCat.DefinedIn),
			RawCategory:        rawCat,
			Children:           []*PolicyPlusCategory{},
			Policies:           []*PolicyPlusPolicy{},
		}
		catIds[cat.UniqueID] = cat
	}

	for _, rawProduct := range b.rawProducts {
		product := &PolicyPlusProduct{
			DisplayName: b.resolveString(rawProduct.DisplayCode, rawProduct.DefinedIn),
			UniqueID:    b.qualifyName(rawProduct.ID, rawProduct.DefinedIn),
			RawProduct:  rawProduct,
			Children:    []*PolicyPlusProduct{},
		}
		productIds[product.UniqueID] = product
	}

	for _, rawSup := range b.rawSupport {
		sup := &PolicyPlusSupport{
			DisplayName: b.resolveString(rawSup.DisplayCode, rawSup.DefinedIn),
			UniqueID:    b.qualifyName(rawSup.ID, rawSup.DefinedIn),
			RawSupport:  rawSup,
			Elements:    []*PolicyPlusSupportEntry{},
		}
		if rawSup.Entries != nil {
			for _, rawSupEntry := range rawSup.Entries {
				supEntry := &PolicyPlusSupportEntry{
					RawSupportEntry: rawSupEntry,
				}
				sup.Elements = append(sup.Elements, supEntry)
			}
		}
		supIds[sup.UniqueID] = sup
	}

	for _, rawPol := range b.rawPolicies {
		pol := &PolicyPlusPolicy{
			DisplayExplanation: b.resolveString(rawPol.ExplainCode, rawPol.DefinedIn),
			DisplayName:        b.resolveString(rawPol.DisplayCode, rawPol.DefinedIn),
			UniqueID:           b.qualifyName(rawPol.ID, rawPol.DefinedIn),
			RawPolicy:          rawPol,
		}
		if rawPol.PresentationID != "" {
			pol.Presentation = b.resolvePresentation(rawPol.PresentationID, rawPol.DefinedIn)
		}
		polIds[pol.UniqueID] = pol
	}

	// Second pass: Resolve references
	for _, cat := range catIds {
		if cat.RawCategory.ParentID != "" {
			parentCatName := b.resolveRef(cat.RawCategory.ParentID, cat.RawCategory.DefinedIn)
			if parentCat, ok := catIds[parentCatName]; ok {
				parentCat.Children = append(parentCat.Children, cat)
				cat.Parent = parentCat
			} else if parentCat, ok := b.FlatCategories[parentCatName]; ok {
				parentCat.Children = append(parentCat.Children, cat)
				cat.Parent = parentCat
			}
		}
	}

	for _, product := range productIds {
		if product.RawProduct.Parent != nil {
			parentProductID := b.qualifyName(product.RawProduct.Parent.ID, product.RawProduct.DefinedIn)
			if parentProduct, ok := productIds[parentProductID]; ok {
				parentProduct.Children = append(parentProduct.Children, product)
				product.Parent = parentProduct
			} else if parentProduct, ok := b.FlatProducts[parentProductID]; ok {
				parentProduct.Children = append(parentProduct.Children, product)
				product.Parent = parentProduct
			}
		}
	}

	for _, sup := range supIds {
		for _, supEntry := range sup.Elements {
			targetID := b.resolveRef(supEntry.RawSupportEntry.ProductID, sup.RawSupport.DefinedIn)
			if product, ok := productIds[targetID]; ok {
				supEntry.Product = product
			} else if product, ok := b.FlatProducts[targetID]; ok {
				supEntry.Product = product
			} else if supDef, ok := supIds[targetID]; ok {
				supEntry.SupportDefinition = supDef
			} else if supDef, ok := b.SupportDefinitions[targetID]; ok {
				supEntry.SupportDefinition = supDef
			}
		}
	}

	for _, pol := range polIds {
		catID := b.resolveRef(pol.RawPolicy.CategoryID, pol.RawPolicy.DefinedIn)
		if ownerCat, ok := catIds[catID]; ok {
			ownerCat.Policies = append(ownerCat.Policies, pol)
			pol.Category = ownerCat
		} else if ownerCat, ok := b.FlatCategories[catID]; ok {
			ownerCat.Policies = append(ownerCat.Policies, pol)
			pol.Category = ownerCat
		}

		supportID := b.resolveRef(pol.RawPolicy.SupportedCode, pol.RawPolicy.DefinedIn)
		if support, ok := supIds[supportID]; ok {
			pol.SupportedOn = support
		} else if support, ok := b.SupportDefinitions[supportID]; ok {
			pol.SupportedOn = support
		}
	}

	// Third pass: Add to final lists
	for k, v := range catIds {
		b.FlatCategories[k] = v
		if v.Parent == nil {
			b.Categories[k] = v
		}
	}

	for k, v := range productIds {
		b.FlatProducts[k] = v
		if v.Parent == nil {
			b.Products[k] = v
		}
	}

	for k, v := range polIds {
		b.Policies[k] = v
	}

	for k, v := range supIds {
		b.SupportDefinitions[k] = v
	}

	// Clean up
	b.rawCategories = nil
	b.rawProducts = nil
	b.rawPolicies = nil
	b.rawSupport = nil
}

func (b *AdmxBundle) resolveString(displayCode string, admx *AdmxFile) string {
	if displayCode == "" {
		return ""
	}
	if !strings.HasPrefix(displayCode, "$(string.") {
		return displayCode
	}
	stringID := displayCode[9 : len(displayCode)-1]
	if adml, ok := b.sourceFiles[admx]; ok {
		if str, ok := adml.StringTable[stringID]; ok {
			return str
		}
	}
	return displayCode
}

// ResolveString resolves a string code from ADML string table (public method)
func (b *AdmxBundle) ResolveString(displayCode string, admx *AdmxFile) string {
	return b.resolveString(displayCode, admx)
}

func (b *AdmxBundle) resolvePresentation(displayCode string, admx *AdmxFile) *Presentation {
	if !strings.HasPrefix(displayCode, "$(presentation.") {
		return nil
	}
	presID := displayCode[15 : len(displayCode)-1]
	if adml, ok := b.sourceFiles[admx]; ok {
		if pres, ok := adml.PresentationTable[presID]; ok {
			return pres
		}
	}
	return nil
}

func (b *AdmxBundle) qualifyName(id string, admx *AdmxFile) string {
	return admx.AdmxNamespace + ":" + id
}

func (b *AdmxBundle) resolveRef(ref string, admx *AdmxFile) string {
	if strings.Contains(ref, ":") {
		parts := strings.SplitN(ref, ":", 2)
		if ns, ok := admx.Prefixes[parts[0]]; ok {
			return ns + ":" + parts[1]
		}
		return ref
	}
	return b.qualifyName(ref, admx)
}
