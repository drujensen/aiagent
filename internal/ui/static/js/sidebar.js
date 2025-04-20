function updateSidebarTransform() {
    const sidebar = document.querySelector('.sidebar');
    if (sidebar.classList.contains('active')) {
        sidebar.style.transform = 'translateX(0)';
    } else {
        sidebar.style.transform = 'translateX(100%)';
    }
}

document.addEventListener('DOMContentLoaded', function() {
    // Close sidebar when clicking outside on mobile
    document.addEventListener('click', function(event) {
        const sidebar = document.querySelector('.sidebar');
        const hamburger = document.querySelector('.hamburger');
        if (hamburger.contains(event.target)) {
              sidebar.classList.toggle('active');
              updateSidebarTransform();
        }

    });
});

document.addEventListener('htmx:afterRequest', function(event) {
    if (event.detail.target.classList.contains('sidebar')) {
        updateSidebarTransform();
    }
});
