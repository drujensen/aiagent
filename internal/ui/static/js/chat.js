class ChatWebSocket {
    constructor() {
        this.socket = null;
        this.reconnectTimeout = null;
        this.chatId = null;
    }

    connect(chatId) {
        this.chatId = chatId;
        if (this.socket) {
            this.socket.close();
        }

        const url = `ws://${window.location.host}/api/ws/chat?chat_id=${chatId}`;
        this.socket = new WebSocket(url);

        this.socket.onopen = this.onOpen.bind(this);
        this.socket.onmessage = this.onMessage.bind(this);
        this.socket.onclose = this.onClose.bind(this);
        this.socket.onerror = this.onError.bind(this);
    }

    onOpen() {
        console.log('WebSocket connection established');
        this.clearReconnectTimeout();
    }

    onMessage(event) {
        try {
            const message = JSON.parse(event.data);
            console.log('WebSocket message received:', message);
            const messagesContainer = document.getElementById('messages-container');
            if (messagesContainer) {
                const messageElement = document.createElement('div');
                messageElement.classList.add('message');
                if (message.role === 'user') {
                    messageElement.classList.add('user-message');
                } else if (message.role === 'assistant') {
                    messageElement.classList.add('agent-message');
                } else {
                    messageElement.classList.add('tool-message');
                }
                messageElement.innerHTML = `${message.content}`;
                messagesContainer.appendChild(messageElement);
                // Scroll to the bottom
                messagesContainer.scrollTop = messagesContainer.scrollHeight;
            }
        } catch (error) {
            console.error('Error parsing message:', error);
        }
    }

    onClose() {
        console.log('WebSocket connection closed');
        this.scheduleReconnect();
    }

    onError(error) {
        console.error('WebSocket error:', error);
        this.scheduleReconnect();
    }

    scheduleReconnect() {
        if (!this.reconnectTimeout) {
            this.reconnectTimeout = setTimeout(() => {
                this.connect(this.chatId);
            }, 5000); // Reconnect after 5 seconds
        }
    }

    clearReconnectTimeout() {
        if (this.reconnectTimeout) {
            clearTimeout(this.reconnectTimeout);
            this.reconnectTimeout = null;
        }
    }

    sendMessage(message) {
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
            // Append the user's message to the UI immediately
            const messagesContainer = document.getElementById('messages-container');
            if (messagesContainer) {
                const messageElement = document.createElement('div');
                messageElement.classList.add('message', 'user-message');
                messageElement.innerHTML = `${message}`;
                messagesContainer.appendChild(messageElement);
                messagesContainer.scrollTop = messagesContainer.scrollHeight;
            }
            this.socket.send(JSON.stringify({
                Chat_id: this.chatId,
                message: {
                    role: 'user',
                    content: message
                }
            }));
        } else {
            console.error('WebSocket not connected. Cannot send message.');
        }
    }
}

const chatWebSocket = new ChatWebSocket();

// Wait for the DOM to be fully loaded
document.addEventListener('DOMContentLoaded', function() {
    const messageForm = document.getElementById('message-form');
    const messagesContainer = document.getElementById('messages-container');

    // Check if required elements exist
    if (!messagesContainer) {
        console.error('messages-container not found');
        return;
    }
    if (!messageForm) {
        console.error('message-form not found');
        return;
    }

    const chatId = messagesContainer.dataset.chatId;
    if (chatId && /^[0-9a-fA-F]{24}$/.test(chatId)) {
        // Valid 24-character hex string
        document.getElementById('chat-id').value = chatId;
        document.getElementById('message-input-section').style.display = 'block';
        chatWebSocket.connect(chatId);

        // Set up form submission
        messageForm.addEventListener('submit', function(event) {
            event.preventDefault();
            const messageInput = document.getElementById('message-input');
            const message = messageInput.value.trim();
            if (message) {
                chatWebSocket.sendMessage(message);
                messageInput.value = ''; // Clear the input after sending
            }
        });
    } else {
        console.error('Invalid chat ID:', chatId);
    }
});
