function scrollToBottom() {
            const messageHistory = document.getElementById('message-history');
            if (messageHistory) {
                messageHistory.scrollTop = messageHistory.scrollHeight;
            }
        }

document.addEventListener('DOMContentLoaded', scrollToBottom);


function addCopyButtons() {
    // Find all pre elements
    const codeBlocks = document.querySelectorAll('pre');

    codeBlocks.forEach((block) => {
        // Create copy button
        const button = document.createElement('button');
        button.className = 'copy-button';
        button.textContent = 'Copy';

        // Add click event
        button.addEventListener('click', () => {
            // Get the code text
            const code = block.querySelector('code')?.textContent || block.textContent;

            // Copy to clipboard
            navigator.clipboard.writeText(code).then(() => {
                // Show success feedback
                button.textContent = 'Copied!';
                button.classList.add('copied');

                // Reset after 2 seconds
                setTimeout(() => {
                    button.textContent = 'Copy';
                    button.classList.remove('copied');
                }, 2000);
            }).catch(err => {
                console.error('Failed to copy:', err);
            });
        });

        // Add button to code block
        block.appendChild(button);
    });
}

// Run when page loads
document.addEventListener('DOMContentLoaded', addCopyButtons);

// Set up MutationObserver
const observer = new MutationObserver((mutationsList, observer) => {
    for (let mutation of mutationsList) {
        if (mutation.type === 'childList') {
            // New elements have been added to the message_history div
            addCopyButtons();
        }
    }
});

// Start observing the message_history div
const targetNode = document.getElementById('message-history');
observer.observe(targetNode, { childList: true });
