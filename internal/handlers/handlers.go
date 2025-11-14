package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"gopolicy/internal/policy"
)

type SourceFactory func(policy.AdmxPolicySection) (policy.PolicySource, error)

type PolicyHandler struct {
	workspace     *policy.AdmxBundle
	renderer      pageRenderer
	source        policy.PolicySource
	sourceFactory SourceFactory
	detailBuilder *PolicyDetailBuilder
}

func NewPolicyHandler(workspace *policy.AdmxBundle) (*PolicyHandler, error) {
	source, err := policy.NewRegistrySource(policy.Machine)
	if err != nil {
		return nil, fmt.Errorf("kayıt kaynağı oluşturulamadı: %w", err)
	}

	return &PolicyHandler{
		workspace: workspace,
		renderer:  newDefaultRenderer(),
		source:    source,
		sourceFactory: func(section policy.AdmxPolicySection) (policy.PolicySource, error) {
			return policy.NewRegistrySource(section)
		},
		detailBuilder: NewPolicyDetailBuilder(workspace),
	}, nil
}

func (h *PolicyHandler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if err := h.renderer.Render(w, nil); err != nil {
		respondError(w, http.StatusInternalServerError, "Sayfa oluşturulamadı")
	}
}

func (h *PolicyHandler) HandleCategories(w http.ResponseWriter, r *http.Request) {
	var userRoots []*CategoryNode
	var computerRoots []*CategoryNode

	for _, cat := range h.workspace.Categories {
		// Check if category has user policies
		if hasPoliciesInSection(cat, policy.User) {
			userRoots = append(userRoots, buildCategoryTreeForSection(cat, policy.User))
		}
		// Check if category has computer policies
		if hasPoliciesInSection(cat, policy.Machine) {
			computerRoots = append(computerRoots, buildCategoryTreeForSection(cat, policy.Machine))
		}
	}

	sortCategoryNodes(userRoots)
	sortCategoryNodes(computerRoots)

	respondSuccess(w, CategoriesResponse{
		User:     userRoots,
		Computer: computerRoots,
	})
}

func (h *PolicyHandler) HandlePolicies(w http.ResponseWriter, r *http.Request) {
	categoryID := r.URL.Query().Get("category")
	if categoryID == "" {
		respondError(w, http.StatusBadRequest, "Category ID required")
		return
	}

	cat, ok := h.workspace.FlatCategories[categoryID]
	if !ok {
		respondError(w, http.StatusNotFound, "Category not found")
		return
	}

	items := make([]PolicyListItem, 0, len(cat.Policies))
	for _, pol := range cat.Policies {
		state, _, err := policy.GetPolicyState(h.source, pol.RawPolicy)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Policy state okunamadı")
			return
		}

		items = append(items, PolicyListItem{
			ID:          pol.UniqueID,
			Name:        pol.DisplayName,
			Description: pol.DisplayExplanation,
			State:       state.String(),
			Section:     sectionName(pol.RawPolicy.Section),
		})
	}

	respondSuccess(w, items)
}

func (h *PolicyHandler) HandlePolicy(w http.ResponseWriter, r *http.Request) {
	policyID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/policy/"), "/")
	pol, ok := h.workspace.Policies[policyID]
	if !ok {
		respondError(w, http.StatusNotFound, "Policy not found")
		return
	}

	state, options, err := policy.GetPolicyState(h.source, pol.RawPolicy)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Policy state okunamadı")
		return
	}

	detail := h.detailBuilder.Build(pol, state, options)
	respondSuccess(w, detail)
}

func (h *PolicyHandler) HandleSetPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req setPolicyRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	pol, ok := h.workspace.Policies[req.PolicyID]
	if !ok {
		respondError(w, http.StatusNotFound, "Policy not found")
		return
	}

	section, err := resolveSection(req.Section, pol.RawPolicy.Section)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	state, err := resolvePolicyState(req.State)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	source, err := h.sourceFactory(section)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Registry source creation failed")
		return
	}

	if err := policy.SetPolicyState(source, pol.RawPolicy, state, req.Options); err != nil {
		respondError(w, http.StatusInternalServerError, "Policy update failed")
		return
	}

	verifyState, _, err := policy.GetPolicyState(source, pol.RawPolicy)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Policy verify failed")
		return
	}

	respondSuccess(w, map[string]interface{}{
		"success":       true,
		"message":       "Policy updated successfully",
		"verifiedState": verifyState.String(),
	})
}

func (h *PolicyHandler) HandleSources(w http.ResponseWriter, r *http.Request) {
	respondSuccess(w, []map[string]interface{}{
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
	})
}

func (h *PolicyHandler) HandleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	respondSuccess(w, map[string]interface{}{
		"success": true,
		"message": "Changes saved",
	})
}

// HandleSearch searches policies by name or description
func (h *PolicyHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		respondError(w, http.StatusBadRequest, "Search query required")
		return
	}

	// Get section filter (both, user, or computer)
	sectionFilter := strings.ToLower(r.URL.Query().Get("section"))
	if sectionFilter == "" {
		sectionFilter = "both"
	}

	// Normalize query for case-insensitive search
	queryLower := strings.ToLower(query)

	var userResults []SearchResultItem
	var computerResults []SearchResultItem

	// Search through all policies
	for _, pol := range h.workspace.Policies {
		// Check if query matches name or description (case-insensitive)
		nameLower := strings.ToLower(pol.DisplayName)
		descLower := strings.ToLower(pol.DisplayExplanation)

		if !strings.Contains(nameLower, queryLower) && !strings.Contains(descLower, queryLower) {
			continue
		}

		// Get policy state
		state, _, err := policy.GetPolicyState(h.source, pol.RawPolicy)
		if err != nil {
			// Skip policies with errors
			continue
		}

		// Get category information
		categoryName := ""
		categoryID := ""
		if pol.Category != nil {
			categoryName = pol.Category.DisplayName
			categoryID = pol.Category.UniqueID
		}

		// Create search result item
		item := SearchResultItem{
			ID:           pol.UniqueID,
			Name:         pol.DisplayName,
			Description:  pol.DisplayExplanation,
			State:        state.String(),
			Section:      sectionName(pol.RawPolicy.Section),
			CategoryID:   categoryID,
			CategoryName: categoryName,
		}

		// Add to appropriate section based on filter
		section := pol.RawPolicy.Section

		// If filter is "user", only add user policies
		if sectionFilter == "user" {
			if section == policy.User || section == policy.Both {
				userResults = append(userResults, item)
			}
		} else if sectionFilter == "computer" {
			// If filter is "computer", only add computer policies
			if section == policy.Machine || section == policy.Both {
				computerResults = append(computerResults, item)
			}
		} else {
			// If filter is "both", add to both sections
			if section == policy.User || section == policy.Both {
				userResults = append(userResults, item)
			}
			if section == policy.Machine || section == policy.Both {
				computerResults = append(computerResults, item)
			}
		}
	}

	// Sort results by name
	sort.Slice(userResults, func(i, j int) bool {
		return userResults[i].Name < userResults[j].Name
	})
	sort.Slice(computerResults, func(i, j int) bool {
		return computerResults[i].Name < computerResults[j].Name
	})

	respondSuccess(w, SearchResponse{
		User:     userResults,
		Computer: computerResults,
		Query:    query,
		Total:    len(userResults) + len(computerResults),
	})
}

// hasPoliciesInSection checks if a category (or any of its children) has policies in the given section
func hasPoliciesInSection(cat *policy.PolicyPlusCategory, section policy.AdmxPolicySection) bool {
	// Check direct policies
	for _, pol := range cat.Policies {
		if pol.RawPolicy.Section == section || pol.RawPolicy.Section == policy.Both {
			return true
		}
	}
	// Check children
	for _, child := range cat.Children {
		if hasPoliciesInSection(child, section) {
			return true
		}
	}
	return false
}

// buildCategoryTreeForSection builds a category tree filtered by section
func buildCategoryTreeForSection(cat *policy.PolicyPlusCategory, section policy.AdmxPolicySection) *CategoryNode {
	// Count policies in this section
	policyCount := 0
	for _, pol := range cat.Policies {
		if pol.RawPolicy.Section == section || pol.RawPolicy.Section == policy.Both {
			policyCount++
		}
	}

	node := &CategoryNode{
		ID:          cat.UniqueID,
		Name:        cat.DisplayName,
		Description: cat.DisplayExplanation,
		Children:    []*CategoryNode{},
		PolicyCount: policyCount,
	}

	// Add children that have policies in this section
	for _, child := range cat.Children {
		if hasPoliciesInSection(child, section) {
			node.Children = append(node.Children, buildCategoryTreeForSection(child, section))
		}
	}
	sortCategoryNodes(node.Children)
	return node
}

// buildCategoryTree builds a category tree without section filtering (kept for backward compatibility if needed)
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
	sortCategoryNodes(node.Children)
	return node
}

func sortCategoryNodes(nodes []*CategoryNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})
}

type setPolicyRequest struct {
	PolicyID string                 `json:"policyId"`
	State    string                 `json:"state"`
	Section  string                 `json:"section,omitempty"`
	Options  map[string]interface{} `json:"options"`
}

func resolveSection(requested string, defaultSection policy.AdmxPolicySection) (policy.AdmxPolicySection, error) {
	if requested == "" {
		if defaultSection == policy.Both {
			return policy.Machine, nil
		}
		return defaultSection, nil
	}

	switch strings.ToLower(requested) {
	case "machine":
		return policy.Machine, nil
	case "user":
		return policy.User, nil
	default:
		return policy.Both, fmt.Errorf("invalid section: machine or user")
	}
}

func resolvePolicyState(state string) (policy.PolicyState, error) {
	switch strings.ToLower(strings.ReplaceAll(state, " ", "")) {
	case "enabled":
		return policy.PolicyStateEnabled, nil
	case "disabled":
		return policy.PolicyStateDisabled, nil
	case "notconfigured":
		return policy.PolicyStateNotConfigured, nil
	default:
		return policy.PolicyStateNotConfigured, fmt.Errorf("invalid state: enabled, disabled or notconfigured")
	}
}
