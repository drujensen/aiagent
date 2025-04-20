document.addEventListener('htmx:afterRequest', function(event) {
    if (event.detail.target.id === 'model') {
        const providerSelect = document.getElementById('provider');
        const modelSelect = document.getElementById('model');
        const endpointInput = document.getElementById('endpoint');
        const apiKeyLabel = document.getElementById('api-key-label');
        const customModelContainer = document.getElementById('custom-model-container');
        const customModelInput = document.getElementById('custom_model_name');
        const providerErrorDiv = document.getElementById('provider-error');

        // Update endpoint and API key label from response headers
        const providerBaseUrl = event.detail.xhr.getResponseHeader('X-Provider-URL');
        const apiKeyName = event.detail.xhr.getResponseHeader('X-Provider-Key-Name');
        const providerType = event.detail.xhr.getResponseHeader('X-Provider-Type');

        if (apiKeyName) {
            apiKeyLabel.textContent = apiKeyName;
        }

        if (providerBaseUrl && (!endpointInput.value || endpointInput.dataset.autoSet === 'true')) {
            endpointInput.value = providerBaseUrl;
            endpointInput.dataset.autoSet = 'true';
        }

        // Show custom model input for generic providers or if no models
        customModelContainer.style.display = (providerType === 'generic' || modelSelect.options.length <= 1) ? 'block' : 'none';

        // Sync custom model input with selected model
        if (modelSelect.value && modelSelect.value !== 'custom') {
            customModelInput.value = modelSelect.value;
        }
    }
});

document.addEventListener('htmx:responseError', function(event) {
    if (event.detail.target.id === 'model') {
        const providerErrorDiv = document.getElementById('provider-error');
        providerErrorDiv.style.display = 'block';
        providerErrorDiv.textContent = 'Error loading models: ' + event.detail.xhr.responseText;
    }
});

document.addEventListener('DOMContentLoaded', function() {
    const modelSelect = document.getElementById('model');
    const customModelInput = document.getElementById('custom_model_name');
    const form = document.querySelector('form');

    // Handle model selection
    modelSelect.addEventListener('change', function() {
        if (this.value === 'custom') {
            customModelInput.focus();
        } else if (this.value) {
            customModelInput.value = this.value;
        }
    });

    // Handle custom model input
    customModelInput.addEventListener('input', function() {
        if (this.value.trim()) {
            let option = Array.from(modelSelect.options).find(opt => opt.value === this.value.trim());
            if (!option) {
                option = document.createElement('option');
                option.value = this.value.trim();
                option.textContent = 'Custom: ' + this.value.trim();
                modelSelect.appendChild(option);
            }
            modelSelect.value = this.value.trim();
        }
    });

    // Ensure custom model is set before form submission
    form.addEventListener('htmx:beforeRequest', function() {
        if (customModelInput.value.trim()) {
            let option = Array.from(modelSelect.options).find(opt => opt.value === customModelInput.value.trim());
            if (!option) {
                option = document.createElement('option');
                option.value = customModelInput.value.trim();
                option.textContent = 'Custom: ' + customModelInput.value.trim();
                modelSelect.appendChild(option);
            }
            modelSelect.value = customModelInput.value.trim();
        }
    });
});
