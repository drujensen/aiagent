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
        if (data.type === 'message_history_refresh') {
            handleMessageHistoryRefresh(data);
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

function handleMessageHistoryRefresh(data) {
    const chatId = document.getElementById('messages-container').getAttribute('data-chat-id');
    if (data.chat_id !== chatId) {
        return; // Not for this chat
    }

    // Fetch the latest messages from the server
    fetchLatestMessages(chatId);
}

async function fetchLatestMessages(chatId) {
    try {
        const response = await fetch(`/chats/${chatId}/messages`);
        if (!response.ok) {
            console.error('Failed to fetch messages:', response.statusText);
            return;
        }

        const data = await response.json();
        if (data.messages) {
            // Clear any temporary tool results before updating with server data
            const thinkingMessage = document.getElementById('thinking-message');
            if (thinkingMessage) {
                const toolContainer = thinkingMessage.querySelector('.tool-results');
                if (toolContainer) {
                    toolContainer.innerHTML = '';
                }
            }

            updateMessageList(data.messages);
        }
    } catch (error) {
        console.error('Error fetching messages:', error);
    }
}

// formatToolResult formats tool execution results for display (similar to TUI)
function formatToolResult(toolName, result, diff, error) {
    if (error) {
        return `<div class="tool-error">Error: ${error}</div>`;
    }

    switch (toolName) {
        case "FileWrite":
            return formatFileWriteResult(result, diff);
        case "FileSearch":
            return formatFileSearchResult(result);
        case "Memory":
            return formatMemoryResult(result);
        default:
            // Try to extract summary from JSON responses
            try {
                const jsonResponse = JSON.parse(result);
                if (jsonResponse.summary) {
                    return jsonResponse.summary;
                }
            } catch (e) {
                // Not JSON or no summary, return as-is
            }
            return result;
    }
}

// formatFileWriteResult formats FileWrite tool results
function formatFileWriteResult(result, diff) {
    try {
        const resultData = JSON.parse(result);
        let output = '';

        // Use the summary from the JSON response
        if (resultData.summary) {
            output += resultData.summary;
        } else {
            output += 'File modified successfully';
        }

        // Add the diff if available
        if (diff) {
            output += '\n\n' + formatDiff(diff);
        } else if (resultData.diff) {
            output += '\n\n' + formatDiff(resultData.diff);
        }

        return output;
    } catch (e) {
        // If parsing fails, try to extract summary
        try {
            const jsonResponse = JSON.parse(result);
            if (jsonResponse.summary) {
                return jsonResponse.summary;
            }
        } catch (e2) {
            // Return raw result if parsing fails
        }
        return result;
    }
}

// formatFileSearchResult formats FileSearch tool results
function formatFileSearchResult(result) {
    try {
        const response = JSON.parse(result);
        if (response.summary) {
            return response.summary;
        }
    } catch (e) {
        // If parsing fails, return the original result
    }
    return result;
}

// formatMemoryResult formats Memory tool results
function formatMemoryResult(result) {
    try {
        const data = JSON.parse(result);
        let output = '';

        // Try parsing as entities array
        if (Array.isArray(data) && data.length > 0) {
            const firstItem = data[0];
            if (firstItem.name && firstItem.type) {
                // This looks like entities
                output += `<div class="memory-entities">Memory Entities (${data.length} created):<ul>`;
                const maxEntities = 5;
                for (let i = 0; i < Math.min(data.length, maxEntities); i++) {
                    const entity = data[i];
                    output += `<li>${entity.name} (${entity.type})</li>`;
                }
                if (data.length > maxEntities) {
                    output += `<li>... and ${data.length - maxEntities} more entities</li>`;
                }
                output += '</ul></div>';
            } else if (firstItem.source && firstItem.type && firstItem.target) {
                // This looks like relations
                output += `<div class="memory-relations">Memory Relations (${data.length} created):<ul>`;
                const maxRelations = 5;
                for (let i = 0; i < Math.min(data.length, maxRelations); i++) {
                    const relation = data[i];
                    output += `<li>${relation.source} --${relation.type}--> ${relation.target}</li>`;
                }
                if (data.length > maxRelations) {
                    output += `<li>... and ${data.length - maxRelations} more relations</li>`;
                }
                output += '</ul></div>';
            } else {
                // Unknown array format
                output += `Memory operation completed (${data.length} items)`;
            }
        } else {
            // Try to extract summary
            if (data.summary) {
                output += data.summary;
            } else {
                output += 'Memory operation completed';
            }
        }

        return output;
    } catch (e) {
        return result;
    }
}

// formatDiff formats diff content for display
function formatDiff(diff) {
    if (!diff) return '';

    let diffContent = diff;

    // Extract diff content from markdown code block if present
    if (diff.includes('```diff')) {
        const start = diff.indexOf('```diff\n');
        if (start !== -1) {
            const startPos = start + 8; // Length of "```diff\n"
            const end = diff.indexOf('\n```', startPos);
            if (end !== -1) {
                diffContent = diff.substring(startPos, end);
            } else {
                diffContent = diff.substring(startPos);
            }
        }
    }

    // Check if this looks like a unified diff
    const hasDiffMarkers = diffContent.includes('---') ||
                          diffContent.includes('+++') ||
                          diffContent.includes('@@');

    if (!hasDiffMarkers) {
        // If it doesn't look like a diff, just return the content
        return diffContent;
    }

    // Format as a diff with syntax highlighting
    return `<div class="diff-content">${diffContent}</div>`;
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

    // Format the tool result using the same logic as TUI
    const formattedResult = formatToolResult(data.tool_name, data.result, data.diff, data.error);

    // Add the tool result
    const toolResult = document.createElement('div');
    toolResult.className = `tool-result ${data.error ? 'error' : 'success'}`;
    toolResult.innerHTML = `
        <div class="tool-name">${data.tool_name}</div>
        <div class="tool-output">${formattedResult}</div>
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
