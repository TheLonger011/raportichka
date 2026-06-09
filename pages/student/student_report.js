(function () {
    function injectReportButton() {
        const controls = document.querySelector('.grades-controls');
        if (!controls || document.getElementById('btnStudentReport')) return;

        const btn = document.createElement('button');
        btn.id = 'btnStudentReport';
        btn.className = 'btn-report';
        btn.innerHTML = `
            <svg viewBox="0 0 24 24">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
                <polyline points="14 2 14 8 20 8"/>
                <line x1="16" y1="13" x2="8" y2="13"/>
                <line x1="16" y1="17" x2="8" y2="17"/>
                <polyline points="10 9 9 9 8 9"/>
            </svg>
            Скачать отчёт
        `;
        btn.onclick = openReportModal;
        controls.appendChild(btn);
    }

    function openReportModal() {
        document.getElementById('studentReportOverlay')?.remove();

        const yearEl  = document.getElementById('yearSel');
        const monthEl = document.getElementById('monthSel');
        const year    = yearEl  ? +yearEl.value  : new Date().getFullYear();
        const month   = monthEl ? +monthEl.value : new Date().getMonth() + 1;
        const monthName = monthEl ? monthEl.options[monthEl.selectedIndex]?.text : '';

        const overlay = document.createElement('div');
        overlay.className = 'overlay on';
        overlay.id = 'studentReportOverlay';
        overlay.innerHTML = `
            <div class="modal-box" style="width:400px">
                <div class="modal-head">
                    <div class="m-name">Отчёт об успеваемости</div>
                    <div class="m-date">${monthName} ${year}</div>
                </div>
                <div class="modal-body" style="display:flex;flex-direction:column;gap:14px">
                    <p style="font-size:14px;color:var(--muted);line-height:1.7">
                        Будет сформирован PDF-отчёт с вашими оценками и посещаемостью за выбранный месяц.
                    </p>
                    <div style="display:flex;flex-direction:column;gap:8px">
                        <div class="report-info-row">
                            <span class="report-info-icon">
                                <svg viewBox="0 0 24 24"><rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
                            </span>
                            <span>Период: <strong>${monthName} ${year}</strong></span>
                        </div>
                        <div class="report-info-row">
                            <span class="report-info-icon">
                                <svg viewBox="0 0 24 24"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
                            </span>
                            <span>Включает все предметы группы</span>
                        </div>
                    </div>
                </div>
                <div class="modal-foot" style="justify-content:space-between">
                    <button class="btn-cancel" onclick="closeReportModal()">Отмена</button>
                    <button class="btn-load" id="btnDoReport" onclick="generateStudentReport(${year},${month})">
                        <svg viewBox="0 0 24 24" style="width:15px;height:15px;stroke:currentColor;fill:none;stroke-width:2;stroke-linecap:round;stroke-linejoin:round;margin-right:6px;vertical-align:middle">
                            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                            <polyline points="7 10 12 15 17 10"/>
                            <line x1="12" y1="15" x2="12" y2="3"/>
                        </svg>
                        Скачать PDF
                    </button>
                </div>
            </div>
        `;
        overlay.addEventListener('click', e => { if (e.target === overlay) closeReportModal(); });
        document.body.appendChild(overlay);
    }

    window.closeReportModal = function () {
        document.getElementById('studentReportOverlay')?.remove();
    };

    window.generateStudentReport = async function (year, month) {
        const btn = document.getElementById('btnDoReport');
        btn.disabled = true;
        btn.innerHTML = `<span style="display:inline-flex;align-items:center;gap:6px">
            <span style="width:14px;height:14px;border:2px solid rgba(255,255,255,.3);border-top-color:#fff;border-radius:50%;display:inline-block;animation:spin .7s linear infinite"></span>
            Генерация…
        </span>`;

        try {
            const [gradesResp, meResp] = await Promise.all([
                fetch(`/api/student/grades?year=${year}&month=${month}`),
                fetch('/api/me')
            ]);
            if (!gradesResp.ok) throw new Error('Ошибка загрузки оценок');
            const data = await gradesResp.json();
            const me   = meResp.ok ? await meResp.json() : {};

            closeReportModal();
            printReport(me, data, year, month);
        } catch (e) {
            alert('Ошибка: ' + e.message);
            btn.disabled = false;
            btn.innerHTML = 'Скачать PDF';
        }
    };

    function printReport(me, subjects, year, month) {
        const monthNames = ['','Январь','Февраль','Март','Апрель','Май','Июнь',
            'Июль','Август','Сентябрь','Октябрь','Ноябрь','Декабрь'];

        const now     = new Date();
        const dateStr = now.toLocaleDateString('ru-RU', { day:'2-digit', month:'long', year:'numeric' });

        let totalGrades = 0, sumAll = 0, cnt5 = 0, cnt4 = 0, cnt3 = 0, cnt2 = 0;
        (subjects || []).forEach(subj => {
            (subj.grades || []).forEach(g => {
                if (g.value != null) {
                    totalGrades++;
                    sumAll += g.value;
                    if (g.value === 5) cnt5++;
                    else if (g.value === 4) cnt4++;
                    else if (g.value === 3) cnt3++;
                    else if (g.value === 2) cnt2++;
                }
            });
        });
        const overallAvg = totalGrades > 0 ? (sumAll / totalGrades).toFixed(2) : '—';

        const rows = (subjects || []).map(subj => {
            const grades        = (subj.grades || []).filter(g => g.value != null);
            const absTotal      = subj.abs_total    || 0;
            const absExcused    = subj.abs_excused  || 0;
            const absUnexcused  = absTotal - absExcused;
            const avg           = subj.avg != null ? (+subj.avg).toFixed(2) : '—';

            const gradesList = grades.map(g => {
                const cls = g.value >= 5 ? 'g5' : g.value >= 4 ? 'g4' : g.value >= 3 ? 'g3' : 'g2';
                return `<span class="grade-pill ${cls}">${g.value}</span>`;
            }).join('') || '<span style="color:#aaa">нет</span>';

            return `
                <tr>
                    <td class="subj-name">${esc(subj.subject_name)}</td>
                    <td class="grades-cell">${gradesList}</td>
                    <td class="num-cell avg-val">${avg}</td>
                    <td class="num-cell">${absTotal}</td>
                    <td class="num-cell">${absExcused}</td>
                    <td class="num-cell ${absUnexcused > 0 ? 'bad' : ''}">${absUnexcused}</td>
                </tr>
            `;
        }).join('');

        const html = `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8"/>
<title>Отчёт — ${esc(me.full_name || 'Студент')}</title>
<style>
@import url('https://fonts.googleapis.com/css2?family=Geologica:wght@300;400;500;600&family=Unbounded:wght@700&display=swap');
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:'Geologica',sans-serif;color:#1a1d27;background:#fff;padding:32px 36px;font-size:13px}
.report-header{display:flex;align-items:flex-start;justify-content:space-between;margin-bottom:28px;padding-bottom:20px;border-bottom:2px solid #e4e8ef}
.report-brand{font-family:'Unbounded',sans-serif;font-size:16px;font-weight:700;color:#1a1d27;letter-spacing:-.01em}
.report-brand em{color:#3d6ce7;font-style:normal}
.report-title{font-size:18px;font-weight:700;color:#1a1d27;margin-bottom:4px}
.report-meta{font-size:12px;color:#8490a4;line-height:1.8}
.report-meta strong{color:#1a1d27}
.summary-row{display:flex;gap:14px;margin-bottom:24px;flex-wrap:wrap}
.sum-card{background:#f0f2f7;border-radius:10px;padding:12px 18px;flex:1;min-width:90px;text-align:center}
.sum-val{font-family:'Unbounded',sans-serif;font-size:20px;font-weight:700;color:#1a1d27}
.sum-val.c5{color:#15803d}.sum-val.c4{color:#1d4ed8}.sum-val.c3{color:#b45309}.sum-val.c2{color:#b91c1c}
.sum-lbl{font-size:11px;color:#8490a4;margin-top:3px}
table{width:100%;border-collapse:collapse;margin-bottom:24px}
thead th{background:#f0f2f7;padding:10px 12px;text-align:left;font-size:11px;font-weight:700;letter-spacing:.05em;text-transform:uppercase;color:#8490a4;border-bottom:2px solid #e4e8ef}
thead th.num-h{text-align:center}
tbody tr{border-bottom:1px solid #f0f2f7}
tbody tr:last-child{border-bottom:none}
td{padding:10px 12px;vertical-align:middle}
.subj-name{font-weight:600;color:#1a1d27;font-size:13px;max-width:200px}
.grades-cell{display:flex;flex-wrap:wrap;gap:4px;align-items:center}
.grade-pill{display:inline-flex;align-items:center;justify-content:center;width:24px;height:24px;border-radius:6px;font-size:12px;font-weight:700}
.g5{background:#dcfce7;color:#15803d}.g4{background:#dbeafe;color:#1d4ed8}.g3{background:#fef3c7;color:#b45309}.g2{background:#fee2e2;color:#b91c1c}
.num-cell{text-align:center;font-weight:600;font-size:13px;color:#1a1d27}
.avg-val{color:#3d6ce7;font-weight:700}
.bad{color:#b91c1c}
.report-footer{margin-top:28px;padding-top:16px;border-top:1px solid #e4e8ef;display:flex;justify-content:space-between;font-size:11px;color:#8490a4}
@media print{body{padding:16px 20px}.no-print{display:none}}
</style>
</head>
<body>
<div class="report-header">
    <div>
        <div class="report-brand"><em>журнал</em></div>
        <div style="font-size:11px;color:#8490a4;margin-top:4px">Электронный журнал успеваемости</div>
    </div>
    <div style="text-align:right">
        <div class="report-title">Отчёт об успеваемости</div>
        <div class="report-meta">
            <strong>${esc(me.full_name || 'Студент')}</strong><br>
            Группа: <strong>${esc(me.group_name || '—')}</strong><br>
            Период: <strong>${monthNames[month]} ${year}</strong><br>
            Сформирован: ${dateStr}
        </div>
    </div>
</div>

<div class="summary-row">
    <div class="sum-card"><div class="sum-val">${totalGrades}</div><div class="sum-lbl">Всего оценок</div></div>
    <div class="sum-card"><div class="sum-val">${overallAvg}</div><div class="sum-lbl">Средний балл</div></div>
    <div class="sum-card"><div class="sum-val c5">${cnt5}</div><div class="sum-lbl">Пятёрок</div></div>
    <div class="sum-card"><div class="sum-val c4">${cnt4}</div><div class="sum-lbl">Четвёрок</div></div>
    <div class="sum-card"><div class="sum-val c3">${cnt3}</div><div class="sum-lbl">Троек</div></div>
    <div class="sum-card"><div class="sum-val c2">${cnt2}</div><div class="sum-lbl">Двоек</div></div>
</div>

<table>
    <thead>
        <tr>
            <th>Предмет</th>
            <th>Оценки</th>
            <th class="num-h">Средний</th>
            <th class="num-h">Пропусков</th>
            <th class="num-h">Уваж.</th>
            <th class="num-h">Неуваж.</th>
        </tr>
    </thead>
    <tbody>
        ${rows || '<tr><td colspan="6" style="text-align:center;color:#8490a4;padding:24px">Данных нет</td></tr>'}
    </tbody>
</table>

<div class="report-footer">
    <span>Документ сформирован автоматически</span>
    <span>${dateStr}</span>
</div>

<script>window.onload=function(){window.print()}<\/script>
</body>
</html>`;

        const win = window.open('', '_blank');
        if (!win) { alert('Разрешите открытие всплывающих окон в браузере'); return; }
        win.document.write(html);
        win.document.close();
    }

    function esc(s) {
        return (s || '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
    }

    const observer = new MutationObserver(() => {
        if (document.querySelector('.grades-controls') && !document.getElementById('btnStudentReport')) {
            injectReportButton();
        }
    });
    observer.observe(document.body, { childList: true, subtree: true });

    document.addEventListener('DOMContentLoaded', () => {
        if (document.querySelector('.grades-controls')) injectReportButton();
    });

})();