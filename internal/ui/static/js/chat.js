let socket = null;
let currentConversationId = null;

function connectWebSocket(conversationId) {
    if (socket) {
        socket.close();
    }
    const apiKey = 'default-api-key'; // Should be securely retrieved from backend or env
    const url = `ws://localhost:8080/api/ws/chat?conversation_id=${conversationId}&api_key=${apiKey}`;
    socket = new WebSocket(url);

    socket.onopen = function() {
        console.log('WebSocket connection established');
    };

    socket.onmessage = function(event) {
        const message = JSON.parse(event.data);
        appendMessage(message);
    };

    socket.onclose = function() {
        console.log('WebSocket connection closed');
    };

    socket.onerror = function(error) {
        console.error('WebSocket error:', error);
    };
}

function appendMessage(message) {
    const messagesDiv = document.querySelector('#messages-container .messages');
    if (!messagesDiv) return;
    const messageDiv = document.createElement('div');
    messageDiv.className = `message ${message.role === 'user' ? 'user-message' : message.role === 'assistant' ? 'agent-message' : 'tool-message'}`;
    messageDiv.innerHTML = `<strong>${message.role} (${new Date(message.timestamp).toLocaleString()}):</strong> ${message.content}`;
    messagesDiv.appendChild(messageDiv);
    messagesDiv.scrollTop = messagesDiv.scrollHeight;
}

function refreshSidebar() {
    fetch('/sidebar/conversations')
        .then(response => {
            if (!response.ok) throw new Error('Failed to refresh sidebar');
            return response.text();
        })
        .then(html => {
            document.getElementById('sidebar-conversations').innerHTML = html;
        })
        .catch(error => console.error('Error refreshing sidebar:', error));
}

document.addEventListener('DOMContentLoaded', function() {
    // Initial sidebar load
    refreshSidebar();

    const startButton = document.getElementById('start-conversation');
    if (startButton) {
        startButton.addEventListener('click', function() {
            const agentId = document.getElementById('agent-select').value;
            if (!agentId) {
                alert('Please select an AI Agent');
                return;
            }

            fetch('/api/conversations', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-API-Key': 'default-api-key' // Should be securely retrieved
                },
                body: JSON.stringify({ agent_id: agentId })
            })
            .then(response => {
                if (!response.ok) throw new Error('Failed to start conversation');
                return response.json();
            })
            .then(data => {
                currentConversationId = data.id;
                document.getElementById('conversation-id').value = currentConversationId;
                document.getElementById('conversation-starter').style.display = 'none';
                const messagesContainer = document.getElementById('messages-container');
                messagesContainer.style.display = 'block';
                messagesContainer.innerHTML = '<div class="messages"></div>';
                document.getElementById('message-input-section').style.display = 'flex';
                connectWebSocket(currentConversationId);
                // Refresh sidebar after creating a new conversation
                refreshSidebar();
            })
            .catch(error => {
                console.error('Error starting conversation:', error);
                alert('Failed to start conversation');
            });
        });
    }

    const messageForm = document.getElementById('message-form');
    if (messageForm) {
        messageForm.addEventListener('submit', function(event) {
            event.preventDefault();
            const input = document.getElementById('message-input');
            const message = input.value.trim();
            if (!message || !currentConversationId || !socket || socket.readyState !== WebSocket.OPEN) {
                console.warn('Cannot send message: No conversation selected or WebSocket not open');
                return;
            }
            const payload = {
                conversation_id: currentConversationId,
                message: {
                    role: 'user',
                    content: message,
                    timestamp: new Date().toISOString()
                }
            };
            socket.send(JSON.stringify(payload));
            input.value = '';
        });
    }
});

document.addEventListener('htmx:afterSwap', function(event) {
    if (event.target.id === 'message-history') {
        const messagesDiv = event.target.querySelector('.messages');
        if (messagesDiv) {
            currentConversationId = messagesDiv.dataset.conversationId;
            document.getElementById('conversation-id').value = currentConversationId;
            document.getElementById('conversation-starter').style.display = 'none';
            document.getElementById('messages-container').style.display = 'block';
            document.getElementById('message-input-section').style.display = 'flex';
            connectWebSocket(currentConversationId);
            messagesDiv.scrollTop = messagesDiv.scrollHeight;
        }
    }
});
