// Global variables
let currentCategory = null;
let currentPolicy = null;
let categoriesData = { user: [], computer: [] };
let searchDebounceTimer = null;
let currentSearchResults = null;

// On page load
document.addEventListener('DOMContentLoaded', () => {
    loadCategories();
});

// Load categories
async function loadCategories() {
    try {
        const response = await fetch('/api/categories');
        const data = await response.json();
        categoriesData = {
            user: data.user || [],
            computer: data.computer || []
        };
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
    
    // User Configuration Section
    const userSection = document.createElement('div');
    userSection.className = 'category-section';
    const userHeader = document.createElement('div');
    userHeader.className = 'category-section-header';
    userHeader.innerHTML = 'User Configuration';
    userHeader.onclick = () => toggleSection(userSection);
    userSection.appendChild(userHeader);
    
    const userContent = document.createElement('div');
    userContent.className = 'category-section-content';
    userContent.style.display = 'block'; // Start expanded
    if (categoriesData.user.length === 0) {
        userContent.innerHTML = '<p class="loading">Loading user categories...</p>';
    } else {
        categoriesData.user.forEach(cat => {
            userContent.appendChild(renderCategoryNode(cat));
        });
    }
    userSection.appendChild(userContent);
    userSection.classList.add('expanded'); // Mark as expanded
    userHeader.innerHTML = '▼ User Configuration'; // Add arrow for expanded state
    tree.appendChild(userSection);
    
    // Computer Configuration Section
    const computerSection = document.createElement('div');
    computerSection.className = 'category-section';
    const computerHeader = document.createElement('div');
    computerHeader.className = 'category-section-header';
    computerHeader.innerHTML = '▼ Computer Configuration';
    computerHeader.onclick = () => toggleSection(computerSection);
    computerSection.appendChild(computerHeader);
    
    const computerContent = document.createElement('div');
    computerContent.className = 'category-section-content';
    computerContent.style.display = 'block'; // Start expanded
    if (categoriesData.computer.length === 0) {
        computerContent.innerHTML = '<p class="loading">Loading computer categories...</p>';
    } else {
        categoriesData.computer.forEach(cat => {
            computerContent.appendChild(renderCategoryNode(cat));
        });
    }
    computerSection.appendChild(computerContent);
    computerSection.classList.add('expanded'); // Mark as expanded
    tree.appendChild(computerSection);
}

// Toggle section expand/collapse
function toggleSection(section) {
    const content = section.querySelector('.category-section-content');
    const header = section.querySelector('.category-section-header');
    const isExpanded = content.style.display !== 'none';
    
    content.style.display = isExpanded ? 'none' : 'block';
    section.classList.toggle('expanded', !isExpanded);
    
    // Update arrow icon - preserve emoji and text
    let text = header.innerHTML;
    if (text.includes('▼')) {
        text = text.replace('▼', '▶');
    } else if (text.includes('▶')) {
        text = text.replace('▶', '▼');
    } else {
        // First time, add arrow at the beginning
        text = (isExpanded ? '▶' : '▼') + ' ' + text;
    }
    header.innerHTML = text;
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
            ${category.name} <span style="color: var(--text-light); font-size: 0.8125rem;">(${category.policyCount})</span>
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
    const category = findCategoryInAll(categoryId);
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

// Find category (recursive) - searches in both user and computer categories
function findCategory(id, cats) {
    if (!cats) return null;
    for (const cat of cats) {
        if (cat.id === id) return cat;
        if (cat.children) {
            const found = findCategory(id, cat.children);
            if (found) return found;
        }
    }
    return null;
}

// Find category in all sections
function findCategoryInAll(id) {
    let found = findCategory(id, categoriesData.user);
    if (found) return found;
    found = findCategory(id, categoriesData.computer);
    return found;
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
        div.dataset.policyId = policy.id;
        div.onclick = () => {
            // Remove active class from all policy items
            document.querySelectorAll('.policy-item').forEach(item => {
                item.classList.remove('active');
            });
            // Add active class to clicked item
            div.classList.add('active');
            openPolicyEditor(policy.id);
        };
        
        const stateClass = policy.state.toLowerCase().replace(' ', '-');
        
        div.innerHTML = `
            <h4>${policy.name}</h4>
            <p>${policy.description || 'No description'}</p>
            <div style="display: flex; align-items: center; gap: 12px; margin-top: 8px;">
                <span class="policy-state ${stateClass}">${policy.state}</span>
                <small style="color: var(--text-light); font-size: 0.8125rem;">${policy.section}</small>
            </div>
        `;
        
        list.appendChild(div);
    });
}

// Open policy editor in right panel
async function openPolicyEditor(policyId) {
    try {
        const response = await fetch(`/api/policy/${encodeURIComponent(policyId)}`);
        const policy = await response.json();
        currentPolicy = policy;
        resetApplyButton();
        
        const panel = document.getElementById('policy-detail-panel');
        const title = document.getElementById('policy-detail-title');
        const body = document.getElementById('policy-detail-body');
        const mainLayout = document.querySelector('.main-layout');
        
        // Show panel and update layout
        panel.classList.remove('hidden');
        if (mainLayout) {
            mainLayout.classList.add('with-panel');
        }
        
        title.textContent = policy.name;
        
        // Determine state badge color and text
        const currentState = policy.state || 'Not Configured';
        let stateBadgeClass = 'not-configured';
        if (currentState === 'Enabled') stateBadgeClass = 'enabled';
        else if (currentState === 'Disabled') stateBadgeClass = 'disabled';
        
        // Create panel content
        let html = `
            <p class="policy-description">${policy.description || 'No description available'}</p>
            
            <div class="form-group">
                <label>
                    Policy State: 
                    <span class="policy-state ${stateBadgeClass}" style="margin-left: 10px;">
                        Current: ${currentState}
                    </span>
                </label>
                <div class="radio-group">
                    <label>
                        <input type="radio" name="policy-state" value="NotConfigured" ${currentState === 'Not Configured' ? 'checked' : ''}>
                        Not Configured
                    </label>
                    <label>
                        <input type="radio" name="policy-state" value="Enabled" ${currentState === 'Enabled' ? 'checked' : ''}>
                        Enabled
                    </label>
                    <label>
                        <input type="radio" name="policy-state" value="Disabled" ${currentState === 'Disabled' ? 'checked' : ''}>
                        Disabled
                    </label>
                </div>
            </div>
            
            <div id="policy-elements">
        `;
        
        // Add policy elements
        if (policy.elements && policy.elements.length > 0) {
            html += '<h4 style="margin-top: 24px; color: var(--text-primary); font-size: 1rem; font-weight: 600; margin-bottom: 16px;">Settings:</h4>';
            policy.elements.forEach(elem => {
                html += renderPolicyElement(elem);
            });
        }
        
        html += `
            </div>
            <div class="registry-info">
                <small><strong>Registry:</strong> ${policy.registryKey}</small>
            </div>
            <div class="panel-actions">
                <button id="apply-policy-button" onclick="applyPolicy()">Apply</button>
                <button onclick="closePolicyPanel()">Cancel</button>
            </div>
        `;
        
        body.innerHTML = html;
        
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
                if (elements) {
                    elements.style.display = radio.value === 'Enabled' ? 'block' : 'none';
                }
            });
        });
        
        // Set initial state
        const selectedState = document.querySelector('input[name="policy-state"]:checked');
        if (selectedState) {
            const elements = document.getElementById('policy-elements');
            if (elements) {
                elements.style.display = selectedState.value === 'Enabled' ? 'block' : 'none';
            }
        }
        
    } catch (error) {
        console.error('Failed to load policy:', error);
        showError('Failed to load policy');
    }
}

// Close policy panel
function closePolicyPanel() {
    const panel = document.getElementById('policy-detail-panel');
    const mainLayout = document.querySelector('.main-layout');
    
    panel.classList.add('hidden');
    currentPolicy = null;
    
    // Remove active class from all policy items
    document.querySelectorAll('.policy-item').forEach(item => {
        item.classList.remove('active');
    });
    
    // Update layout to remove panel
    if (mainLayout) {
        mainLayout.classList.remove('with-panel');
    }
    
    resetApplyButton();
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
            let hasValue = elem.defaultValue !== undefined && elem.defaultValue !== null && elem.defaultValue !== '';
            if (elem.maxLength) {
                textAttrs += ` maxlength="${elem.maxLength}"`;
            }
            if (hasValue) {
                textAttrs += ` value="${escapeHtml(String(elem.defaultValue))}"`;
            }
            html += `<input type="text" class="form-control" id="elem-${elem.id}" data-element-id="${elem.id}" data-element-type="text" placeholder="${escapeHtml(elem.label || '')}"${textAttrs}>`;
            if (hasValue) {
                html += `<small style="display: block; color: var(--success-color); margin-top: 6px; font-weight: 500; font-size: 0.8125rem;">✓ Saved value loaded</small>`;
            }
            if (elem.maxLength) {
                html += `<small style="display: block; color: var(--text-light); margin-top: 6px; font-size: 0.8125rem;">Maximum length: ${elem.maxLength} characters</small>`;
            }
            break;
        
        case 'decimal':
            let numAttrs = '';
            let hasNumValue = elem.defaultValue !== undefined && elem.defaultValue !== null;
            if (elem.minValue !== undefined) {
                numAttrs += ` min="${elem.minValue}"`;
            }
            if (elem.maxValue !== undefined) {
                numAttrs += ` max="${elem.maxValue}"`;
            }
            if (hasNumValue) {
                numAttrs += ` value="${elem.defaultValue}"`;
            }
            if (elem.required) {
                numAttrs += ` required`;
            }
            html += `<input type="number" class="form-control" id="elem-${elem.id}" data-element-id="${elem.id}" data-element-type="decimal" placeholder="${escapeHtml(elem.label || '')}"${numAttrs}>`;
            if (hasNumValue) {
                html += `<small style="display: block; color: var(--success-color); margin-top: 6px; font-weight: 500; font-size: 0.8125rem;">✓ Saved value: ${elem.defaultValue}</small>`;
            }
            if (elem.minValue !== undefined || elem.maxValue !== undefined) {
                html += `<small style="display: block; color: var(--text-light); margin-top: 6px; font-size: 0.8125rem;">Value range: ${elem.minValue || 0} - ${elem.maxValue || 'unlimited'}</small>`;
            }
            break;
        
        case 'boolean':
            const boolChecked = elem.defaultValue === true;
            html += `<div style="display: flex; align-items: center; gap: 10px;">
                <input type="checkbox" id="elem-${elem.id}" data-element-id="${elem.id}" data-element-type="boolean" ${boolChecked ? 'checked' : ''}>
                <label for="elem-${elem.id}" style="cursor: pointer; margin: 0;">Enable</label>
            </div>`;
            if (elem.defaultValue !== undefined && elem.defaultValue !== null) {
                html += `<small style="display: block; color: var(--success-color); margin-top: 6px; font-weight: 500; font-size: 0.8125rem;">✓ Saved value: ${boolChecked ? 'On' : 'Off'}</small>`;
            }
            break;
        
        case 'enum':
            html += `<select class="form-control" id="elem-${elem.id}" data-element-id="${elem.id}" data-element-type="enum"${elem.required ? ' required' : ''}>`;
            if (!elem.required) {
                html += `<option value="">-- Select --</option>`;
            }
            let selectedOptionName = null;
            if (elem.options && elem.options.length > 0) {
                elem.options.forEach(opt => {
                    const selected = (elem.defaultValue !== undefined && elem.defaultValue === opt.index) ? ' selected' : '';
                    if (selected) selectedOptionName = opt.displayName;
                    html += `<option value="${opt.index}"${selected}>${escapeHtml(opt.displayName)}</option>`;
                });
            } else {
                html += `<option value="">No options available</option>`;
            }
            html += '</select>';
            if (selectedOptionName) {
                html += `<small style="display: block; color: var(--success-color); margin-top: 6px; font-weight: 500; font-size: 0.8125rem;">✓ Saved selection: ${escapeHtml(selectedOptionName)}</small>`;
            }
            break;
        
        case 'list':
            html += createListInputHTML(elem);
            break;
        
        case 'multiText':
            let multiTextValue = '';
            let hasMultiText = false;
            if (elem.defaultValue !== undefined && elem.defaultValue !== null) {
                if (Array.isArray(elem.defaultValue)) {
                    multiTextValue = elem.defaultValue.join('\n');
                    hasMultiText = elem.defaultValue.length > 0;
                } else {
                    multiTextValue = String(elem.defaultValue);
                    hasMultiText = multiTextValue.length > 0;
                }
            }
            html += `<textarea class="form-control" id="elem-${elem.id}" data-element-id="${elem.id}" data-element-type="multitext" rows="4" placeholder="Enter one value per line">${escapeHtml(multiTextValue)}</textarea>`;
            if (hasMultiText) {
                const lineCount = Array.isArray(elem.defaultValue) ? elem.defaultValue.length : multiTextValue.split('\n').filter(l => l.trim()).length;
                html += `<small style="display: block; color: var(--success-color); margin-top: 6px; font-weight: 500; font-size: 0.8125rem;">✓ ${lineCount} lines of saved values loaded</small>`;
            } else {
                html += `<small style="display: block; color: var(--text-light); margin-top: 6px; font-size: 0.8125rem;">Enter one value per line</small>`;
            }
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
        html += `<input type="text" class="form-control" data-list-key="true" placeholder="Key" style="flex: 1;">`;
        html += `<input type="text" class="form-control" data-list-value="true" placeholder="Value" style="flex: 1;">`;
    } else {
        html += `<input type="text" class="form-control" data-list-value="true" placeholder="Add value" style="flex: 1;">`;
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
            item.innerHTML = `<span><strong>${escapeHtml(key)}:</strong> ${escapeHtml(value)}</span><button onclick="this.parentElement.remove()" style="background: #dc3545; color: white; border: none; padding: 4px 8px; border-radius: 3px; cursor: pointer;">Delete</button>`;
            itemsContainer.appendChild(item);
        }
    } else if (Array.isArray(defaultValue)) {
        defaultValue.forEach(value => {
            const item = document.createElement('div');
            item.className = 'list-item';
            item.style.cssText = 'display: flex; justify-content: space-between; align-items: center; padding: 8px; background: #f5f5f5; border-radius: 4px; margin-bottom: 5px;';
            item.dataset.value = value;
            item.innerHTML = `<span>${escapeHtml(value)}</span><button onclick="this.parentElement.remove()" style="background: #dc3545; color: white; border: none; padding: 4px 8px; border-radius: 3px; cursor: pointer;">Delete</button>`;
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
            alert('Please enter both key and value');
            return;
        }
        
        const item = document.createElement('div');
        item.className = 'list-item';
        item.style.cssText = 'display: flex; justify-content: space-between; align-items: center; padding: 8px; background: #f5f5f5; border-radius: 4px; margin-bottom: 5px;';
        item.dataset.key = key;
        item.dataset.value = value;
        item.innerHTML = `<span><strong>${escapeHtml(key)}:</strong> ${escapeHtml(value)}</span><button onclick="this.parentElement.remove()" style="background: #dc3545; color: white; border: none; padding: 4px 8px; border-radius: 3px; cursor: pointer;">Delete</button>`;
        
        container.appendChild(item);
        keyInput.value = '';
        valueInput.value = '';
        keyInput.focus();
    } else {
        const valueInput = wrapper.querySelector('[data-list-value]');
        const value = valueInput.value.trim();
        
        if (!value) {
            alert('Please enter a value');
            return;
        }
        
        const item = document.createElement('div');
        item.className = 'list-item';
        item.style.cssText = 'display: flex; justify-content: space-between; align-items: center; padding: 8px; background: #f5f5f5; border-radius: 4px; margin-bottom: 5px;';
        item.dataset.value = value;
        item.innerHTML = `<span>${escapeHtml(value)}</span><button onclick="this.parentElement.remove()" style="background: #dc3545; color: white; border: none; padding: 4px 8px; border-radius: 3px; cursor: pointer;">Delete</button>`;
        
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
    setApplyButtonLoading(true);
    
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
            closePolicyPanel();
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
    } finally {
        setApplyButtonLoading(false);
    }
}

// Close modal (kept for compatibility, redirects to closePolicyPanel)
function closeModal() {
    closePolicyPanel();
}

// Refresh Explorer
async function refreshExplorer() {
    try {
        const response = await fetch('/api/refresh-explorer', {
            method: 'POST'
        });
        
        if (response.ok) {
            const result = await response.json();
            showSuccess(result.message || 'Windows Explorer restarted successfully');
        } else {
            const errorData = await response.json().catch(() => ({}));
            showError(errorData.error || errorData.message || 'Failed to restart Explorer');
        }
    } catch (error) {
        console.error('Refresh Explorer error:', error);
        showError('Failed to restart Explorer: ' + error.message);
    }
}

function setApplyButtonLoading(isLoading) {
    const button = document.getElementById('apply-policy-button');
    if (!button) return;
    if (isLoading) {
        button.disabled = true;
        button.dataset.originalText = button.dataset.originalText || button.textContent;
        button.textContent = 'Applying...';
        button.classList.add('loading');
    } else {
        button.disabled = false;
        button.textContent = button.dataset.originalText || 'Apply';
        button.classList.remove('loading');
    }
}

function resetApplyButton() {
    const button = document.getElementById('apply-policy-button');
    if (button) {
        button.disabled = false;
        button.textContent = button.dataset.originalText || 'Apply';
        button.classList.remove('loading');
    }
}

// Notifications
function showSuccess(message) {
    showNotification(message, 'success');
}

function showError(message) {
    showNotification(message, 'error');
}

function showNotification(message, type) {
    const container = document.getElementById('notification-container');
    if (!container) {
        alert(message);
        return;
    }

    const notification = document.createElement('div');
    notification.className = `notification ${type}`;
    notification.innerHTML = `<span>${message}</span>`;

    container.appendChild(notification);

    setTimeout(() => {
        notification.classList.add('hide');
    }, 3500);

    setTimeout(() => {
        notification.remove();
    }, 4000);
}

// Modal click handler removed - using side panel now

// ======== SEARCH FUNCTIONALITY ========

// Handle search key press with debouncing
function handleSearchKeyPress(event) {
    const query = event.target.value.trim();
    
    // Clear previous timer
    if (searchDebounceTimer) {
        clearTimeout(searchDebounceTimer);
    }
    
    // If empty, clear search results
    if (query === '') {
        clearSearchResults();
        return;
    }
    
    // Get selected section
    const sectionRadio = document.querySelector('input[name="search-section"]:checked');
    const section = sectionRadio ? sectionRadio.value : 'both';
    
    // Debounce: wait 500ms after user stops typing
    searchDebounceTimer = setTimeout(() => {
        performSearch(query, section);
    }, 500);
}

// Perform search
async function performSearch(query, section) {
    if (!query) return;
    
    try {
        const response = await fetch(`/api/search?q=${encodeURIComponent(query)}&section=${encodeURIComponent(section)}`);
        if (!response.ok) {
            throw new Error('Search failed');
        }
        
        const data = await response.json();
        currentSearchResults = data;
        
        // Display results based on section
        displaySearchResults(data, section);
        
    } catch (error) {
        console.error('Search error:', error);
        showError('Search failed');
    }
}

// Display search results
function displaySearchResults(data, section) {
    const policiesList = document.getElementById('policies');
    const infoPanel = document.getElementById('policy-info');
    
    // Determine which results to show
    let results = [];
    let sectionName = '';
    
    if (section === 'user') {
        results = data.user || [];
        sectionName = 'User';
    } else if (section === 'computer') {
        results = data.computer || [];
        sectionName = 'Computer';
    } else if (section === 'both') {
        // Combine both user and computer results
        const userResults = data.user || [];
        const computerResults = data.computer || [];
        results = [...userResults, ...computerResults];
        sectionName = 'All';
    }
    
    // Update info panel
    infoPanel.innerHTML = `
        <h3>Search Results</h3>
        <p><strong>${results.length}</strong> ${sectionName.toLowerCase()} polic${results.length === 1 ? 'y' : 'ies'} found.</p>
        <p style="color: var(--text-secondary); font-size: 0.875rem; margin-top: 4px;">Search: "${escapeHtml(data.query)}"</p>
    `;
    
    // Clear and show results
    policiesList.innerHTML = '';
    
    if (results.length === 0) {
        policiesList.innerHTML = `<p style="padding: 20px; text-align: center; color: #666;">
            No results found.
        </p>`;
        return;
    }
    
    // Render each result
    results.forEach(policy => {
        const div = document.createElement('div');
        div.className = 'policy-item search-result';
        div.dataset.policyId = policy.id;
        div.onclick = () => {
            // Remove active class from all policy items
            document.querySelectorAll('.policy-item').forEach(item => {
                item.classList.remove('active');
            });
            // Add active class to clicked item
            div.classList.add('active');
            openPolicyEditor(policy.id);
        };
        
        const stateClass = policy.state.toLowerCase().replace(' ', '-');
        
        div.innerHTML = `
            <h4>${escapeHtml(policy.name)}</h4>
            <p>${escapeHtml(policy.description || 'No description')}</p>
            <div style="display: flex; justify-content: space-between; align-items: center; margin-top: 8px; flex-wrap: wrap; gap: 8px;">
                <div style="display: flex; align-items: center; gap: 12px;">
                    <span class="policy-state ${stateClass}">${policy.state}</span>
                    <small style="color: var(--text-light); font-size: 0.8125rem;">${policy.section}</small>
                </div>
                <small style="color: var(--text-secondary); font-size: 0.8125rem;">
                    ${escapeHtml(policy.categoryName || 'No category')}
                </small>
            </div>
        `;
        
        policiesList.appendChild(div);
    });
}

// Clear search results
function clearSearchResults() {
    currentSearchResults = null;
    
    // Clear search input
    const searchInput = document.getElementById('search-input');
    if (searchInput) searchInput.value = '';
    
    // Clear policies list
    const policiesList = document.getElementById('policies');
    const infoPanel = document.getElementById('policy-info');
    
    policiesList.innerHTML = '<p style="color: var(--text-secondary); padding: 20px; text-align: center;">Select a category or search for policies.</p>';
    infoPanel.innerHTML = `
        <h3>Info Panel</h3>
        <p>Select a category or policy.</p>
    `;
    
    // If a category was selected, reload its policies
    if (currentCategory) {
        loadPolicies(currentCategory);
    }
}

