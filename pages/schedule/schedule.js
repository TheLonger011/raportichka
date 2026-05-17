let currentTab = 'schedule';
let allFiles = { schedule: [], substitutions: [] };

async function init() {
    await loadFiles();
    adaptMobileButtons();
}

async function loadFiles() {
    try {
        const r = await fetch('/api/schedule/files');
        allFiles = await r.json();
        renderPanel('schedule');
        renderPanel('substitutions');
    } catch (e) {
        document.getElementById('panel-schedule').innerHTML = '<div class="loading">Ошибка загрузки</div>';
        document.getElementById('panel-substitutions').innerHTML = '<div class="loading">Ошибка загрузки</div>';
    }
}

function renderPanel(category) {
    const panel = document.getElementById(`panel-${category}`);
    const files = allFiles[category] || [];

    if (files.length === 0) {
        panel.innerHTML = `
            <div class="empty-state">
                <svg viewBox="0 0 24 24"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
                <p>Файлов нет</p>
                <small>Нажмите «Обновить» чтобы скачать файлы с сайта колледжа</small>
            </div>`;
        return;
    }

    panel.innerHTML = `<div class="files-grid">${files.map(f => fileCard(f, category)).join('')}</div>`;
}

function fileCard(f, category) {
    const icon = typeIcon(f.type);
    const size = formatSize(f.size);
    const date = new Date(f.modified).toLocaleDateString('ru', { day: '2-digit', month: 'short', year: 'numeric' });

    return `
        <div class="file-card">
            <div class="file-card-top">
                <div class="file-icon ${f.type}">${icon}</div>
                <div class="file-info">
                    <div class="file-name">${esc(f.name)}</div>
                    <div class="file-meta">${size} · ${date}</div>
                </div>
            </div>
            <div class="file-actions">
                <a class="btn-open" href="${f.path}" target="_blank">
                    <svg viewBox="0 0 24 24"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
                    Открыть
                </a>
                <button class="btn-del" onclick="deleteFile('${category}','${esc(f.name)}')" title="Удалить">
                    <svg viewBox="0 0 24 24"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6"/><path d="M14 11v6"/><path d="M9 6V4h6v2"/></svg>
                </button>
            </div>
        </div>`;
}

function typeIcon(type) {
    switch (type) {
        case 'pdf':   return '📄';
        case 'word':  return '📝';
        case 'image': return '🖼️';
        case 'excel': return '📊';
        default:      return '📁';
    }
}

function formatSize(bytes) {
    if (bytes < 1024) return bytes + ' Б';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(0) + ' КБ';
    return (bytes / (1024 * 1024)).toFixed(1) + ' МБ';
}

function switchTab(tab) {
    currentTab = tab;
    document.getElementById('tab-schedule').classList.toggle('active', tab === 'schedule');
    document.getElementById('tab-substitutions').classList.toggle('active', tab === 'substitutions');
    document.getElementById('panel-schedule').style.display = tab === 'schedule' ? '' : 'none';
    document.getElementById('panel-substitutions').style.display = tab === 'substitutions' ? '' : 'none';
}

async function syncFiles() {
    const btn = document.getElementById('syncBtn');
    const status = document.getElementById('syncStatus');
    btn.disabled = true;
    btn.classList.add('spinning');
    status.textContent = 'Синхронизация...';
    try {
        await fetch('/api/schedule/sync', { method: 'POST' });
        await new Promise(r => setTimeout(r, 3000));
        await loadFiles();
        status.textContent = 'Обновлено ✓';
    } catch (e) {
        status.textContent = 'Ошибка';
    } finally {
        btn.disabled = false;
        btn.classList.remove('spinning');
        setTimeout(() => status.textContent = '', 3000);
    }
}

async function deleteFile(category, name) {
    if (!confirm(`Удалить файл «${name}»?`)) return;
    try {
        await fetch('/api/schedule/delete', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ category, name })
        });
        await loadFiles();
    } catch (e) {
        alert('Ошибка при удалении');
    }
}

function esc(s) {
    if (!s) return '';
    return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

function adaptMobileButtons() {
    const syncBtn = document.getElementById('syncBtn');
    if (!syncBtn) return;

    const syncTextSpan = syncBtn.querySelector('.sync-text');

    if (window.innerWidth <= 640) {
        syncBtn.style.width = '38px';
        syncBtn.style.height = '38px';
        syncBtn.style.padding = '0';
        syncBtn.style.fontSize = '0';
        syncBtn.style.gap = '0';
        syncBtn.style.justifyContent = 'center';
        syncBtn.style.alignItems = 'center';
        syncBtn.style.minWidth = '38px';

        const svg = syncBtn.querySelector('svg');
        if (svg) {
            svg.style.width = '18px';
            svg.style.height = '18px';
            svg.style.margin = '0';
        }

        if (syncTextSpan) {
            syncTextSpan.style.display = 'none';
            syncTextSpan.style.visibility = 'hidden';
        }

        const syncStatus = document.getElementById('syncStatus');
        if (syncStatus) {
            syncStatus.style.display = 'none';
        }
    } else {
        syncBtn.style.width = '';
        syncBtn.style.height = '';
        syncBtn.style.padding = '';
        syncBtn.style.fontSize = '';
        syncBtn.style.gap = '';
        syncBtn.style.justifyContent = '';
        syncBtn.style.alignItems = '';
        syncBtn.style.minWidth = '';

        const svg = syncBtn.querySelector('svg');
        if (svg) {
            svg.style.width = '';
            svg.style.height = '';
            svg.style.margin = '';
        }

        if (syncTextSpan) {
            syncTextSpan.style.display = '';
            syncTextSpan.style.visibility = '';
        }

        const syncStatus = document.getElementById('syncStatus');
        if (syncStatus) {
            syncStatus.style.display = '';
        }
    }
}

window.addEventListener('resize', function() {
    adaptMobileButtons();
});

window.addEventListener('orientationchange', function() {
    setTimeout(adaptMobileButtons, 100);
});

init();