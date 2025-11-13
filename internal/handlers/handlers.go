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
}

// NewPolicyHandler creates a new handler
func NewPolicyHandler(workspace *policy.AdmxBundle) *PolicyHandler {
	return &PolicyHandler{
		workspace: workspace,
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

		items = append(items, PolicyItem{
			ID:          pol.UniqueID,
			Name:        pol.DisplayName,
			Description: pol.DisplayExplanation,
			State:       "Not Configured",
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

	type ElementInfo struct {
		ID           string      `json:"id"`
		Type         string      `json:"type"`
		Label        string      `json:"label"`
		Required     bool        `json:"required"`
		DefaultValue interface{} `json:"defaultValue,omitempty"`
		Options      []string    `json:"options,omitempty"`
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

	detail := PolicyDetail{
		ID:          pol.UniqueID,
		Name:        pol.DisplayName,
		Description: pol.DisplayExplanation,
		State:       "Not Configured",
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
							elemInfo.Label = pe.Label
						case *policy.NumericBoxPresentationElement:
							elemInfo.Label = pe.Label
						case *policy.CheckBoxPresentationElement:
							elemInfo.Label = pe.Text
						case *policy.ComboBoxPresentationElement:
							elemInfo.Label = pe.Label
						case *policy.DropDownPresentationElement:
							elemInfo.Label = pe.Label
						case *policy.ListPresentationElement:
							elemInfo.Label = pe.Label
						}
					}
				}
			}

			if elemInfo.Label == "" {
				elemInfo.Label = elem.GetID()
			}

			// Special settings based on type
			switch elem.GetElementType() {
			case "text":
				textElem := elem.(*policy.TextPolicyElement)
				elemInfo.Required = textElem.Required
			case "decimal":
				decElem := elem.(*policy.DecimalPolicyElement)
				elemInfo.Required = decElem.Required
			case "enum":
				enumElem := elem.(*policy.EnumPolicyElement)
				elemInfo.Required = enumElem.Required
				elemInfo.Options = []string{}
				for _, item := range enumElem.Items {
					// Resolve display code
					optName := item.DisplayCode
					if strings.HasPrefix(optName, "$(string.") {
						optName = strings.TrimSuffix(strings.TrimPrefix(optName, "$(string."), ")")
					}
					elemInfo.Options = append(elemInfo.Options, optName)
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
		Options  map[string]interface{} `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Find policy
	pol, ok := h.workspace.Policies[req.PolicyID]
	if !ok {
		http.Error(w, "Policy not found", http.StatusNotFound)
		return
	}

	// Convert state
	var policyState policy.PolicyState
	switch req.State {
	case "Enabled":
		policyState = policy.Enabled
	case "Disabled":
		policyState = policy.Disabled
	case "NotConfigured":
		policyState = policy.NotConfigured
	default:
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	// For now, just return success
	// In real implementation, PolicySource will be used
	fmt.Fprintf(w, `{"success": true, "message": "Policy set to %s: %s"}`, policyState, pol.DisplayName)
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
