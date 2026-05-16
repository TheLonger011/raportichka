let groups = [], curGroup = null, subjects = [];

async function init() {
    await loadGroups();
}

async function loadGroups() {
    const grid = document.getElementById('groupsGrid');
    grid.innerHTML = '<div class="loading"><div class="spinner"></div>Загрузка...</div>';
    try {
        const r = await fetch('/api/groups');
        groups = await r.json() || [];

        const counts = await Promise.all(
            groups.map(g => fetch(`/api/students?group_id=${g.id}`).then(r => r.json()).then(s => s ? s.length : 0))
        );

        const total = counts.reduce((a, b) => a + b, 0);
        const navStat = document.getElementById('navStat');
        if (navStat) navStat.textContent = `${groups.length} групп · ${total} студентов`;

        grid.innerHTML = groups.map((g, i) => `
            <div class="group-card" onclick="selectGroup(${g.id})">
                <div class="group-card-top">
                    <div class="group-name">${esc(g.name)}</div>
                    <div class="group-icon">
                        <svg viewBox="0 0 24 24"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
                    </div>
                </div>
                <div class="group-meta">
                    <span class="group-badge">
                        <svg viewBox="0 0 24 24"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
                        ${counts[i]} студентов
                    </span>
                    <span class="group-badge" style="color:#22c55e;border-color:#bbf7d0;background:#f0fdf4">активная</span>
                </div>
                <div class="group-footer">
                    <span class="group-link">Журнал группы <svg viewBox="0 0 24 24"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg></span>
                </div>
            </div>
        `).join('');
    } catch (e) {
        grid.innerHTML = '<div class="loading">Ошибка загрузки</div>';
    }
}

function selectGroup(id) {
    curGroup = groups.find(g => g.id === id);
    showSubjects();
}

async function showSubjects() {
    document.getElementById('view-groups').style.display = 'none';
    document.getElementById('view-subjects').style.display = '';
    document.getElementById('subjPageTitle').textContent = curGroup.name;

    const grid = document.getElementById('subjectsGrid');
    grid.innerHTML = '<div class="loading"><div class="spinner"></div>Загрузка...</div>';
    try {
        const r = await fetch(`/api/subjects?group_id=${curGroup.id}`);
        subjects = await r.json() || [];
        grid.innerHTML = subjects.map(s => `
            <div class="subject-card" onclick="goGrades(${s.id})">${esc(s.name)}</div>
        `).join('');
    } catch (e) {
        grid.innerHTML = '<div class="loading">Ошибка загрузки</div>';
    }
}

function showGroups() {
    document.getElementById('view-subjects').style.display = 'none';
    document.getElementById('view-groups').style.display = '';
    curGroup = null;
}

function goGrades(subjectId) {
    const subject = subjects.find(s => s.id === subjectId);
    sessionStorage.setItem('gradeCtx', JSON.stringify({ group: curGroup, subject }));
    window.location.href = '/grades';
}

function esc(s) {
    if (!s) return '';
    return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

init();