let groups = [], curGroup = null, subjects = [], students = [];

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

    const subjGrid = document.getElementById('subjectsGrid');
    const studList = document.getElementById('studentsList');
    subjGrid.innerHTML = '<div class="loading"><div class="spinner"></div></div>';
    studList.innerHTML = '<div class="loading"><div class="spinner"></div></div>';

    try {
        const [subjResp, studResp] = await Promise.all([
            fetch(`/api/subjects?group_id=${curGroup.id}`),
            fetch(`/api/students?group_id=${curGroup.id}`)
        ]);
        subjects = await subjResp.json() || [];
        students = await studResp.json() || [];

        renderSubjects();
        renderStudents();
    } catch (e) {
        subjGrid.innerHTML = '<div class="loading">Ошибка загрузки</div>';
        studList.innerHTML = '<div class="loading">Ошибка загрузки</div>';
    }
}

function renderSubjects() {
    const grid = document.getElementById('subjectsGrid');
    if (!subjects.length) {
        grid.innerHTML = '<div class="empty-hint">Предметы не добавлены</div>';
        return;
    }
    grid.innerHTML = subjects.map(s => `
        <div class="subject-card" onclick="goGrades(${s.id})">${esc(s.name)}</div>
    `).join('');
}

function renderStudents() {
    const list = document.getElementById('studentsList');
    const counter = document.getElementById('studentsCount');
    if (counter) counter.textContent = students.length;

    if (!students.length) {
        list.innerHTML = '<div class="empty-hint">Студентов пока нет</div>';
        return;
    }
    list.innerHTML = students.map((st, i) => `
        <div class="student-row" id="strow-${st.id}">
            <div class="student-num">${i + 1}</div>
            <div class="student-name-text">${esc(st.full_name)}</div>
            <button class="btn-del-student" onclick="confirmDeleteStudent(${st.id}, '${esc(st.full_name).replace(/'/g,"&#39;")}')" title="Удалить студента">
                <svg viewBox="0 0 24 24"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6"/><path d="M14 11v6"/><path d="M9 6V4h6v2"/></svg>
            </button>
        </div>
    `).join('');
}

function showGroups() {
    document.getElementById('view-subjects').style.display = 'none';
    document.getElementById('view-groups').style.display = '';
    curGroup = null;
    subjects = [];
    students = [];
}

function goGrades(subjectId) {
    const subject = subjects.find(s => s.id === subjectId);
    sessionStorage.setItem('gradeCtx', JSON.stringify({ group: curGroup, subject }));
    window.location.href = '/grades';
}

function openAddStudentModal() {
    const existing = document.getElementById('addStudentOverlay');
    if (existing) existing.remove();

    const overlay = document.createElement('div');
    overlay.className = 'overlay on';
    overlay.id = 'addStudentOverlay';
    overlay.innerHTML = `
        <div class="modal-box" style="width:380px">
            <div class="modal-head">
                <div class="m-name">Добавить студента</div>
                <div class="m-date">${esc(curGroup.name)}</div>
            </div>
            <div class="modal-body">
                <div style="display:flex;flex-direction:column;gap:6px">
                    <label style="font-size:12.5px;font-weight:600;color:var(--muted);text-transform:uppercase;letter-spacing:.04em">Фамилия И.О.</label>
                    <input
                        type="text"
                        id="newStudentName"
                        placeholder="Иванов И.И."
                        style="padding:11px 14px;border:1px solid var(--border);border-radius:10px;font-family:'Geologica',sans-serif;font-size:15px;background:var(--bg);color:var(--text);width:100%;transition:border-color .15s"
                        onkeydown="if(event.key==='Enter') doAddStudent()"
                        autofocus
                    />
                    <div id="addStudentError" style="font-size:13px;color:#b91c1c;min-height:18px;margin-top:2px"></div>
                </div>
            </div>
            <div class="modal-foot" style="justify-content:space-between">
                <button class="btn-cancel" onclick="closeAddStudentModal()">Отмена</button>
                <button class="btn-primary" id="btnDoAdd" onclick="doAddStudent()">Добавить</button>
            </div>
        </div>
    `;
    overlay.addEventListener('click', e => { if (e.target === overlay) closeAddStudentModal(); });
    document.body.appendChild(overlay);
    setTimeout(() => document.getElementById('newStudentName')?.focus(), 50);
}

function closeAddStudentModal() {
    document.getElementById('addStudentOverlay')?.remove();
}

async function doAddStudent() {
    const input = document.getElementById('newStudentName');
    const errEl = document.getElementById('addStudentError');
    const btn = document.getElementById('btnDoAdd');
    const name = input.value.trim();

    if (!name) { errEl.textContent = 'Введите имя студента'; return; }
    errEl.textContent = '';
    btn.disabled = true;
    btn.textContent = 'Добавление…';

    try {
        const resp = await fetch('/api/students/add', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ full_name: name, group_id: curGroup.id })
        });
        if (!resp.ok) {
            const msg = await resp.text();
            errEl.textContent = msg || 'Ошибка добавления';
            return;
        }
        closeAddStudentModal();
        const studResp = await fetch(`/api/students?group_id=${curGroup.id}`);
        students = await studResp.json() || [];
        renderStudents();
        showToast(`${name} добавлен ✓`);
    } catch (e) {
        errEl.textContent = 'Ошибка соединения';
    } finally {
        btn.disabled = false;
        btn.textContent = 'Добавить';
    }
}

function confirmDeleteStudent(id, name) {
    const existing = document.getElementById('delStudentOverlay');
    if (existing) existing.remove();

    const overlay = document.createElement('div');
    overlay.className = 'overlay on';
    overlay.id = 'delStudentOverlay';
    overlay.innerHTML = `
        <div class="modal-box" style="width:380px">
            <div class="modal-head">
                <div class="m-name">Удалить студента?</div>
                <div class="m-date">${esc(name)}</div>
            </div>
            <div class="modal-body">
                <p style="font-size:14px;color:var(--muted);line-height:1.6">
                    Все оценки этого студента будут удалены вместе с ним.<br>
                    Это действие нельзя отменить.
                </p>
            </div>
            <div class="modal-foot" style="justify-content:space-between">
                <button class="btn-cancel" onclick="closeDelStudentModal()">Отмена</button>
                <button class="btn-danger" id="btnDoDel" onclick="doDeleteStudent(${id}, '${esc(name).replace(/'/g,"&#39;")}')">
                    Удалить
                </button>
            </div>
        </div>
    `;
    overlay.addEventListener('click', e => { if (e.target === overlay) closeDelStudentModal(); });
    document.body.appendChild(overlay);
}

function closeDelStudentModal() {
    document.getElementById('delStudentOverlay')?.remove();
}

async function doDeleteStudent(id, name) {
    const btn = document.getElementById('btnDoDel');
    btn.disabled = true;
    btn.textContent = 'Удаление…';

    try {
        const resp = await fetch('/api/students/delete', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ student_id: id, group_id: curGroup.id })
        });
        if (!resp.ok) {
            const msg = await resp.text();
            alert(msg || 'Ошибка удаления');
            return;
        }
        closeDelStudentModal();
        students = students.filter(s => s.id !== id);
        renderStudents();
        showToast(`${name} удалён`);
    } catch (e) {
        alert('Ошибка соединения');
    } finally {
        btn.disabled = false;
    }
}

function openVedomostGroupModal() {
    const existing = document.getElementById('vedomostGroupOverlay');
    if (existing) existing.remove();

    const now = new Date();
    const curMonth = now.getMonth() + 1;
    const curYear = now.getFullYear();

    const monthOptions = [
        ['1','Январь'],['2','Февраль'],['3','Март'],['4','Апрель'],
        ['5','Май'],['6','Июнь'],['7','Июль'],['8','Август'],
        ['9','Сентябрь'],['10','Октябрь'],['11','Ноябрь'],['12','Декабрь'],
    ].map(([v, l]) => `<option value="${v}"${+v === curMonth ? ' selected' : ''}>${l}</option>`).join('');

    const yearOptions = [2025, 2026]
        .map(y => `<option value="${y}"${y === curYear ? ' selected' : ''}>${y}</option>`).join('');

    const overlay = document.createElement('div');
    overlay.className = 'overlay on';
    overlay.id = 'vedomostGroupOverlay';
    overlay.innerHTML = `
        <div class="modal-box" style="width:340px">
            <div class="modal-head">
                <div class="m-name">Ведомость группы</div>
                <div class="m-date">${esc(curGroup.name)}</div>
            </div>
            <div class="modal-body">
                <div style="display:flex;flex-direction:column;gap:14px">
                    <div style="display:flex;flex-direction:column;gap:6px">
                        <label style="font-size:12.5px;font-weight:600;color:var(--muted);text-transform:uppercase;letter-spacing:.04em">Год</label>
                        <select id="vdgYear" style="padding:10px 14px;border:1px solid var(--border);border-radius:10px;font-family:'Geologica',sans-serif;font-size:15px;background:var(--surface);color:var(--text)">
                            ${yearOptions}
                        </select>
                    </div>
                    <div style="display:flex;flex-direction:column;gap:6px">
                        <label style="font-size:12.5px;font-weight:600;color:var(--muted);text-transform:uppercase;letter-spacing:.04em">Месяц</label>
                        <select id="vdgMonth" style="padding:10px 14px;border:1px solid var(--border);border-radius:10px;font-family:'Geologica',sans-serif;font-size:15px;background:var(--surface);color:var(--text)">
                            ${monthOptions}
                        </select>
                    </div>
                </div>
            </div>
            <div class="modal-foot" style="justify-content:space-between">
                <button class="btn-cancel" onclick="closeVedomostGroupModal()">Отмена</button>
                <button class="btn-primary" onclick="goToVedomostFull()">Открыть предметы →</button>
            </div>
        </div>
    `;
    overlay.addEventListener('click', e => {
        if (e.target === overlay) closeVedomostGroupModal();
    });
    document.body.appendChild(overlay);
}

function closeVedomostGroupModal() {
    document.getElementById('vedomostGroupOverlay')?.remove();
}

function goToVedomostFull() {
    const year = +document.getElementById('vdgYear').value;
    const month = +document.getElementById('vdgMonth').value;
    closeVedomostGroupModal();
    openFullVedomostDialog(curGroup, year, month);
}

function openFullVedomostDialog(group, year, month) {
    fetch(`/api/subjects?group_id=${group.id}`)
        .then(r => r.json())
        .then(subjects => renderFullVedomostModal(group, subjects, year, month))
        .catch(() => alert('Не удалось загрузить предметы'));
}

function renderFullVedomostModal(group, subjects, year, month) {
    document.getElementById('vedomostFullOverlay')?.remove();

    const monthNames = ['','январь','февраль','март','апрель','май','июнь','июль','август','сентябрь','октябрь','ноябрь','декабрь'];

    const subjectCheckboxes = subjects.map(s => `
        <label style="display:flex;align-items:center;gap:10px;padding:8px 12px;background:var(--bg);border:1px solid var(--border);border-radius:9px;cursor:pointer;font-size:14px;font-weight:500;color:var(--text);">
            <input type="checkbox" name="subj" value="${s.id}" checked style="width:16px;height:16px;accent-color:var(--accent);">
            <span>${esc(s.name)}</span>
        </label>
    `).join('');

    const overlay = document.createElement('div');
    overlay.className = 'overlay on';
    overlay.id = 'vedomostFullOverlay';
    overlay.innerHTML = `
        <div class="modal-box" style="width:460px;max-width:calc(100vw - 32px)">
            <div class="modal-head">
                <div class="m-name">Ведомость · ${esc(group.name)}</div>
                <div class="m-date">${monthNames[month]} ${year}</div>
            </div>
            <div class="modal-body" style="max-height:55vh;overflow-y:auto;display:flex;flex-direction:column;gap:16px">
                <div style="display:flex;flex-direction:column;gap:6px">
                    <div style="font-size:11px;font-weight:700;letter-spacing:.08em;text-transform:uppercase;color:var(--muted)">Предметы</div>
                    <div id="vdFullSubjects" style="display:flex;flex-direction:column;gap:6px">${subjectCheckboxes}</div>
                    <div style="display:flex;gap:8px;padding-left:2px;margin-top:4px">
                        <button onclick="vdAllCheck(true)" style="background:none;border:none;font-family:'Geologica',sans-serif;font-size:13px;font-weight:600;color:var(--accent);cursor:pointer">Все</button>
                        <span style="color:var(--border)">·</span>
                        <button onclick="vdAllCheck(false)" style="background:none;border:none;font-family:'Geologica',sans-serif;font-size:13px;font-weight:600;color:var(--accent);cursor:pointer">Снять</button>
                    </div>
                </div>
                <div style="display:flex;flex-direction:column;gap:10px">
                    <div style="font-size:11px;font-weight:700;letter-spacing:.08em;text-transform:uppercase;color:var(--muted)">Параметры</div>
                    <div style="display:flex;flex-direction:column;gap:6px">
                        <label style="font-size:12.5px;font-weight:600;color:var(--muted)">Учебных часов за месяц</label>
                        <input type="number" id="vdFullHours" value="120" min="1" max="500" style="padding:9px 13px;border:1px solid var(--border);border-radius:9px;font-family:'Geologica',sans-serif;font-size:14px;background:var(--surface);color:var(--text);width:100%">
                    </div>
                    <div style="display:flex;flex-direction:column;gap:6px">
                        <label style="font-size:12.5px;font-weight:600;color:var(--muted)">Классный руководитель</label>
                        <input type="text" id="vdFullHead" style="padding:9px 13px;border:1px solid var(--border);border-radius:9px;font-family:'Geologica',sans-serif;font-size:14px;background:var(--surface);color:var(--text);width:100%">
                    </div>
                    <div style="display:flex;flex-direction:column;gap:6px">
                        <label style="font-size:12.5px;font-weight:600;color:var(--muted)">Заведующий отделением</label>
                        <input type="text" id="vdFullDept" style="padding:9px 13px;border:1px solid var(--border);border-radius:9px;font-family:'Geologica',sans-serif;font-size:14px;background:var(--surface);color:var(--text);width:100%">
                    </div>
                </div>
            </div>
            <div class="modal-foot" style="justify-content:space-between">
                <button class="btn-cancel" onclick="closeFullVedomost()">Отмена</button>
                <button class="btn-primary" id="vdFullBtn" onclick="downloadFullVedomost(${group.id},${year},${month})">Скачать .xlsx</button>
            </div>
        </div>
    `;
    overlay.addEventListener('click', e => { if (e.target === overlay) closeFullVedomost(); });
    document.body.appendChild(overlay);
}

window.vdAllCheck = function(v) {
    document.querySelectorAll('#vdFullSubjects input[type=checkbox]').forEach(cb => cb.checked = v);
};

function closeFullVedomost() {
    document.getElementById('vedomostFullOverlay')?.remove();
}

window.downloadFullVedomost = async function(groupId, year, month) {
    const btn = document.getElementById('vdFullBtn');
    btn.disabled = true;
    btn.textContent = 'Генерация…';

    const ids = [...document.querySelectorAll('#vdFullSubjects input[type=checkbox]:checked')].map(cb => +cb.value);
    if (ids.length === 0) {
        alert('Выберите хотя бы один предмет');
        btn.disabled = false;
        btn.textContent = 'Скачать .xlsx';
        return;
    }

    const payload = {
        group_id: groupId, year, month, subject_ids: ids,
        total_hours: +document.getElementById('vdFullHours').value || 120,
        head_teacher: document.getElementById('vdFullHead').value.trim(),
        dept_head: document.getElementById('vdFullDept').value.trim(),
    };

    try {
        const resp = await fetch('/api/vedomost/generate', {
            method: 'POST', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload),
        });
        if (!resp.ok) throw new Error(await resp.text());
        const blob = await resp.blob();
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        const cd = resp.headers.get('Content-Disposition') || '';
        const m = cd.match(/filename="?([^"]+)"?/);
        a.download = m ? m[1] : `vedomost_${year}_${month}.xlsx`;
        a.href = url;
        document.body.appendChild(a); a.click(); a.remove();
        URL.revokeObjectURL(url);
        closeFullVedomost();
    } catch (e) {
        alert('Ошибка: ' + e.message);
    } finally {
        btn.disabled = false;
        btn.textContent = 'Скачать .xlsx';
    }
};

function showToast(text) {
    let t = document.getElementById('groupsToast');
    if (!t) {
        t = document.createElement('div');
        t.id = 'groupsToast';
        t.className = 'groups-toast';
        document.body.appendChild(t);
    }
    t.textContent = text;
    t.classList.add('visible');
    clearTimeout(t._timer);
    t._timer = setTimeout(() => t.classList.remove('visible'), 2500);
}

function esc(s) {
    if (!s) return '';
    return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

init();