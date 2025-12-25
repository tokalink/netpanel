// VPS Panel - Main JavaScript

// Utility functions
function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function formatUptime(seconds) {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    
    if (days > 0) return `${days}d ${hours}h`;
    if (hours > 0) return `${hours}h ${mins}m`;
    return `${mins}m`;
}

// Toggle sidebar for mobile
function toggleSidebar() {
    document.querySelector('.sidebar').classList.toggle('open');
}

// Logout function
async function logout() {
    try {
        await fetch('/api/auth/logout', { method: 'POST' });
    } catch (err) {
        console.error('Logout error:', err);
    }
    window.location.href = '/login';
}

// API helper
async function api(endpoint, options = {}) {
    const response = await fetch(`/api${endpoint}`, {
        headers: {
            'Content-Type': 'application/json',
            ...options.headers
        },
        ...options
    });
    
    if (response.status === 401) {
        window.location.href = '/login';
        return null;
    }
    
    return response.json();
}

// Load user profile
async function loadProfile() {
    try {
        const user = await api('/auth/profile');
        if (user) {
            document.getElementById('userName').textContent = user.username;
            document.getElementById('userRole').textContent = user.role === 'admin' ? 'Administrator' : 'User';
            document.getElementById('userAvatar').textContent = user.username.charAt(0).toUpperCase();
        }
    } catch (err) {
        console.error('Failed to load profile:', err);
    }
}

// Initialize on dashboard pages
if (window.location.pathname.startsWith('/dashboard')) {
    loadProfile();
}
