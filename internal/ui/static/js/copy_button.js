function addCopyButtonToBlock(block) {
    if (block.querySelector('.copy-button')) return;
    const button = document.createElement('button');
    button.className = 'copy-button';
    button.textContent = 'Copy';
    button.addEventListener('click', () => {
        const code = block.querySelector('code')?.textContent || block.textContent;
        const copyToClipboard = (text) => {
            if (navigator.clipboard && window.isSecureContext) {
                return navigator.clipboard.writeText(text).then(() => true).catch(() => false);
            } else {
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
    block.appendChild(button);
}

function initCopyButtons() {
    const codeBlocks = document.querySelectorAll('pre');
    codeBlocks.forEach(block => addCopyButtonToBlock(block));
}

document.addEventListener('DOMContentLoaded', initCopyButtons);
