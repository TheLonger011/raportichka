(function () {

    const DEFAULT_TOTAL_HOURS = 120;
    const DEFAULT_HEAD_TEACHER = '';
    const DEFAULT_DEPT_HEAD = '';


    function injectButton() {
        const navRight = document.querySelector('.nav-right');
        if (!navRight) return;

        const btn = document.createElement('button');
        btn.className = 'btn-vedomost';
        btn.id = 'btnVedomost';
        btn.innerHTML = `
            <svg viewBox="0 0 24 24">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
                <polyline points="14 2 14 8 20 8"/>
                <line x1="16" y1="13" x2="8" y2="13"/>
                <line x1="16" y1="17" x2="8" y2="17"/>
                <polyline points="10 9 9 9 8 9"/>
            </svg>
            Ведомость
        `;
        btn.onclick = openVedomostModal;
        const themeBtn = document.getElementById('themeBtn');
        navRight.insertBefore(btn, themeBtn);
    }


    function openVedomostModal() {
        const raw = sessionStorage.getItem('gradeCtx');
        if (!raw) { alert('Контекст группы не найден'); return; }
        const ctx = JSON.parse(raw);

        const year = +document.getElementById('yearSel').value;
        const month = +document.getElementById('monthSel').value;

        fetch(`/api/subjects?group_id=${ctx.group.id}`)
            .then(r => r.json())
            .then(subjects => renderModal(ctx, subjects, year, month))
            .catch(() => alert('Не удалось загрузить предметы'));
    }

    function renderModal(ctx, subjects, year, month) {
        const existing = document.getElementById('vedomostOverlay');
        if (existing) existing.remove();

        const monthNames = ['', 'январь', 'февраль', 'март', 'апрель', 'май', 'июнь', 'сентябрь', 'октябрь', 'ноябрь', 'декабрь'];

        const subjectCheckboxes = subjects.map(s => `
            <label class="vd-checkbox-row">
                <input type="checkbox" name="subj" value="${s.id}" checked>
                <span>${escHtml(s.name)}</span>
            </label>
        `).join('');

        const overlay = document.createElement('div');
        overlay.className = 'overlay on';
        overlay.id = 'vedomostOverlay';
        overlay.innerHTML = `
            <div class="modal-box vd-modal">
                <div class="modal-head">
                    <div class="m-name">Генерация ведомости</div>
                    <div class="m-date">${ctx.group.name} · ${monthNames[month]} ${year}</div>
                </div>
                <div class="modal-body vd-body">
                    <div class="vd-section">
                        <div class="vd-section-title">Предметы</div>
                        <div class="vd-subjects" id="vdSubjects">
                            ${subjectCheckboxes}
                        </div>
                        <div class="vd-subject-actions">
                            <button class="vd-link" onclick="vedomostSelectAll(true)">Все</button>
                            <span class="vd-sep">·</span>
                            <button class="vd-link" onclick="vedomostSelectAll(false)">Снять</button>
                        </div>
                    </div>
                    <div class="vd-section">
                        <div class="vd-section-title">Параметры</div>
                        <div class="vd-fields">
                            <div class="vd-field">
                                <label>Учебных часов за месяц</label>
                                <input type="number" id="vdHours" value="${DEFAULT_TOTAL_HOURS}" min="1" max="500">
                            </div>
                            <div class="vd-field">
                                <label>Классный руководитель</label>
                                <input type="text" id="vdHeadTeacher" value="${DEFAULT_HEAD_TEACHER}">
                            </div>
                            <div class="vd-field">
                                <label>Заведующий отделением</label>
                                <input type="text" id="vdDeptHead" value="${DEFAULT_DEPT_HEAD}">                        </div>
                    </div>
                </div>
                <div class="modal-foot" style="justify-content: space-between;">
                    <button class="btn-cancel" onclick="closeVedomostModal()">Отмена</button>
                    <button class="btn-load" id="vdGenBtn" onclick="generateVedomost(${ctx.group.id},${year},${month})">
                        Скачать .xlsx
                    </button>
                </div>
            </div>
        `;
        overlay.addEventListener('click', e => {
            if (e.target === overlay) closeVedomostModal();
        });
        document.body.appendChild(overlay);
    }

    window.closeVedomostModal = function () {
        const el = document.getElementById('vedomostOverlay');
        if (el) el.remove();
    };

    window.vedomostSelectAll = function (checked) {
        document.querySelectorAll('#vdSubjects input[type=checkbox]')
            .forEach(cb => cb.checked = checked);
    };

    window.generateVedomost = async function (groupId, year, month) {
        const btn = document.getElementById('vdGenBtn');
        btn.disabled = true;
        btn.textContent = 'Генерация…';

        const checkedBoxes = [...document.querySelectorAll('#vdSubjects input[type=checkbox]:checked')];
        const subjectIds = checkedBoxes.map(cb => +cb.value);

        if (subjectIds.length === 0) {
            alert('Выберите хотя бы один предмет');
            btn.disabled = false;
            btn.textContent = 'Скачать .xlsx';
            return;
        }

        const payload = {
            group_id:     groupId,
            year:         year,
            month:        month,
            subject_ids:  subjectIds,
            total_hours:  +document.getElementById('vdHours').value || 120,
            head_teacher: document.getElementById('vdHeadTeacher').value.trim(),
            dept_head:    document.getElementById('vdDeptHead').value.trim(),
        };

        try {
            const resp = await fetch('/api/vedomost/generate', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload),
            });
            if (!resp.ok) {
                const msg = await resp.text();
                throw new Error(msg || resp.statusText);
            }
            const blob = await resp.blob();
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            const cd = resp.headers.get('Content-Disposition') || '';
            const match = cd.match(/filename="?([^"]+)"?/);
            a.download = match ? match[1] : `vedomost_${year}_${month}.xlsx`;
            a.href = url;
            document.body.appendChild(a);
            a.click();
            a.remove();
            URL.revokeObjectURL(url);
            closeVedomostModal();
        } catch (e) {
            alert('Ошибка генерации: ' + e.message);
        } finally {
            btn.disabled = false;
            btn.textContent = 'Скачать .xlsx';
        }
    };

    function escHtml(s) {
        return (s || '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
    }

    document.addEventListener('DOMContentLoaded', injectButton);
})();