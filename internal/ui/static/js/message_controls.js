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

// WebSocket connection for real-time updates
let ws = null;

function connectWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;

    ws = new WebSocket(wsUrl);

    ws.onopen = function(event) {
        console.log('WebSocket connected');
    };

    ws.onmessage = function(event) {
        const data = JSON.parse(event.data);
        if (data.type === 'message_history_update') {
            handleMessageHistoryUpdate(data);
        } else if (data.type === 'tool_call_update') {
            handleToolCallUpdate(data);
        }
    };

    ws.onclose = function(event) {
        console.log('WebSocket disconnected, reconnecting...');
        setTimeout(connectWebSocket, 1000);
    };

    ws.onerror = function(error) {
        console.error('WebSocket error:', error);
    };
}

function handleMessageHistoryUpdate(data) {
    const chatId = document.getElementById('messages-container').getAttribute('data-chat-id');
    if (data.chat_id !== chatId) {
        return; // Not for this chat
    }

    // Update the message list with new messages
    updateMessageList(data.messages);
}

function handleToolCallUpdate(data) {
    // Find the thinking message and add tool result to it
    const thinkingMessage = document.getElementById('thinking-message');
    if (!thinkingMessage) return;

    // Replace the thinking content with tool execution status
    const thinkingContent = thinkingMessage.querySelector('.thinking-content');
    if (thinkingContent && thinkingContent.textContent.includes('Thinking')) {
        thinkingContent.innerHTML = 'Executing tools...';
    }

    // Add tool result to the thinking message
    let toolContainer = thinkingMessage.querySelector('.tool-results');
    if (!toolContainer) {
        toolContainer = document.createElement('div');
        toolContainer.className = 'tool-results';
        thinkingMessage.appendChild(toolContainer);
    }

    // Add the tool result
    const toolResult = document.createElement('div');
    toolResult.className = 'tool-result';
    toolResult.innerHTML = `
        <div class="tool-name">↳ ${data.tool_name}</div>
        <div class="tool-output">${data.result || data.error || 'Tool executed'}</div>
        ${data.diff ? `<div class="tool-diff"><pre>${data.diff}</pre></div>` : ''}
    `;

    toolContainer.appendChild(toolResult);

    // Scroll to the bottom
    scrollToResponse();
}

function updateMessageList(messages) {
    const messageHistory = document.getElementById('message-history');
    if (!messageHistory) return;

    // For incremental updates, don't clear everything - just update what's changed
    const messagesContainer = document.getElementById('messages-container');
    if (!messagesContainer) return;

    // Keep the intro message if no messages
    let html = '';
    if (messages.length === 0) {
        html = '<div class="message-intro"><p>How can I help you today?</p></div>';
    } else {
        // Group messages by conversation session
        let userMsgIndex = 0;
        let inSession = false;

        messages.forEach((msg, i) => {
            // Start a new session when we see a user message
            if (msg.role === 'user' && !inSession) {
                html += `<div id="message-session-${userMsgIndex}" class="message-session">`;
                inSession = true;
                userMsgIndex++;
            }

            // Render the message based on role
            if (msg.role === 'user') {
                html += `<div class="message user-message"><div class="message-content">${marked.parse(msg.content)}</div></div>`;
            } else if (msg.role === 'assistant') {
                html += `<div class="message agent-message"><div class="message-content">${marked.parse(msg.content)}</div></div>`;
            } else if (msg.role === 'tool') {
                html += `<div class="message tool-message"><div class="message-content">${marked.parse(msg.content)}</div>`;
                if (msg.tool_call_events && msg.tool_call_events.length > 0) {
                    msg.tool_call_events.forEach(event => {
                        html += `<div class="tool-event"><div class="tool-name">${event.tool_name}</div>`;
                        if (event.result) {
                            html += `<div class="tool-result">${event.result}</div>`;
                        }
                        if (event.diff) {
                            html += `<div class="tool-diff"><pre>${event.diff}</pre></div>`;
                        }
                        if (event.error) {
                            html += `<div class="tool-error">${event.error}</div>`;
                        }
                        html += '</div>';
                    });
                }
                html += '</div>';
            }

            // End a session when we see the next user message or end of messages
            const isLastMessage = i === messages.length - 1;
            const nextIsUser = !isLastMessage && messages[i + 1].role === 'user';
            if ((nextIsUser || isLastMessage) && inSession) {
                html += '</div>';
                inSession = false;
            }
        });

        // Add placeholder for next message session
        html += '<div id="next-message-session"></div>';
    }

    messagesContainer.innerHTML = html;

    // Re-initialize copy buttons and scroll to bottom
    initCopyButtons();
    scrollToResponse();
}

document.addEventListener('DOMContentLoaded', () => {
    initCopyButtons();
    scrollToResponse();
    connectWebSocket();

    const textarea = document.getElementById('message-input');
    if (textarea) {
        textarea.addEventListener('keydown', function(event) {
            if (event.key === 'Enter' && !event.shiftKey) {
                event.preventDefault();  // Prevent newline insertion
                // HTMX will handle the form submission
            }
        });
    }
});
