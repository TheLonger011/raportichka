let ctx = null, modalCtx = null;
let savedScrollPosition = { top: 0, left: 0 };

function init() {
    const raw = sessionStorage.getItem('gradeCtx');
    if (!raw) { window.location.href = '/'; return; }
    ctx = JSON.parse(raw);
    document.getElementById('bcGroup').textContent = ctx.group.name;
    document.getElementById('bcSubject').textContent = ctx.subject.name;
    loadGrades();
}

function saveScrollPosition() {
    const wrap = document.querySelector('.table-wrap');
    if (wrap) {
        savedScrollPosition = { top: wrap.scrollTop, left: wrap.scrollLeft };
    }
}

function restoreScrollPosition() {
    requestAnimationFrame(() => {
        const wrap = document.querySelector('.table-wrap');
        if (wrap) {
            wrap.scrollTop = savedScrollPosition.top;
            wrap.scrollLeft = savedScrollPosition.left;
        }
    });
}

async function loadGrades(preserveScroll = true) {
    const year  = document.getElementById('yearSel').value;
    const month = document.getElementById('monthSel').value;
    const wrap  = document.getElementById('gradesWrap');

    if (preserveScroll) {
        const oldWrap = document.querySelector('.table-wrap');
        if (oldWrap) {
            savedScrollPosition = { top: oldWrap.scrollTop, left: oldWrap.scrollLeft };
        }
    }

    wrap.innerHTML = '<div class="loading"><div class="spinner"></div>Загрузка...</div>';

    try {
        const r    = await fetch(`/api/grades?group_id=${ctx.group.id}&subject_id=${ctx.subject.id}&year=${year}&month=${month}`);
        const data = await r.json();
        renderTable(data, year, month, preserveScroll);
    } catch (e) {
        wrap.innerHTML = '<div class="loading">Ошибка загрузки</div>';
    }
}

function renderTable(data, year, month, preserveScroll = true) {
    if (!data || data.length === 0) {
        document.getElementById('gradesWrap').innerHTML = '<div class="loading">Нет данных</div>';
        return;
    }

    function getCells(row) {
        if (row.days && typeof row.days === 'object') return row.days;
        const obj = {};
        for (const k of Object.keys(row)) {
            if (k.startsWith('day_')) obj[k] = row[k];
        }
        return obj;
    }

    const firstCells = getCells(data[0]);
    const days = Object.keys(firstCells)
        .filter(k => k.startsWith('day_'))
        .map(k => +k.split('_')[1])
        .sort((a, b) => a - b);

    let totalG = 0, sumG = 0;
    const cnt = { 5: 0, 4: 0, 3: 0, 2: 0 };

    let html = '<div class="table-wrap"><table class="gt"><thead><tr>';
    html += '<th>Студент</th>';
    days.forEach(d => html += `<th>${d}</th>`);
    html += '<th>Ср.</th>';
    html += '</tr></thead><tbody>';

    for (const st of data) {
        const cells = getCells(st);
        let sum = 0, c = 0;
        html += '<tr>';
        html += `<td class="student-name">${esc(st.student_name)}</td>`;

        for (const d of days) {
            const cell = cells[`day_${d}`];
            const date = `${year}-${String(month).padStart(2, '0')}-${String(d).padStart(2, '0')}`;
            const sid  = st.student_id;
            const sname = esc(st.student_name);

            if (cell && cell.value != null && cell.status === 'present') {
                const val = cell.value;
                html += `<td class="grade-cell g${val}" onclick="openModal(${sid},'${sname}','${date}',${d},${month},${year})">${val}</td>`;
                sum += val; c++; totalG++; sumG += val;
                if (cnt[val] !== undefined) cnt[val]++;
            } else if (cell && cell.status === 'absent') {
                html += `<td class="absent" onclick="openModal(${sid},'${sname}','${date}',${d},${month},${year})">Н</td>`;
            } else if (cell && cell.status === 'excused') {
                html += `<td class="excused" onclick="openModal(${sid},'${sname}','${date}',${d},${month},${year})">УВ</td>`;
            } else {
                html += `<td class="empty" onclick="openModal(${sid},'${sname}','${date}',${d},${month},${year})">—</td>`;
            }
        }

        const avg = c > 0 ? (sum / c).toFixed(2) : '—';
        html += `<td class="avg-cell">${avg}</td>`;
        html += '</tr>';
    }

    html += '</tbody></table></div>';

    const avgAll = totalG > 0 ? (sumG / totalG).toFixed(2) : '—';
    html += `<div class="stats-row">
        <div class="stat-item"><span class="stat-label">Оценок:</span><span class="stat-value">${totalG}</span></div>
        <div class="stat-item"><span class="stat-label">Средний балл:</span><span class="stat-value">${avgAll}</span></div>
        <div class="stat-item"><span class="stat-label">Пятёрок:</span><span class="stat-value">${cnt[5]}</span></div>
        <div class="stat-item"><span class="stat-label">Четвёрок:</span><span class="stat-value">${cnt[4]}</span></div>
        <div class="stat-item"><span class="stat-label">Троек:</span><span class="stat-value">${cnt[3]}</span></div>
        <div class="stat-item"><span class="stat-label">Двоек:</span><span class="stat-value">${cnt[2]}</span></div>
    </div>`;

    document.getElementById('gradesWrap').innerHTML = html;

    if (preserveScroll && (savedScrollPosition.top > 0 || savedScrollPosition.left > 0)) {
        setTimeout(() => restoreScrollPosition(), 10);
    }
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
    saveScrollPosition();
    await apiSet({ student_id: modalCtx.sid, subject_id: ctx.subject.id, date: modalCtx.date, grade: g, status: 'present' });
    closeModal();
    await loadGrades(true);
    msg('Оценка сохранена ✓');
}

async function saveAbsent(status) {
    if (!modalCtx) return;
    saveScrollPosition();
    await apiSet({ student_id: modalCtx.sid, subject_id: ctx.subject.id, date: modalCtx.date, grade: null, status });
    closeModal();
    await loadGrades(true);
    msg(status === 'absent' ? 'Отмечено: Н ✓' : 'Отмечено: УВ ✓');
}

async function clearGrade() {
    if (!modalCtx) return;
    saveScrollPosition();
    await apiSet({ student_id: modalCtx.sid, subject_id: ctx.subject.id, date: modalCtx.date, grade: null, status: '' });
    closeModal();
    await loadGrades(true);
    msg('Очищено ✓');
}

async function apiSet(body) {
    try {
        await fetch('/api/set-grade', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        });
    } catch (e) {
        msg('❌ Ошибка сохранения');
    }
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
    return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/'/g, '&#39;');
}

init();