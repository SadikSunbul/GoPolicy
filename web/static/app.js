// Global variables
let currentCategory = null;
let currentPolicy = null;
let categoriesData = { user: [], computer: [] };
let searchDebounceTimer = null;
let currentSearchResults = null;

// On page load
document.addEventListener('DOMContentLoaded', () => {
    loadCategories();
    loadSources();
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
    userHeader.innerHTML = '👤 Kullanıcı Yapılandırması';
    userHeader.onclick = () => toggleSection(userSection);
    userSection.appendChild(userHeader);
    
    const userContent = document.createElement('div');
    userContent.className = 'category-section-content';
    userContent.style.display = 'block'; // Start expanded
    if (categoriesData.user.length === 0) {
        userContent.innerHTML = '<p class="loading">Kullanıcı kategorileri yükleniyor...</p>';
    } else {
        categoriesData.user.forEach(cat => {
            userContent.appendChild(renderCategoryNode(cat));
        });
    }
    userSection.appendChild(userContent);
    userSection.classList.add('expanded'); // Mark as expanded
    userHeader.innerHTML = '▼ 👤 Kullanıcı Yapılandırması'; // Add arrow for expanded state
    tree.appendChild(userSection);
    
    // Computer Configuration Section
    const computerSection = document.createElement('div');
    computerSection.className = 'category-section';
    const computerHeader = document.createElement('div');
    computerHeader.className = 'category-section-header';
    computerHeader.innerHTML = '▼ 🖥️ Bilgisayar Yapılandırması';
    computerHeader.onclick = () => toggleSection(computerSection);
    computerSection.appendChild(computerHeader);
    
    const computerContent = document.createElement('div');
    computerContent.className = 'category-section-content';
    computerContent.style.display = 'block'; // Start expanded
    if (categoriesData.computer.length === 0) {
        computerContent.innerHTML = '<p class="loading">Bilgisayar kategorileri yükleniyor...</p>';
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
        resetApplyButton();
        
        console.log('Policy loaded:', policy); // Debug: API'dan gelen veriyi kontrol et
        
        const modal = document.getElementById('policy-edit-modal');
        const title = document.getElementById('modal-title');
        const body = document.getElementById('modal-body');
        
        title.textContent = policy.name;
        
        // Determine state badge color and text
        const currentState = policy.state || 'Not Configured';
        let stateBadgeClass = 'not-configured';
        if (currentState === 'Enabled') stateBadgeClass = 'enabled';
        else if (currentState === 'Disabled') stateBadgeClass = 'disabled';
        
        // Create modal content
        let html = `
            <p style="margin-bottom: 20px; color: #666;">${policy.description}</p>
            
            <div class="form-group">
                <label>
                    Policy State: 
                    <span class="policy-state ${stateBadgeClass}" style="margin-left: 10px;">
                        Şu an: ${currentState}
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
            let hasValue = elem.defaultValue !== undefined && elem.defaultValue !== null && elem.defaultValue !== '';
            if (elem.maxLength) {
                textAttrs += ` maxlength="${elem.maxLength}"`;
            }
            if (hasValue) {
                textAttrs += ` value="${escapeHtml(String(elem.defaultValue))}"`;
            }
            html += `<input type="text" class="form-control" id="elem-${elem.id}" data-element-id="${elem.id}" data-element-type="text" placeholder="${escapeHtml(elem.label || '')}"${textAttrs}>`;
            if (hasValue) {
                html += `<small style="display: block; color: #4caf50; margin-top: 3px; font-weight: 600;">✓ Kayıtlı değer yüklendi</small>`;
            }
            if (elem.maxLength) {
                html += `<small style="display: block; color: #666; margin-top: 3px;">Maksimum uzunluk: ${elem.maxLength} karakter</small>`;
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
                html += `<small style="display: block; color: #4caf50; margin-top: 3px; font-weight: 600;">✓ Kayıtlı değer: ${elem.defaultValue}</small>`;
            }
            if (elem.minValue !== undefined || elem.maxValue !== undefined) {
                html += `<small style="display: block; color: #666; margin-top: 3px;">Değer aralığı: ${elem.minValue || 0} - ${elem.maxValue || 'sınırsız'}</small>`;
            }
            break;
        
        case 'boolean':
            const boolChecked = elem.defaultValue === true;
            html += `<div style="display: flex; align-items: center; gap: 10px;">
                <input type="checkbox" id="elem-${elem.id}" data-element-id="${elem.id}" data-element-type="boolean" ${boolChecked ? 'checked' : ''}>
                <label for="elem-${elem.id}" style="cursor: pointer; margin: 0;">Etkinleştir</label>
            </div>`;
            if (elem.defaultValue !== undefined && elem.defaultValue !== null) {
                html += `<small style="display: block; color: #4caf50; margin-top: 3px; font-weight: 600;">✓ Kayıtlı değer: ${boolChecked ? 'Açık' : 'Kapalı'}</small>`;
            }
            break;
        
        case 'enum':
            html += `<select class="form-control" id="elem-${elem.id}" data-element-id="${elem.id}" data-element-type="enum"${elem.required ? ' required' : ''}>`;
            if (!elem.required) {
                html += `<option value="">-- Seçin --</option>`;
            }
            let selectedOptionName = null;
            if (elem.options && elem.options.length > 0) {
                elem.options.forEach(opt => {
                    const selected = (elem.defaultValue !== undefined && elem.defaultValue === opt.index) ? ' selected' : '';
                    if (selected) selectedOptionName = opt.displayName;
                    html += `<option value="${opt.index}"${selected}>${escapeHtml(opt.displayName)}</option>`;
                });
            } else {
                html += `<option value="">Seçenek bulunamadı</option>`;
            }
            html += '</select>';
            if (selectedOptionName) {
                html += `<small style="display: block; color: #4caf50; margin-top: 3px; font-weight: 600;">✓ Kayıtlı seçim: ${escapeHtml(selectedOptionName)}</small>`;
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
            html += `<textarea class="form-control" id="elem-${elem.id}" data-element-id="${elem.id}" data-element-type="multitext" rows="4" placeholder="Her satıra bir değer girin">${escapeHtml(multiTextValue)}</textarea>`;
            if (hasMultiText) {
                const lineCount = Array.isArray(elem.defaultValue) ? elem.defaultValue.length : multiTextValue.split('\n').filter(l => l.trim()).length;
                html += `<small style="display: block; color: #4caf50; margin-top: 3px; font-weight: 600;">✓ ${lineCount} satır kayıtlı değer yüklendi</small>`;
            } else {
                html += `<small style="display: block; color: #666; margin-top: 3px;">Her satıra bir değer girin</small>`;
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
    } finally {
        setApplyButtonLoading(false);
    }
}

// Close modal
function closeModal() {
    document.getElementById('policy-edit-modal').style.display = 'none';
    currentPolicy = null;
    resetApplyButton();
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
    notification.innerHTML = `${type === 'error' ? '❌' : '✅'} <span>${message}</span>`;

    container.appendChild(notification);

    setTimeout(() => {
        notification.classList.add('hide');
    }, 3500);

    setTimeout(() => {
        notification.remove();
    }, 4000);
}

// Close modal when clicking outside
window.onclick = function(event) {
    const modal = document.getElementById('policy-edit-modal');
    if (event.target === modal) {
        closeModal();
    }
}

// ======== SEARCH FUNCTIONALITY ========

// Handle search key press with debouncing
function handleSearchKeyPress(event, section) {
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
    
    // Debounce: wait 500ms after user stops typing
    searchDebounceTimer = setTimeout(() => {
        performSearch(query, section);
    }, 500);
}

// Perform search
async function performSearch(query, section) {
    if (!query) return;
    
    try {
        const response = await fetch(`/api/search?q=${encodeURIComponent(query)}`);
        if (!response.ok) {
            throw new Error('Search failed');
        }
        
        const data = await response.json();
        currentSearchResults = data;
        
        // Display results based on section
        displaySearchResults(data, section);
        
    } catch (error) {
        console.error('Search error:', error);
        showError('Arama başarısız oldu');
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
        sectionName = 'Kullanıcı';
    } else if (section === 'computer') {
        results = data.computer || [];
        sectionName = 'Bilgisayar';
    }
    
    // Update info panel
    infoPanel.innerHTML = `
        <h3>🔍 Arama Sonuçları</h3>
        <p><strong>${results.length}</strong> ${sectionName.toLowerCase()} politikası bulundu.</p>
        <p style="color: #666; font-size: 14px;">Arama: "${escapeHtml(data.query)}"</p>
    `;
    
    // Clear and show results
    policiesList.innerHTML = '';
    
    if (results.length === 0) {
        policiesList.innerHTML = `<p style="padding: 20px; text-align: center; color: #666;">
            Bu bölümde sonuç bulunamadı.
        </p>`;
        return;
    }
    
    // Render each result
    results.forEach(policy => {
        const div = document.createElement('div');
        div.className = 'policy-item search-result';
        div.onclick = () => openPolicyEditor(policy.id);
        
        const stateClass = policy.state.toLowerCase().replace(' ', '-');
        
        div.innerHTML = `
            <h4>⚙️ ${escapeHtml(policy.name)}</h4>
            <p>${escapeHtml(policy.description || 'Açıklama yok')}</p>
            <div style="display: flex; justify-content: space-between; align-items: center; margin-top: 8px;">
                <div>
                    <span class="policy-state ${stateClass}">${policy.state}</span>
                    <small style="color: #888; margin-left: 10px;">${policy.section}</small>
                </div>
                <small style="color: #667eea; font-size: 11px;">
                    📁 ${escapeHtml(policy.categoryName || 'Kategori yok')}
                </small>
            </div>
        `;
        
        policiesList.appendChild(div);
    });
}

// Clear search results
function clearSearchResults() {
    currentSearchResults = null;
    
    // Clear search inputs
    const userInput = document.getElementById('user-search-input');
    const computerInput = document.getElementById('computer-search-input');
    
    if (userInput) userInput.value = '';
    if (computerInput) computerInput.value = '';
    
    // Clear policies list
    const policiesList = document.getElementById('policies');
    const infoPanel = document.getElementById('policy-info');
    
    policiesList.innerHTML = '<p>Bir kategori seçin veya arama yapın.</p>';
    infoPanel.innerHTML = `
        <h3>Info Panel</h3>
        <p>Select a category or policy.</p>
    `;
    
    // If a category was selected, reload its policies
    if (currentCategory) {
        loadPolicies(currentCategory);
    }
}

