// Dashboard specific JavaScript

let cpuChart, memChart;
let cpuHistory = [];
let memHistory = [];
const MAX_HISTORY = 30;

// Initialize charts
function initCharts() {
    const chartOptions = {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
            legend: { display: false }
        },
        scales: {
            y: {
                beginAtZero: true,
                max: 100,
                ticks: { callback: v => v + '%' }
            },
            x: {
                display: false
            }
        },
        elements: {
            line: { tension: 0.4 },
            point: { radius: 0 }
        }
    };

    cpuChart = new Chart(document.getElementById('cpuChart'), {
        type: 'line',
        data: {
            labels: Array(MAX_HISTORY).fill(''),
            datasets: [{
                data: Array(MAX_HISTORY).fill(0),
                borderColor: '#3b82f6',
                backgroundColor: 'rgba(59, 130, 246, 0.1)',
                fill: true,
                borderWidth: 2
            }]
        },
        options: chartOptions
    });

    memChart = new Chart(document.getElementById('memChart'), {
        type: 'line',
        data: {
            labels: Array(MAX_HISTORY).fill(''),
            datasets: [{
                data: Array(MAX_HISTORY).fill(0),
                borderColor: '#10b981',
                backgroundColor: 'rgba(16, 185, 129, 0.1)',
                fill: true,
                borderWidth: 2
            }]
        },
        options: chartOptions
    });
}

// Update dashboard with stats
function updateDashboard(stats) {
    // CPU
    const cpuUsage = stats.cpu.usage_percent.toFixed(1);
    document.getElementById('cpuUsage').textContent = cpuUsage + '%';
    document.getElementById('cpuBar').style.width = cpuUsage + '%';

    // Memory
    const memUsage = stats.memory.used_percent.toFixed(1);
    document.getElementById('memUsage').textContent = memUsage + '%';
    document.getElementById('memBar').style.width = memUsage + '%';

    // Disk (primary partition)
    if (stats.disk && stats.disk.length > 0) {
        const primaryDisk = stats.disk[0];
        const diskUsage = primaryDisk.used_percent.toFixed(1);
        document.getElementById('diskUsage').textContent = diskUsage + '%';
        document.getElementById('diskBar').style.width = diskUsage + '%';

        // Update disk list
        const diskList = document.getElementById('diskList');
        diskList.innerHTML = stats.disk.map(d => `
            <div class="disk-item">
                <div class="disk-item-header">
                    <span class="disk-name">${d.mountpoint}</span>
                    <span class="disk-percent">${d.used_percent.toFixed(1)}%</span>
                </div>
                <div class="disk-bar">
                    <div class="disk-bar-fill" style="width: ${d.used_percent}%"></div>
                </div>
                <div class="disk-info">
                    <span>Used: ${formatBytes(d.used)}</span>
                    <span>Free: ${formatBytes(d.free)}</span>
                    <span>Total: ${formatBytes(d.total)}</span>
                </div>
            </div>
        `).join('');
    }

    // Network
    document.getElementById('netInfo').textContent =
        `↓ ${formatBytes(stats.network.bytes_recv)} / ↑ ${formatBytes(stats.network.bytes_sent)}`;

    // Host info
    document.getElementById('hostname').textContent = stats.host.hostname;
    document.getElementById('uptime').textContent = formatUptime(stats.host.uptime);
    document.getElementById('osInfo').textContent = stats.host.os;
    document.getElementById('platformInfo').textContent =
        `${stats.host.platform} ${stats.host.platform_version}`;
    document.getElementById('archInfo').textContent = stats.host.kernel_arch;
    document.getElementById('coresInfo').textContent = stats.cpu.cores;
    document.getElementById('totalMemInfo').textContent = formatBytes(stats.memory.total);

    // Update charts
    cpuHistory.push(parseFloat(cpuUsage));
    memHistory.push(parseFloat(memUsage));

    if (cpuHistory.length > MAX_HISTORY) cpuHistory.shift();
    if (memHistory.length > MAX_HISTORY) memHistory.shift();

    cpuChart.data.datasets[0].data = [...cpuHistory];
    memChart.data.datasets[0].data = [...memHistory];
    cpuChart.update('none');
    memChart.update('none');
}

// Connect to WebSocket for real-time updates
function connectWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws/stats`);

    ws.onopen = () => {
        console.log('WebSocket connected');
    };

    ws.onmessage = (event) => {
        try {
            const stats = JSON.parse(event.data);
            updateDashboard(stats);
        } catch (err) {
            console.error('Failed to parse stats:', err);
        }
    };

    ws.onclose = () => {
        console.log('WebSocket disconnected, reconnecting in 3s...');
        setTimeout(connectWebSocket, 3000);
    };

    ws.onerror = (err) => {
        console.error('WebSocket error:', err);
        ws.close();
    };
}

// Initial load
async function loadInitialStats() {
    try {
        const data = await api('/dashboard');
        if (data && data.stats) {
            updateDashboard(data.stats);
        }
    } catch (err) {
        console.error('Failed to load initial stats:', err);
    }
}

// Initialize dashboard
document.addEventListener('DOMContentLoaded', () => {
    initCharts();
    loadInitialStats();
    connectWebSocket();
});
