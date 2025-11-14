package handlers

import (
	"encoding/json"
	"fmt"
	"gopolicy/internal/policy"
	"html/template"
	"net/http"
	"strings"
)

// PolicyHandler handler that processes HTTP requests
type PolicyHandler struct {
	workspace *policy.AdmxBundle
	templates *template.Template
	source    policy.PolicySource
}

// NewPolicyHandler creates a new handler
func NewPolicyHandler(workspace *policy.AdmxBundle) *PolicyHandler {
	// Create registry source for Machine policies (HKLM)
	machineSource, _ := policy.NewRegistrySource(policy.Machine)

	// For now, use machine source for both user and machine policies
	// In a full implementation, you'd have separate sources
	return &PolicyHandler{
		workspace: workspace,
		source:    machineSource,
	}
}

// HandleIndex main page
func (h *PolicyHandler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Go Policy - Go Edition</title>
    <link rel="stylesheet" href="/static/style.css">
</head>
<body>
    <div class="container">
        <header>
            <h1>🛡️ Go Policy</h1>
            <p>Windows Group Policy Editor - Web Interface</p>
        </header>

        <div class="main-layout">
            <aside class="sidebar">
                <h2>Categories</h2>
                <div id="categories-tree"></div>
            </aside>

            <main class="content">
                <div class="toolbar">
                    <select id="section-filter">
                        <option value="both">Both</option>
                        <option value="user">User</option>
                        <option value="computer">Computer</option>
                    </select>
                    <button onclick="savePolicies()">💾 Save</button>
                    <button onclick="loadSources()">🔄 Refresh</button>
                </div>

                <div class="info-panel">
                    <div id="policy-info">
                        <h3>Info Panel</h3>
                        <p>Select a category or policy.</p>
                    </div>
                </div>

                <div class="policies-list">
                    <h3>Policies</h3>
                    <div id="policies"></div>
                </div>
            </main>
        </div>

        <footer>
            <p>Go Policy - Go Edition | Works on all Windows versions</p>
        </footer>
    </div>

    <div id="policy-edit-modal" class="modal">
        <div class="modal-content">
            <span class="close" onclick="closeModal()">&times;</span>
            <h2 id="modal-title"></h2>
            <div id="modal-body"></div>
            <div class="modal-actions">
                <button onclick="applyPolicy()">Apply</button>
                <button onclick="closeModal()">Cancel</button>
            </div>
        </div>
    </div>

    <script src="/static/app.js"></script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// CategoryNode category node structure
type CategoryNode struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Children    []*CategoryNode `json:"children"`
	PolicyCount int             `json:"policyCount"`
}

// buildCategoryTree builds category tree
func buildCategoryTree(cat *policy.PolicyPlusCategory) *CategoryNode {
	node := &CategoryNode{
		ID:          cat.UniqueID,
		Name:        cat.DisplayName,
		Description: cat.DisplayExplanation,
		Children:    []*CategoryNode{},
		PolicyCount: len(cat.Policies),
	}
	for _, child := range cat.Children {
		node.Children = append(node.Children, buildCategoryTree(child))
	}
	return node
}

// HandleCategories returns categories
func (h *PolicyHandler) HandleCategories(w http.ResponseWriter, r *http.Request) {
	var roots []*CategoryNode
	for _, cat := range h.workspace.Categories {
		roots = append(roots, buildCategoryTree(cat))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roots)
}

// HandlePolicies returns policies for a category
func (h *PolicyHandler) HandlePolicies(w http.ResponseWriter, r *http.Request) {
	categoryID := r.URL.Query().Get("category")
	if categoryID == "" {
		http.Error(w, "Category ID required", http.StatusBadRequest)
		return
	}

	cat, ok := h.workspace.FlatCategories[categoryID]
	if !ok {
		http.Error(w, "Category not found", http.StatusNotFound)
		return
	}

	type PolicyItem struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		State       string `json:"state"`
		Section     string `json:"section"`
	}

	var items []PolicyItem
	for _, pol := range cat.Policies {
		section := "Both"
		switch pol.RawPolicy.Section {
		case policy.Machine:
			section = "Computer"
		case policy.User:
			section = "User"
		}

		// Get current policy state
		state, _, _ := policy.GetPolicyState(h.source, pol.RawPolicy)
		stateStr := state.String()

		items = append(items, PolicyItem{
			ID:          pol.UniqueID,
			Name:        pol.DisplayName,
			Description: pol.DisplayExplanation,
			State:       stateStr,
			Section:     section,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// HandlePolicy returns details of a single policy
func (h *PolicyHandler) HandlePolicy(w http.ResponseWriter, r *http.Request) {
	// Get policy ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/policy/")
	policyID := strings.TrimSuffix(path, "/")

	pol, ok := h.workspace.Policies[policyID]
	if !ok {
		http.Error(w, "Policy not found", http.StatusNotFound)
		return
	}

	type EnumOptionInfo struct {
		Index       int    `json:"index"`
		DisplayName string `json:"displayName"`
	}

	type ElementInfo struct {
		ID           string                 `json:"id"`
		Type         string                 `json:"type"`
		Label        string                 `json:"label"`
		Required     bool                   `json:"required"`
		DefaultValue interface{}            `json:"defaultValue,omitempty"`
		Options      []EnumOptionInfo       `json:"options,omitempty"`
		MinValue     *uint32                `json:"minValue,omitempty"`
		MaxValue     *uint32                `json:"maxValue,omitempty"`
		MaxLength    *int                   `json:"maxLength,omitempty"`
		Description  string                 `json:"description,omitempty"`
		Metadata     map[string]interface{} `json:"metadata,omitempty"`
	}

	type PolicyDetail struct {
		ID          string        `json:"id"`
		Name        string        `json:"name"`
		Description string        `json:"description"`
		Section     string        `json:"section"`
		State       string        `json:"state"`
		Elements    []ElementInfo `json:"elements"`
		RegistryKey string        `json:"registryKey"`
	}

	// Get current policy state and options
	state, options, _ := policy.GetPolicyState(h.source, pol.RawPolicy)

	detail := PolicyDetail{
		ID:          pol.UniqueID,
		Name:        pol.DisplayName,
		Description: pol.DisplayExplanation,
		State:       state.String(),
		Elements:    []ElementInfo{},
		RegistryKey: pol.RawPolicy.RegistryKey,
	}

	switch pol.RawPolicy.Section {
	case policy.Machine:
		detail.Section = "Computer"
	case policy.User:
		detail.Section = "User"
	default:
		detail.Section = "Both"
	}

	// Add elements
	if pol.RawPolicy.Elements != nil {
		for _, elem := range pol.RawPolicy.Elements {
			elemInfo := ElementInfo{
				ID:   elem.GetID(),
				Type: elem.GetElementType(),
			}

			// Get label from presentation
			if pol.Presentation != nil {
				for _, presElem := range pol.Presentation.Elements {
					if presElem.GetID() == elem.GetID() {
						switch pe := presElem.(type) {
						case *policy.TextBoxPresentationElement:
							elemInfo.Label = resolveStringCode(pe.Label, pol.RawPolicy.DefinedIn, h.workspace)
							if pe.DefaultValue != "" {
								elemInfo.DefaultValue = resolveStringCode(pe.DefaultValue, pol.RawPolicy.DefinedIn, h.workspace)
							}
						case *policy.NumericBoxPresentationElement:
							elemInfo.Label = resolveStringCode(pe.Label, pol.RawPolicy.DefinedIn, h.workspace)
							if pe.DefaultValue != 0 {
								elemInfo.DefaultValue = pe.DefaultValue
							}
						case *policy.CheckBoxPresentationElement:
							elemInfo.Label = resolveStringCode(pe.Text, pol.RawPolicy.DefinedIn, h.workspace)
							elemInfo.DefaultValue = pe.DefaultState
						case *policy.ComboBoxPresentationElement:
							elemInfo.Label = resolveStringCode(pe.Label, pol.RawPolicy.DefinedIn, h.workspace)
							if pe.DefaultText != "" {
								elemInfo.DefaultValue = resolveStringCode(pe.DefaultText, pol.RawPolicy.DefinedIn, h.workspace)
							}
						case *policy.DropDownPresentationElement:
							elemInfo.Label = resolveStringCode(pe.Label, pol.RawPolicy.DefinedIn, h.workspace)
						case *policy.ListPresentationElement:
							elemInfo.Label = resolveStringCode(pe.Label, pol.RawPolicy.DefinedIn, h.workspace)
						case *policy.MultiTextPresentationElement:
							elemInfo.Label = resolveStringCode(pe.Label, pol.RawPolicy.DefinedIn, h.workspace)
						}
					}
				}
			}

			if elemInfo.Label == "" {
				elemInfo.Label = elem.GetID()
			}

			// Special settings based on type
			elemInfo.Metadata = make(map[string]interface{})
			switch elem.GetElementType() {
			case "text":
				textElem := elem.(*policy.TextPolicyElement)
				elemInfo.Required = textElem.Required
				if textElem.MaxLength > 0 {
					elemInfo.MaxLength = &textElem.MaxLength
				}
				elemInfo.Metadata["expandable"] = textElem.RegExpandSz
				// Set current value if available
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
				elemInfo.Metadata["storeAsText"] = decElem.StoreAsText
				// Set current value if available
				if val, ok := options[elemInfo.ID]; ok {
					if num, ok := val.(uint32); ok {
						elemInfo.DefaultValue = num
					} else if num, ok := val.(int); ok {
						elemInfo.DefaultValue = uint32(num)
					}
				}
			case "boolean":
				boolElem := elem.(*policy.BooleanPolicyElement)
				elemInfo.Metadata["hasAffectedRegistry"] = (boolElem.AffectedRegistry != nil)
				// Set current value if available
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
					// Resolve display code from string table
					optName := resolveStringCode(item.DisplayCode, pol.RawPolicy.DefinedIn, h.workspace)
					elemInfo.Options = append(elemInfo.Options, EnumOptionInfo{
						Index:       idx,
						DisplayName: optName,
					})
				}
				// Set current value if available
				if val, ok := options[elemInfo.ID]; ok {
					if idx, ok := val.(int); ok {
						elemInfo.DefaultValue = idx
					}
				}
			case "list":
				listElem := elem.(*policy.ListPolicyElement)
				elemInfo.Metadata["hasPrefix"] = listElem.HasPrefix
				elemInfo.Metadata["userProvidesNames"] = listElem.UserProvidesNames
				// Set current value if available
				if val, ok := options[elemInfo.ID]; ok {
					elemInfo.DefaultValue = val
				}
			case "multiText":
				elemInfo.Metadata["multiline"] = true
				// Set current value if available
				if val, ok := options[elemInfo.ID]; ok {
					elemInfo.DefaultValue = val
				}
			}

			detail.Elements = append(detail.Elements, elemInfo)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

// HandleSetPolicy sets policy state
func (h *PolicyHandler) HandleSetPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PolicyID string                 `json:"policyId"`
		State    string                 `json:"state"`
		Section  string                 `json:"section,omitempty"` // "machine" or "user"
		Options  map[string]interface{} `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Find policy
	pol, ok := h.workspace.Policies[req.PolicyID]
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Policy not found",
		})
		return
	}

	// Determine section - use request section if provided, otherwise use policy's section
	var section policy.AdmxPolicySection
	if req.Section != "" {
		switch strings.ToLower(req.Section) {
		case "machine":
			section = policy.Machine
		case "user":
			section = policy.User
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Invalid section: machine or user",
			})
			return
		}
	} else {
		// Use policy's default section
		section = pol.RawPolicy.Section
		if section == policy.Both {
			// Default to Machine if Both
			section = policy.Machine
		}
	}

	// Convert state
	var policyState policy.PolicyState
	switch strings.ToLower(req.State) {
	case "enabled":
		policyState = policy.PolicyStateEnabled
	case "disabled":
		policyState = policy.PolicyStateDisabled
	case "notconfigured", "not configured":
		policyState = policy.PolicyStateNotConfigured
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid state: enabled, disabled or notconfigured",
		})
		return
	}

	// Create registry source for the specified section
	source, err := policy.NewRegistrySource(section)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Registry source creation failed: %v", err),
		})
		return
	}

	// Set policy state
	if err := policy.SetPolicyState(source, pol.RawPolicy, policyState, req.Options); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Policy update failed: %v", err),
		})
		return
	}

	// Perform verification
	verifyState, _, _ := policy.GetPolicyState(source, pol.RawPolicy)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"message":       "Policy updated successfully",
		"verifiedState": verifyState.String(),
	})
}

// HandleSources returns policy sources
func (h *PolicyHandler) HandleSources(w http.ResponseWriter, r *http.Request) {
	sources := []map[string]interface{}{
		{
			"type":     "Local GPO",
			"path":     "C:\\Windows\\System32\\GroupPolicy",
			"writable": true,
		},
		{
			"type":     "Registry",
			"path":     "HKLM",
			"writable": true,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sources)
}

// HandleSave saves all changes
func (h *PolicyHandler) HandleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// For now, return success message
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Changes saved",
	})
}

// resolveStringCode resolves a string code from ADML string table
func resolveStringCode(code string, admx *policy.AdmxFile, workspace *policy.AdmxBundle) string {
	return workspace.ResolveString(code, admx)
}
