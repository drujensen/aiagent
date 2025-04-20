function showTempMessage(form) {
    const messageText = form.elements["message"].value.trim();
    if (!messageText) return;

    const sessionId = 'temp-session-' + Date.now();
    const messageHtml = `
        <div id="${sessionId}" class="message-session">
            <div class="message user-message">
                <div class="message-content">${marked.parse(messageText)}</div>
            </div>
            <div id="thinking-message" class="message agent-message">
                <div class="message-content thinking-content">
                    Thinking<span class="spinner"></span>
                </div>
            </div>
        </div>
    `;
    document.getElementById('next-message-session').innerHTML = messageHtml;
}

function scrollToResponse() {
    const messageHistory = document.getElementById('message-history');
    if (messageHistory) {
        messageHistory.scroll({
            top: messageHistory.scrollHeight,
            behavior: 'smooth'
        });
    }
}

function handleResponseError(form, event) {
    const thinkingMessage = document.getElementById('thinking-message');
    if (thinkingMessage) {
        thinkingMessage.innerHTML = `
            <div class="message-content error-content">
                Error: ${event.detail.xhr.responseText || 'Failed to get response'}
            </div>
        `;
    }
    form.reset();
    toggleCancelButton(false);
}

function resetForm(button) {
    const form = document.getElementById('message-form');
    form.reset();
}

function toggleCancelButton(isRequesting) {
    const sendButton = document.querySelector('#send-button');
    const sendText = document.querySelector('.send-text');
    const cancelText = document.querySelector('.cancel-text');

    if (isRequesting) {
        sendButton.classList.add('cancel-button');
        sendText.style.display = 'none';
        cancelText.style.display = 'inline';
    } else {
        sendButton.classList.remove('cancel-button');
        sendText.style.display = 'inline';
        cancelText.style.display = 'none';
    }
}

document.addEventListener('DOMContentLoaded', () => {
    initCopyButtons();
    scrollToResponse();
});
