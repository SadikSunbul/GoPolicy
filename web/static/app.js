// Global variables
let currentCategory = null;
let currentPolicy = null;
let categories = [];

// On page load
document.addEventListener('DOMContentLoaded', () => {
    loadCategories();
    loadSources();
});

// Load categories
async function loadCategories() {
    try {
        const response = await fetch('/api/categories');
        categories = await response.json();
        renderCategories();
    } catch (error) {
        console.error('Failed to load categories:', error);
        showError('Failed to load categories');
    }
}

// Render categories
function renderCategories() {
    const tree = document.getElementById('categories-tree');
    tree.innerHTML = '';
    
    if (categories.length === 0) {
        tree.innerHTML = '<p class="loading">Loading categories...</p>';
        return;
    }
    
    categories.forEach(cat => {
        tree.appendChild(renderCategoryNode(cat));
    });
}

// Create category node
function renderCategoryNode(category) {
    const div = document.createElement('div');
    div.className = 'category-item';
    div.innerHTML = `
        <div onclick="selectCategory('${category.id}')">
            📁 ${category.name} (${category.policyCount})
        </div>
    `;
    
    if (category.children && category.children.length > 0) {
        const childrenDiv = document.createElement('div');
        childrenDiv.className = 'category-children';
        category.children.forEach(child => {
            childrenDiv.appendChild(renderCategoryNode(child));
        });
        div.appendChild(childrenDiv);
    }
    
    return div;
}

// Select category
async function selectCategory(categoryId) {
    // Mark active category
    document.querySelectorAll('.category-item').forEach(item => {
        item.classList.remove('active');
    });
    event.target.closest('.category-item').classList.add('active');
    
    currentCategory = categoryId;
    
    // Show category information
    const category = findCategory(categoryId, categories);
    if (category) {
        const infoPanel = document.getElementById('policy-info');
        infoPanel.innerHTML = `
            <h3>${category.name}</h3>
            <p>${category.description || 'No description'}</p>
            <p><strong>${category.policyCount} policies</strong> found.</p>
        `;
    }
    
    // Load policies
    await loadPolicies(categoryId);
}

// Find category (recursive)
function findCategory(id, cats) {
    for (const cat of cats) {
        if (cat.id === id) return cat;
        if (cat.children) {
            const found = findCategory(id, cat.children);
            if (found) return found;
        }
    }
    return null;
}

// Load policies
async function loadPolicies(categoryId) {
    try {
        const response = await fetch(`/api/policies?category=${encodeURIComponent(categoryId)}`);
        const policies = await response.json();
        renderPolicies(policies);
    } catch (error) {
        console.error('Failed to load policies:', error);
        showError('Failed to load policies');
    }
}

// Render policies
function renderPolicies(policies) {
    const list = document.getElementById('policies');
    list.innerHTML = '';
    
    if (policies.length === 0) {
        list.innerHTML = '<p>No policies found in this category.</p>';
        return;
    }
    
    policies.forEach(policy => {
        const div = document.createElement('div');
        div.className = 'policy-item';
        div.onclick = () => openPolicyEditor(policy.id);
        
        const stateClass = policy.state.toLowerCase().replace(' ', '-');
        
        div.innerHTML = `
            <h4>⚙️ ${policy.name}</h4>
            <p>${policy.description || 'No description'}</p>
            <span class="policy-state ${stateClass}">${policy.state}</span>
            <small style="color: #888; margin-left: 10px;">${policy.section}</small>
        `;
        
        list.appendChild(div);
    });
}

// Open policy editor
async function openPolicyEditor(policyId) {
    try {
        const response = await fetch(`/api/policy/${encodeURIComponent(policyId)}`);
        const policy = await response.json();
        currentPolicy = policy;
        
        const modal = document.getElementById('policy-edit-modal');
        const title = document.getElementById('modal-title');
        const body = document.getElementById('modal-body');
        
        title.textContent = policy.name;
        
        // Create modal content
        let html = `
            <p style="margin-bottom: 20px; color: #666;">${policy.description}</p>
            
            <div class="form-group">
                <label>Policy State:</label>
                <div class="radio-group">
                    <label>
                        <input type="radio" name="policy-state" value="NotConfigured" ${policy.state === 'Not Configured' ? 'checked' : ''}>
                        Not Configured
                    </label>
                    <label>
                        <input type="radio" name="policy-state" value="Enabled" ${policy.state === 'Enabled' ? 'checked' : ''}>
                        Enabled
                    </label>
                    <label>
                        <input type="radio" name="policy-state" value="Disabled" ${policy.state === 'Disabled' ? 'checked' : ''}>
                        Disabled
                    </label>
                </div>
            </div>
            
            <div id="policy-elements">
        `;
        
        // Add policy elements
        if (policy.elements && policy.elements.length > 0) {
            html += '<h4 style="margin-top: 20px; color: #667eea;">Settings:</h4>';
            policy.elements.forEach(elem => {
                html += renderPolicyElement(elem);
            });
        }
        
        html += `
            </div>
            <div style="margin-top: 20px; padding: 15px; background: #f0f0f0; border-radius: 5px;">
                <small><strong>Registry:</strong> ${policy.registryKey}</small>
            </div>
        `;
        
        body.innerHTML = html;
        modal.style.display = 'block';
        
        // Show/hide elements on state change
        document.querySelectorAll('input[name="policy-state"]').forEach(radio => {
            radio.addEventListener('change', () => {
                const elements = document.getElementById('policy-elements');
                elements.style.display = radio.value === 'Enabled' ? 'block' : 'none';
            });
        });
        
        // Set initial state
        const selectedState = document.querySelector('input[name="policy-state"]:checked').value;
        document.getElementById('policy-elements').style.display = selectedState === 'Enabled' ? 'block' : 'none';
        
    } catch (error) {
        console.error('Failed to load policy:', error);
        showError('Failed to load policy');
    }
}

// Render policy element
function renderPolicyElement(elem) {
    let html = '<div class="form-group">';
    
    html += `<label>${elem.label || elem.id}${elem.required ? ' *' : ''}</label>`;
    
    switch (elem.type) {
        case 'text':
            html += `<input type="text" id="elem-${elem.id}" placeholder="${elem.label}">`;
            break;
        
        case 'decimal':
            html += `<input type="number" id="elem-${elem.id}" placeholder="${elem.label}">`;
            break;
        
        case 'boolean':
            html += `<input type="checkbox" id="elem-${elem.id}">`;
            break;
        
        case 'enum':
            html += `<select id="elem-${elem.id}">`;
            if (elem.options) {
                elem.options.forEach((opt, idx) => {
                    html += `<option value="${idx}">${opt}</option>`;
                });
            }
            html += '</select>';
            break;
        
        case 'list':
            html += `<textarea id="elem-${elem.id}" rows="4" placeholder="One value per line..."></textarea>`;
            break;
        
        case 'multiText':
            html += `<textarea id="elem-${elem.id}" rows="4" placeholder="Multiple text..."></textarea>`;
            break;
        
        default:
            html += `<input type="text" id="elem-${elem.id}">`;
    }
    
    html += '</div>';
    return html;
}

// Apply policy
async function applyPolicy() {
    if (!currentPolicy) return;
    
    const state = document.querySelector('input[name="policy-state"]:checked').value;
    
    // Collect element values
    const options = {};
    if (currentPolicy.elements) {
        currentPolicy.elements.forEach(elem => {
            const input = document.getElementById(`elem-${elem.id}`);
            if (input) {
                switch (elem.type) {
                    case 'boolean':
                        options[elem.id] = input.checked;
                        break;
                    case 'decimal':
                        options[elem.id] = parseInt(input.value) || 0;
                        break;
                    case 'enum':
                        options[elem.id] = parseInt(input.value) || 0;
                        break;
                    case 'list':
                    case 'multiText':
                        options[elem.id] = input.value.split('\n').filter(l => l.trim());
                        break;
                    default:
                        options[elem.id] = input.value;
                }
            }
        });
    }
    
    try {
        const response = await fetch('/api/policy/set', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                policyId: currentPolicy.id,
                state: state,
                options: options
            })
        });
        
        if (response.ok) {
            showSuccess('Policy applied successfully!');
            closeModal();
            // Refresh policy list
            if (currentCategory) {
                await loadPolicies(currentCategory);
            }
        } else {
            showError('Failed to apply policy');
        }
    } catch (error) {
        console.error('Failed to apply policy:', error);
        showError('Failed to apply policy');
    }
}

// Close modal
function closeModal() {
    document.getElementById('policy-edit-modal').style.display = 'none';
    currentPolicy = null;
}

// Load sources
async function loadSources() {
    try {
        const response = await fetch('/api/sources');
        const sources = await response.json();
        console.log('Loaded sources:', sources);
    } catch (error) {
        console.error('Failed to load sources:', error);
    }
}

// Save policies
async function savePolicies() {
    try {
        const response = await fetch('/api/save', {
            method: 'POST'
        });
        
        if (response.ok) {
            const result = await response.json();
            showSuccess(result.message || 'Changes saved!');
        } else {
            showError('Save failed');
        }
    } catch (error) {
        console.error('Save error:', error);
        showError('Save error');
    }
}

// Notifications
function showSuccess(message) {
    alert('✅ ' + message);
}

function showError(message) {
    alert('❌ ' + message);
}

// Close modal when clicking outside
window.onclick = function(event) {
    const modal = document.getElementById('policy-edit-modal');
    if (event.target === modal) {
        closeModal();
    }
}

