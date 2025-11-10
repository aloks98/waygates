const routes = {
    '/': `
        <h2>Home Page</h2>
        <p>This is a SPA test. All routes should serve this same index.html file.</p>
        <p>Try clicking the navigation links - they won't cause page reloads!</p>
        <p>Also try requesting an asset directly, like <a href="/css/style.css">/css/style.css</a>. It should be served correctly.</p>
    `,
    '/about': `
        <h2>About Page</h2>
        <p>This demonstrates client-side routing in a SPA.</p>
        <p>The URL changes but the page doesn't reload.</p>
    `,
    '/contact': `
        <h2>Contact Page</h2>
        <p>Another client-side route.</p>
    `
};

function navigate(e, path) {
    e.preventDefault();
    history.pushState(null, '', path);
    render();
}

function render() {
    const path = window.location.pathname;
    document.getElementById('app').innerHTML = routes[path] || `
        <h2>404 - Not Found (Client Side)</h2>
        <p>Path: ${path}</p>
        <p>This shows the fallback working - unknown routes still serve index.html</p>
    `;
}

// Make navigate function global so it can be called from onclick attributes
window.navigate = navigate;

window.addEventListener('popstate', render);
// Initial render
render();
