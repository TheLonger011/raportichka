let currentContext = null;

async function loadGrades() {
    const year = document.getElementById('yearSelect').value;
    const month = document.getElementById('monthSelect').value;
    const contentDiv = document.getElementById('content');

    contentDiv.innerHTML = `
        <div class="loading">
            <div class="spinner"></div>
            <p>Загрузка...</p>
        </div>
    `;

    try {
        const response = await fetch(`/api/grades?year=${year}&month=${month}`);
        if (!response.ok) throw new Error('Ошибка загрузки');
        const data = await response.json();
        displayGrades(data);
    } catch (error) {
        contentDiv.innerHTML = `<div class="error-message">❌ ${error.message}</div>`;
    }
}

function displayGrades(data) {
    if (!data || data.length === 0) {
        document.getElementById('content').innerHTML = `<div class="error-message">Нет данных</div>`;
        return;
    }

    const days = Object.keys(data[0])
        .filter(k => k.startsWith('day_'))
        .map(k => parseInt(k.split('_')[1]))
        .sort((a, b) => a - b);

    let totalGrades = 0, sumGrades = 0;
    const gradeCounts = {5: 0, 4: 0, 3: 0, 2: 0};

    data.forEach(student => {
        days.forEach(day => {
            const cell = student[`day_${day}`];
            if (cell && cell.value && cell.status === 'present') {
                totalGrades++;
                sumGrades += cell.value;
                if (gradeCounts[cell.value] !== undefined) gradeCounts[cell.value]++;
            }
        });
    });

    const avgGrade = totalGrades > 0 ? (sumGrades / totalGrades).toFixed(2) : '—';

    let html = `<div class="table-wrapper"><table class="grades-table"><thead><tr>
        <th>Студент</th>${days.map(d => `<th>${d}</th>`).join('')}<th>Ср.</th>
    </td></thead><tbody>`;

    data.forEach(student => {
        let sSum = 0, sCnt = 0;
        html += `<tr data-student-id="${student.student_id}">`;
        html += `<td class="student-name">${escapeHtml(student.student_name)}</td>`;

        days.forEach(day => {
            const cell = student[`day_${day}`];
            let displayText = '';
            let displayClass = 'grade-cell grade-empty';

            if (cell && cell.value && cell.status === 'present') {
                displayText = cell.value;
                displayClass = `grade-cell grade-${cell.value}`;
                sSum += cell.value;
                sCnt++;
            } else if (cell && cell.status === 'absent') {
                displayText = 'Н';
                displayClass = 'grade-cell grade-absent';
            } else if (cell && cell.status === 'excused') {
                displayText = 'УВ';
                displayClass = 'grade-cell grade-excused';
            } else {
                displayText = '—';
                displayClass = 'grade-cell grade-empty';
            }

            html += `<td class="${displayClass}" onclick="openModal(${student.student_id}, '${escapeHtml(student.student_name)}', ${day})">${displayText}</td>`;
        });

        const sAvg = sCnt > 0 ? (sSum / sCnt).toFixed(2) : '—';
        html += `<td class="grade-cell" style="font-weight:500">${sAvg}</td>`;
        html += `</tr>`;
    });

    html += `<tr style="background:#f9f9f9"><td class="student-name" style="font-weight:500">Итого</td>
        ${days.map(() => '<td></td>').join('')}
        <td class="grade-cell" style="font-weight:600">${avgGrade}</td>
      </tr>`;

    html += `</tbody></table></div>
        <div class="stats">
            <div class="stat-item"><span class="stat-label">Всего</span><span class="stat-value">${totalGrades}</span></div>
            <div class="stat-item"><span class="stat-label">Ср.балл</span><span class="stat-value">${avgGrade}</span></div>
            <div class="stat-item"><span class="stat-label">5 -</span><span class="stat-value">${gradeCounts[5]}</span></div>
            <div class="stat-item"><span class="stat-label">4 -</span><span class="stat-value">${gradeCounts[4]}</span></div>
            <div class="stat-item"><span class="stat-label">3 -</span><span class="stat-value">${gradeCounts[3]}</span></div>
            <div class="stat-item"><span class="stat-label">2 -</span><span class="stat-value">${gradeCounts[2]}</span></div>
        </div>`;

    document.getElementById('content').innerHTML = html;
}

function escapeHtml(str) {
    if (!str) return '';
    return str
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

function openModal(studentId, studentName, day) {
    const year = document.getElementById('yearSelect').value;
    const month = document.getElementById('monthSelect').value;
    const date = `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;

    currentContext = {
        studentId: studentId,
        date: date,
        day: day
    };

    document.getElementById('modalStudentName').innerText = studentName;
    document.getElementById('modalDate').innerText = `${day}.${month}.${year}`;
    document.getElementById('gradeModal').style.display = 'block';
}

async function setGrade(grade) {
    if (!currentContext) return;

    await saveGrade(currentContext.studentId, currentContext.date, grade, null);
    closeModal();
    await loadGrades();
    showSaveStatus('Оценка сохранена');
}

async function setAbsent(status) {
    if (!currentContext) return;

    await saveGrade(currentContext.studentId, currentContext.date, null, status);
    closeModal();
    await loadGrades();
    showSaveStatus(status === 'absent' ? 'Отмечен как отсутствие' : 'Отмечена уважительная причина');
}

async function clearGrade() {
    if (!currentContext) return;

    await saveGrade(currentContext.studentId, currentContext.date, null, '');
    closeModal();
    await loadGrades();
    showSaveStatus('Оценка очищена');
}

async function saveGrade(studentId, date, grade, status) {
    try {
        const response = await fetch('/api/set-grade', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                student_id: studentId,
                date: date,
                grade: grade,
                status: status
            })
        });

        if (!response.ok) {
            throw new Error('Ошибка сохранения');
        }
    } catch (error) {
        console.error('Save error:', error);
        showSaveStatus('❌ Ошибка сохранения');
    }
}

function closeModal() {
    document.getElementById('gradeModal').style.display = 'none';
    currentContext = null;
}

function showSaveStatus(message) {
    const statusDiv = document.getElementById('saveStatus');
    statusDiv.innerText = message;
    setTimeout(() => {
        statusDiv.innerText = '';
    }, 2000);
}

window.onclick = function(event) {
    const modal = document.getElementById('gradeModal');
    if (event.target === modal) {
        closeModal();
    }
}

window.onload = loadGrades;