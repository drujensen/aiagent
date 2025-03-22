function scrollToBottom() {
    const messageHistory = document.getElementById('message-history');
    if (messageHistory) {
        messageHistory.scrollTop = messageHistory.scrollHeight;
    }
}

document.addEventListener('DOMContentLoaded', scrollToBottom);

function addCopyButtonToBlock(block) {
    // Skip if the block already has a copy button
    if (block.querySelector('.copy-button')) return;

    // Create copy button
    const button = document.createElement('button');
    button.className = 'copy-button';
    button.textContent = 'Copy';

    // Add click event
    button.addEventListener('click', () => {
        const code = block.querySelector('code')?.textContent || block.textContent;

        // Function to handle copying with fallback
        const copyToClipboard = (text) => {
            if (navigator.clipboard && window.isSecureContext) {
                // Modern Clipboard API (preferred)
                return navigator.clipboard.writeText(text).then(() => true).catch(() => false);
            } else {
                // Fallback using execCommand
                const textarea = document.createElement('textarea');
                textarea.value = text;
                document.body.appendChild(textarea);
                textarea.select();
                let success = false;
                try {
                    success = document.execCommand('copy');
                } catch (err) {
                    console.error('Fallback copy failed:', err);
                }
                document.body.removeChild(textarea);
                return Promise.resolve(success);
            }
        };

        copyToClipboard(code).then((success) => {
            if (success) {
                button.textContent = 'Copied!';
                button.classList.add('copied');
                setTimeout(() => {
                    button.textContent = 'Copy';
                    button.classList.remove('copied');
                }, 2000);
            } else {
                console.error('Failed to copy text');
                button.textContent = 'Failed';
                setTimeout(() => {
                    button.textContent = 'Copy';
                }, 2000);
            }
        });
    });

    // Add button to code block
    block.appendChild(button);
}

function addCopyButtons() {
    const codeBlocks = document.querySelectorAll('pre');
    codeBlocks.forEach(block => addCopyButtonToBlock(block));
}

// Run when page loads
document.addEventListener('DOMContentLoaded', addCopyButtons);

// Set up MutationObserver
const observer = new MutationObserver((mutationsList) => {
    for (let mutation of mutationsList) {
        if (mutation.type === 'childList' && mutation.addedNodes.length > 0) {
            mutation.addedNodes.forEach(node => {
                if (node.nodeName === 'PRE') {
                    addCopyButtonToBlock(node);
                } else if (node.querySelectorAll) {
                    const newCodeBlocks = node.querySelectorAll('pre');
                    newCodeBlocks.forEach(block => addCopyButtonToBlock(block));
                }
            });
        }
    }
});

// Start observing the message_history div
const targetNode = document.getElementById('message-history');
if (targetNode) {
    observer.observe(targetNode, { childList: true, subtree: true });
} else {
    console.error('Target node #message-history not found');
}
