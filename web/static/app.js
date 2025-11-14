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
    
    const hasChildren = category.children && category.children.length > 0;
    const toggleIcon = hasChildren ? '▶' : '';
    
    const headerDiv = document.createElement('div');
    headerDiv.className = 'category-header';
    headerDiv.innerHTML = `
        <span class="category-toggle" ${hasChildren ? `onclick="toggleCategory(event, this)"` : ''}>
            ${toggleIcon}
        </span>
        <span class="category-name" onclick="selectCategory('${category.id}', event)">
            📁 ${category.name} (${category.policyCount})
        </span>
    `;
    div.appendChild(headerDiv);
    
    if (hasChildren) {
        const childrenDiv = document.createElement('div');
        childrenDiv.className = 'category-children';
        childrenDiv.style.display = 'none'; 
        category.children.forEach(child => {
            childrenDiv.appendChild(renderCategoryNode(child));
        });
        div.appendChild(childrenDiv);
    }
    
    return div;
}

function toggleCategory(event, element) {
    event.stopPropagation(); 
    
    const categoryItem = element.closest('.category-item');
    const childrenDiv = categoryItem.querySelector('.category-children');
    
    if (childrenDiv) {
        const isHidden = childrenDiv.style.display === 'none';
        childrenDiv.style.display = isHidden ? 'block' : 'none';
        element.textContent = isHidden ? '▼' : '▶';
        categoryItem.classList.toggle('expanded', isHidden);
    }
}

// Select category
async function selectCategory(categoryId, event) {
    if (event) {
        event.stopPropagation();
    }
    
    // Mark active category
    document.querySelectorAll('.category-item').forEach(item => {
        item.classList.remove('active');
    });
    
    const targetElement = event ? event.target.closest('.category-item') : null;
    if (targetElement) {
        targetElement.classList.add('active');
    }
    
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
        
        // Load existing list values after DOM is ready
        if (policy.elements && policy.elements.length > 0) {
            setTimeout(() => {
                policy.elements.forEach(elem => {
                    if (elem.type === 'list' && elem.defaultValue !== undefined && elem.defaultValue !== null) {
                        loadListValues(elem);
                    }
                });
            }, 100);
        }
        
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
    
    html += `<label>${escapeHtml(elem.label || elem.id)}${elem.required ? ' <span style="color: red;">*</span>' : ''}</label>`;
    
    if (elem.description) {
        html += `<small style="display: block; color: #666; margin-bottom: 5px;">${escapeHtml(elem.description)}</small>`;
    }
    
    switch (elem.type) {
        case 'text':
            let textAttrs = '';
            if (elem.maxLength) {
                textAttrs += ` maxlength="${elem.maxLength}"`;
            }
            if (elem.defaultValue !== undefined && elem.defaultValue !== null) {
                textAttrs += ` value="${escapeHtml(String(elem.defaultValue))}"`;
            }
            html += `<input type="text" class="form-control" id="elem-${elem.id}" data-element-id="${elem.id}" data-element-type="text" placeholder="${escapeHtml(elem.label || '')}"${textAttrs}>`;
            if (elem.maxLength) {
                html += `<small style="display: block; color: #666; margin-top: 3px;">Maksimum uzunluk: ${elem.maxLength} karakter</small>`;
            }
            break;
        
        case 'decimal':
            let numAttrs = '';
            if (elem.minValue !== undefined) {
                numAttrs += ` min="${elem.minValue}"`;
            }
            if (elem.maxValue !== undefined) {
                numAttrs += ` max="${elem.maxValue}"`;
            }
            if (elem.defaultValue !== undefined && elem.defaultValue !== null) {
                numAttrs += ` value="${elem.defaultValue}"`;
            }
            if (elem.required) {
                numAttrs += ` required`;
            }
            html += `<input type="number" class="form-control" id="elem-${elem.id}" data-element-id="${elem.id}" data-element-type="decimal" placeholder="${escapeHtml(elem.label || '')}"${numAttrs}>`;
            if (elem.minValue !== undefined || elem.maxValue !== undefined) {
                html += `<small style="display: block; color: #666; margin-top: 3px;">Değer aralığı: ${elem.minValue || 0} - ${elem.maxValue || 'sınırsız'}</small>`;
            }
            break;
        
        case 'boolean':
            html += `<div style="display: flex; align-items: center; gap: 10px;">
                <input type="checkbox" id="elem-${elem.id}" data-element-id="${elem.id}" data-element-type="boolean" ${elem.defaultValue ? 'checked' : ''}>
                <label for="elem-${elem.id}" style="cursor: pointer; margin: 0;">Etkinleştir</label>
            </div>`;
            break;
        
        case 'enum':
            html += `<select class="form-control" id="elem-${elem.id}" data-element-id="${elem.id}" data-element-type="enum"${elem.required ? ' required' : ''}>`;
            if (!elem.required) {
                html += `<option value="">-- Seçin --</option>`;
            }
            if (elem.options && elem.options.length > 0) {
                elem.options.forEach(opt => {
                    const selected = (elem.defaultValue !== undefined && elem.defaultValue === opt.index) ? ' selected' : '';
                    html += `<option value="${opt.index}"${selected}>${escapeHtml(opt.displayName)}</option>`;
                });
            } else {
                html += `<option value="">Seçenek bulunamadı</option>`;
            }
            html += '</select>';
            break;
        
        case 'list':
            html += createListInputHTML(elem);
            break;
        
        case 'multiText':
            let multiTextValue = '';
            if (elem.defaultValue !== undefined && elem.defaultValue !== null) {
                if (Array.isArray(elem.defaultValue)) {
                    multiTextValue = elem.defaultValue.join('\n');
                } else {
                    multiTextValue = String(elem.defaultValue);
                }
            }
            html += `<textarea class="form-control" id="elem-${elem.id}" data-element-id="${elem.id}" data-element-type="multitext" rows="4" placeholder="Her satıra bir değer girin">${escapeHtml(multiTextValue)}</textarea>`;
            html += `<small style="display: block; color: #666; margin-top: 3px;">Her satıra bir değer girin</small>`;
            break;
        
        default:
            html += `<input type="text" class="form-control" id="elem-${elem.id}" data-element-id="${elem.id}" data-element-type="text" placeholder="${escapeHtml(elem.label || '')}">`;
    }
    
    html += '</div>';
    return html;
}

// Create List Input HTML (key-value or simple list)
function createListInputHTML(elem) {
    const isKeyValue = elem.metadata && elem.metadata.userProvidesNames;
    const wrapperId = `list-wrapper-${elem.id}`;
    
    let html = `<div id="${wrapperId}" data-element-id="${elem.id}" data-element-type="list">`;
    html += `<div class="list-input" style="display: flex; gap: 10px; margin-bottom: 10px;">`;
    
    if (isKeyValue) {
        html += `<input type="text" class="form-control" data-list-key="true" placeholder="Anahtar" style="flex: 1;">`;
        html += `<input type="text" class="form-control" data-list-value="true" placeholder="Değer" style="flex: 1;">`;
    } else {
        html += `<input type="text" class="form-control" data-list-value="true" placeholder="Değer ekle" style="flex: 1;">`;
    }
    
    html += `<button type="button" class="btn btn-primary" onclick="addListItem('${wrapperId}', ${isKeyValue})" style="padding: 8px 15px;">+</button>`;
    html += `</div>`;
    html += `<div class="list-items" data-list-items="true" style="margin-top: 10px;"></div>`;
    html += `</div>`;
    return html;
}

// Load List Values
function loadListValues(elem) {
    const wrapperId = `list-wrapper-${elem.id}`;
    const wrapper = document.getElementById(wrapperId);
    if (!wrapper) return;
    
    const itemsContainer = wrapper.querySelector('[data-list-items]');
    if (!itemsContainer) return;
    
    const isKeyValue = elem.metadata && elem.metadata.userProvidesNames;
    const defaultValue = elem.defaultValue;
    
    if (isKeyValue && typeof defaultValue === 'object' && !Array.isArray(defaultValue)) {
        for (const [key, value] of Object.entries(defaultValue)) {
            const item = document.createElement('div');
            item.className = 'list-item';
            item.style.cssText = 'display: flex; justify-content: space-between; align-items: center; padding: 8px; background: #f5f5f5; border-radius: 4px; margin-bottom: 5px;';
            item.dataset.key = key;
            item.dataset.value = value;
            item.innerHTML = `<span><strong>${escapeHtml(key)}:</strong> ${escapeHtml(value)}</span><button onclick="this.parentElement.remove()" style="background: #dc3545; color: white; border: none; padding: 4px 8px; border-radius: 3px; cursor: pointer;">Sil</button>`;
            itemsContainer.appendChild(item);
        }
    } else if (Array.isArray(defaultValue)) {
        defaultValue.forEach(value => {
            const item = document.createElement('div');
            item.className = 'list-item';
            item.style.cssText = 'display: flex; justify-content: space-between; align-items: center; padding: 8px; background: #f5f5f5; border-radius: 4px; margin-bottom: 5px;';
            item.dataset.value = value;
            item.innerHTML = `<span>${escapeHtml(value)}</span><button onclick="this.parentElement.remove()" style="background: #dc3545; color: white; border: none; padding: 4px 8px; border-radius: 3px; cursor: pointer;">Sil</button>`;
            itemsContainer.appendChild(item);
        });
    }
}

// Add List Item
function addListItem(wrapperId, isKeyValue) {
    const wrapper = document.getElementById(wrapperId);
    if (!wrapper) return;
    
    const container = wrapper.querySelector('[data-list-items]');
    
    if (isKeyValue) {
        const keyInput = wrapper.querySelector('[data-list-key]');
        const valueInput = wrapper.querySelector('[data-list-value]');
        
        const key = keyInput.value.trim();
        const value = valueInput.value.trim();
        
        if (!key || !value) {
            alert('Lütfen hem anahtar hem değer girin');
            return;
        }
        
        const item = document.createElement('div');
        item.className = 'list-item';
        item.style.cssText = 'display: flex; justify-content: space-between; align-items: center; padding: 8px; background: #f5f5f5; border-radius: 4px; margin-bottom: 5px;';
        item.dataset.key = key;
        item.dataset.value = value;
        item.innerHTML = `<span><strong>${escapeHtml(key)}:</strong> ${escapeHtml(value)}</span><button onclick="this.parentElement.remove()" style="background: #dc3545; color: white; border: none; padding: 4px 8px; border-radius: 3px; cursor: pointer;">Sil</button>`;
        
        container.appendChild(item);
        keyInput.value = '';
        valueInput.value = '';
        keyInput.focus();
    } else {
        const valueInput = wrapper.querySelector('[data-list-value]');
        const value = valueInput.value.trim();
        
        if (!value) {
            alert('Lütfen bir değer girin');
            return;
        }
        
        const item = document.createElement('div');
        item.className = 'list-item';
        item.style.cssText = 'display: flex; justify-content: space-between; align-items: center; padding: 8px; background: #f5f5f5; border-radius: 4px; margin-bottom: 5px;';
        item.dataset.value = value;
        item.innerHTML = `<span>${escapeHtml(value)}</span><button onclick="this.parentElement.remove()" style="background: #dc3545; color: white; border: none; padding: 4px 8px; border-radius: 3px; cursor: pointer;">Sil</button>`;
        
        container.appendChild(item);
        valueInput.value = '';
        valueInput.focus();
    }
}

// Escape HTML
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Apply policy
async function applyPolicy() {
    if (!currentPolicy) return;
    
    const state = document.querySelector('input[name="policy-state"]:checked').value;
    
    // Collect element values
    const options = {};
    if (currentPolicy.elements && state === 'Enabled') {
        currentPolicy.elements.forEach(elem => {
            const elementId = elem.id;
            
            switch (elem.type) {
                case 'decimal':
                    const decimalInput = document.querySelector(`[data-element-id="${elementId}"]`);
                    if (decimalInput && decimalInput.value) {
                        const val = parseInt(decimalInput.value);
                        if (!isNaN(val)) {
                            options[elementId] = val;
                        }
                    }
                    break;
                    
                case 'text':
                    const textInput = document.querySelector(`[data-element-id="${elementId}"]`);
                    if (textInput && textInput.value) {
                        options[elementId] = textInput.value;
                    }
                    break;
                    
                case 'boolean':
                    const booleanInput = document.querySelector(`[data-element-id="${elementId}"]`);
                    if (booleanInput) {
                        options[elementId] = booleanInput.checked;
                    }
                    break;
                    
                case 'enum':
                    const enumSelect = document.querySelector(`[data-element-id="${elementId}"]`);
                    if (enumSelect && enumSelect.value !== '') {
                        const val = parseInt(enumSelect.value);
                        if (!isNaN(val)) {
                            options[elementId] = val;
                        }
                    }
                    break;
                    
                case 'list':
                    const listWrapper = document.querySelector(`[data-element-id="${elementId}"]`);
                    if (listWrapper) {
                        const items = listWrapper.querySelectorAll('.list-item');
                        const isKeyValue = elem.metadata && elem.metadata.userProvidesNames;
                        
                        if (isKeyValue) {
                            const map = {};
                            items.forEach(item => {
                                if (item.dataset.key && item.dataset.value) {
                                    map[item.dataset.key] = item.dataset.value;
                                }
                            });
                            if (Object.keys(map).length > 0) {
                                options[elementId] = map;
                            }
                        } else {
                            const list = [];
                            items.forEach(item => {
                                if (item.dataset.value) {
                                    list.push(item.dataset.value);
                                }
                            });
                            if (list.length > 0) {
                                options[elementId] = list;
                            }
                        }
                    }
                    break;
                    
                case 'multiText':
                    const multiTextInput = document.querySelector(`[data-element-id="${elementId}"]`);
                    if (multiTextInput && multiTextInput.value) {
                        const lines = multiTextInput.value.split('\n').map(l => l.trim()).filter(l => l);
                        if (lines.length > 0) {
                            options[elementId] = lines;
                        }
                    }
                    break;
            }
        });
    }
    
    // Determine section
    let section = '';
    if (currentPolicy.section) {
        const policySection = currentPolicy.section.toLowerCase();
        if (policySection === 'both') {
            // If both, check if user selected a specific section
            const sectionRadio = document.querySelector('input[name="targetSection"]:checked');
            if (sectionRadio) {
                section = sectionRadio.value;
            } else {
                section = 'machine'; // default
            }
        } else {
            section = policySection === 'user' ? 'user' : 'machine';
        }
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
                section: section,
                options: options
            })
        });
        
        const rawText = await response.text();
        if (response.ok) {
            let resultMessage = 'Policy applied successfully!';
            if (rawText) {
                try {
                    const result = JSON.parse(rawText);
                    if (result && (result.message || result.success)) {
                        resultMessage = result.message || resultMessage;
                    }
                } catch (parseErr) {
                    console.warn('Success response JSON parse error:', parseErr);
                }
            }
            showSuccess(resultMessage);
            closeModal();
            // Refresh policy list
            if (currentCategory) {
                await loadPolicies(currentCategory);
            }
        } else {
            let errorMessage = 'Failed to apply policy';
            if (rawText) {
                try {
                    const errorData = JSON.parse(rawText);
                    errorMessage = errorData.error || errorData.message || errorMessage;
                } catch {
                    errorMessage = rawText;
                }
            }
            showError(errorMessage);
        }
    } catch (error) {
        console.error('Failed to apply policy:', error);
        showError('Failed to apply policy: ' + error.message);
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

