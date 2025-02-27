let socket = null;

function connectWebSocket(ChatId) {
    if (socket) {
        socket.close();
    }
    const url = `ws://localhost:8080/api/ws/chat?Chat_id=${ChatId}`;
    socket = new WebSocket(url);

    socket.onopen = function() {
        console.log('WebSocket connection established');
    };

    socket.onmessage = function(event) {
        const message = JSON.parse(event.data);
        console.log('WebSocket message received:', message);
    };

    socket.onclose = function() {
        console.log('WebSocket connection closed');
    };

    socket.onerror = function(error) {
        console.error('WebSocket error:', error);
    };
}

document.body.addEventListener('htmx:load', function(event) {
    if (event.target.id === 'message-history') {
        const messagesContainer = event.target.querySelector('.messages');
        if (messagesContainer) {
            const ChatId = messagesContainer.dataset.ChatId;
            document.getElementById('Chat-id').value = ChatId;
            document.getElementById('message-input-section').style.display = 'block';
            connectWebSocket(ChatId);
        }
    }
});
