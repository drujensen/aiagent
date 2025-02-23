let socket = null;
let currentConversationId = null;

function connectWebSocket(conversationId) {
    if (socket) {
        socket.close();
    }
    const apiKey = 'default-api-key'; // Replace with secure retrieval
    const url = `ws://localhost:8080/api/ws/chat?conversation_id=${conversationId}&api_key=${apiKey}`;
    socket = new WebSocket(url);

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
    const messagesDiv = document.querySelector('.messages');
    if (!messagesDiv) return;
    const messageDiv = document.createElement('div');
    messageDiv.className = `message ${message.role === 'user' ? 'user-message' : 'agent-message'}`;
    messageDiv.innerHTML = `<strong>${message.role} (${message.timestamp}):</strong> ${message.content}`;
    messagesDiv.appendChild(messageDiv);
    messagesDiv.scrollTop = messagesDiv.scrollHeight;
}

document.addEventListener('htmx:afterSwap', function(event) {
    if (event.target.id === 'message-history') {
        const messagesDiv = event.target.querySelector('.messages');
        if (messagesDiv) {
            currentConversationId = messagesDiv.dataset.conversationId;
            document.getElementById('conversation-id').value = currentConversationId;
            connectWebSocket(currentConversationId);
            messagesDiv.scrollTop = messagesDiv.scrollHeight;
        } else {
            // New conversation started
            currentConversationId = event.detail.xhr.responseText; // Assuming the response is the new conversation ID
            document.getElementById('conversation-id').value = currentConversationId;
            connectWebSocket(currentConversationId);
        }
    }
});

document.getElementById('message-form').addEventListener('submit', function(event) {
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
