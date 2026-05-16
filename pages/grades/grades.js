let ctx = null, modalCtx = null;

function init() {
    const raw = sessionStorage.getItem('gradeCtx');
    if (!raw) { window.location.href = '/'; return; }
    ctx = JSON.parse(raw);
    document.getElementById('bcGroup').textContent = ctx.group.name;
    document.getElementById('bcSubject').textContent = ctx.subject.name;
    loadGrades();
}

async function loadGrades() {
    const year = document.getElementById('yearSel').value;
    const month = document.getElementById('monthSel').value;
    const wrap = document.getElementById('gradesWrap');
    wrap.innerHTML = '<div class="loading"><div class="spinner"></div>Загрузка...</div>';
    try {
        const r = await fetch(`/api/grades?group_id=${ctx.group.id}&subject_id=${ctx.subject.id}&year=${year}&month=${month}`);
        const data = await r.json();
        renderTable(data, year, month);
    } catch (e) {
        wrap.innerHTML = '<div class="loading">Ошибка загрузки</div>';
    }
}

function renderTable(data, year, month) {
    if (!data || data.length === 0) {
        document.getElementById('gradesWrap').innerHTML = '<div class="loading">Нет данных</div>';
        return;
    }

    const days = Object.keys(data[0])
        .filter(k => k.startsWith('day_'))
        .map(k => +k.split('_')[1])
        .sort((a, b) => a - b);

    let totalG = 0, sumG = 0;
    const cnt = { 5: 0, 4: 0, 3: 0, 2: 0 };

    let html = '<div class="table-wrap"><table class="gt"><thead><tr><th>Студент</th>';
    days.forEach(d => html += `<th>${d}</th>`);
    html += '<th>Ср.</th></tr></thead><tbody>';

    data.forEach(st => {
        let sum = 0, c = 0;
        let cells = '';
        days.forEach(d => {
            const cell = st[`day_${d}`];
            const date = `${year}-${String(month).padStart(2, '0')}-${String(d).padStart(2, '0')}`;
            let cls = 'empty', txt = '—';
            if (cell && cell.value && cell.status === 'present') {
                cls = 'grade-cell g' + cell.value; txt = cell.value;
                sum += cell.value; c++;
                totalG++; sumG += cell.value;
                if (cnt[cell.value] !== undefined) cnt[cell.value]++;
            } else if (cell && cell.status === 'absent') {
                cls = 'absent'; txt = 'Н';
            } else if (cell && cell.status === 'excused') {
                cls = 'excused'; txt = 'УВ';
            } else {
                cls = 'empty';
            }
            cells += `<td class="${cls}" onclick="openModal(${st.student_id},'${esc(st.student_name)}','${date}',${d},${month},${year})">${txt}</td>`;
        });
        const avg = c > 0 ? (sum / c).toFixed(2) : '—';
        html += `<tr><td>${esc(st.student_name)}</td>${cells}<td>${avg}</td></tr>`;
    });

    const avgAll = totalG > 0 ? (sumG / totalG).toFixed(2) : '—';
    html += '</tbody></table></div>';
    html += `<div class="stats-row">
        <div class="stat-item"><span class="stat-label">Оценок:</span><span class="stat-value">${totalG}</span></div>
        <div class="stat-item"><span class="stat-label">Средний балл:</span><span class="stat-value">${avgAll}</span></div>
        <div class="stat-item"><span class="stat-label">Пятёрок:</span><span class="stat-value">${cnt[5]}</span></div>
        <div class="stat-item"><span class="stat-label">Четвёрок:</span><span class="stat-value">${cnt[4]}</span></div>
        <div class="stat-item"><span class="stat-label">Троек:</span><span class="stat-value">${cnt[3]}</span></div>
        <div class="stat-item"><span class="stat-label">Двоек:</span><span class="stat-value">${cnt[2]}</span></div>
    </div>`;

    document.getElementById('gradesWrap').innerHTML = html;
}

function openModal(sid, name, date, d, m, y) {
    modalCtx = { sid, date };
    document.getElementById('mName').textContent = name;
    document.getElementById('mDate').textContent = `${d}.${String(m).padStart(2, '0')}.${y}`;
    document.getElementById('overlay').classList.add('on');
}

function closeModal() {
    document.getElementById('overlay').classList.remove('on');
    modalCtx = null;
}

function overlayClick(e) {
    if (e.target === document.getElementById('overlay')) closeModal();
}

async function saveGrade(g) {
    if (!modalCtx) return;
    await apiSet({ student_id: modalCtx.sid, subject_id: ctx.subject.id, date: modalCtx.date, grade: g, status: 'present' });
    closeModal(); loadGrades(); msg('Оценка сохранена ✓');
}

async function saveAbsent(status) {
    if (!modalCtx) return;
    await apiSet({ student_id: modalCtx.sid, subject_id: ctx.subject.id, date: modalCtx.date, grade: null, status });
    closeModal(); loadGrades(); msg(status === 'absent' ? 'Отмечено: Н ✓' : 'Отмечено: УВ ✓');
}

async function clearGrade() {
    if (!modalCtx) return;
    await apiSet({ student_id: modalCtx.sid, subject_id: ctx.subject.id, date: modalCtx.date, grade: null, status: '' });
    closeModal(); loadGrades(); msg('Очищено ✓');
}

async function apiSet(body) {
    try {
        await fetch('/api/set-grade', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
    } catch (e) { msg('❌ Ошибка сохранения'); }
}

let msgTimer;
function msg(text) {
    const el = document.getElementById('saveMsg');
    el.textContent = text;
    clearTimeout(msgTimer);
    msgTimer = setTimeout(() => el.textContent = '', 2500);
}

function esc(s) {
    if (!s) return '';
    return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

init();