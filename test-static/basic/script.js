// Update message when page loads
document.addEventListener('DOMContentLoaded', function() {
    const message = document.getElementById('js-message');
    message.textContent = 'JavaScript loaded successfully!';
    message.classList.add('success');
});

// Test function for button click
function testFunction() {
    alert('JavaScript is working! Button clicked at: ' + new Date().toLocaleTimeString());
}

// Log to console
console.log('Static file test - JavaScript loaded');
console.log('Current URL:', window.location.href);
