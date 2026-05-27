import sys, json, copy, shutil
from openpyxl import load_workbook
from openpyxl.utils import get_column_letter

SUBJECT_COLS = ['C', 'D', 'E', 'F', 'G', 'H']
STUDENT_START_ROW = 12
MAX_STUDENTS = 24

def col_index(letter):
    from openpyxl.utils import column_index_from_string
    return column_index_from_string(letter)

def main():
    data = json.load(sys.stdin)

    template_path = data['template_path']
    output_path   = data['output_path']

    shutil.copy(template_path, output_path)
    wb = load_workbook(output_path)
    ws = wb.active

    subjects   = data['subjects'][:6]
    students   = data['students']
    total_hours= data.get('total_hours', 120)
    head_teacher = data.get('head_teacher', '').strip()
    dept_head    = data.get('dept_head', '').strip()
    group_name   = data.get('group_name', '')
    month_label  = data.get('month_label', '')

    n_subjects = len(subjects)
    n_students = len(students)

    ws['B5'] = f'текущей аттестации за {month_label}'

    ws['B6'] = f'специальность                                                                                                                                                 Курс   Группа {group_name}\n'

    for i, subj in enumerate(subjects):
        ws.cell(row=11, column=col_index(SUBJECT_COLS[i])).value = subj['name']

    for i in range(n_subjects, 6):
        ws.cell(row=11, column=col_index(SUBJECT_COLS[i])).value = None

    subj_id_to_idx = {s['id']: i for i, s in enumerate(subjects)}

    template_last_row = STUDENT_START_ROW + MAX_STUDENTS - 1  # 35

    for row_idx in range(STUDENT_START_ROW, template_last_row + 1):
        ws.cell(row=row_idx, column=col_index('A')).value = None
        ws.cell(row=row_idx, column=col_index('B')).value = None
        for sc in SUBJECT_COLS:
            ws.cell(row=row_idx, column=col_index(sc)).value = None
        ws.cell(row=row_idx, column=col_index('M')).value = None
        ws.cell(row=row_idx, column=col_index('N')).value = None
        ws.cell(row=row_idx, column=col_index('O')).value = None

    for i, st in enumerate(students[:MAX_STUDENTS]):
        row = STUDENT_START_ROW + i
        ws.cell(row=row, column=col_index('A')).value = i + 1
        ws.cell(row=row, column=col_index('B')).value = st['full_name']

        grades_by_subj = st.get('grades', {})
        for sc in SUBJECT_COLS[:n_subjects]:
            cidx = col_index(sc)
            ws.cell(row=row, column=cidx).value = None

        for subj_id_str, grade in grades_by_subj.items():
            subj_id = int(subj_id_str)
            col_offset = subj_id_to_idx.get(subj_id)
            if col_offset is None:
                continue
            sc = SUBJECT_COLS[col_offset]
            ws.cell(row=row, column=col_index(sc)).value = grade

        ws.cell(row=row, column=col_index('M')).value = st.get('absent_total', 0)
        ws.cell(row=row, column=col_index('N')).value = st.get('absent_excused', 0)
        ws.cell(row=row, column=col_index('O')).value = st.get('absent_unexcused', 0)

    ws['C46'] = n_students
    ws['C47'] = total_hours

    pad = ' ' * 70
    ws['B48'] = f'Классный руководитель{pad}{head_teacher}'
    ws['B49'] = f'Заведующий отделением{pad}{dept_head}'

    last_subj_col = SUBJECT_COLS[n_subjects - 1] if n_subjects > 0 else 'C'

    grade_range = f'C{{r}}:{last_subj_col}{{r}}'
    for i in range(n_students):
        row = STUDENT_START_ROW + i
        r = row
        ws.cell(row=r, column=col_index('I')).value = f'=COUNTIF({grade_range.format(r=r)},5)'
        ws.cell(row=r, column=col_index('J')).value = f'=COUNTIF({grade_range.format(r=r)},4)'
        ws.cell(row=r, column=col_index('K')).value = f'=COUNTIF({grade_range.format(r=r)},3)'
        ws.cell(row=r, column=col_index('L')).value = f'=COUNTIF({grade_range.format(r=r)},2)'
        ws.cell(row=r, column=col_index('P')).value = f'=IFERROR(AVERAGE({grade_range.format(r=r)}),"")'

    actual_last = STUDENT_START_ROW + n_students - 1
    student_range = f'{STUDENT_START_ROW}:{actual_last}'

    ws['I36'] = f'=SUM(I{STUDENT_START_ROW}:I{actual_last})'
    ws['J36'] = f'=SUM(J{STUDENT_START_ROW}:J{actual_last})'
    ws['K36'] = f'=SUM(K{STUDENT_START_ROW}:K{actual_last})'
    ws['L36'] = f'=SUM(L{STUDENT_START_ROW}:L{actual_last})'
    ws['M36'] = f'=SUM(M{STUDENT_START_ROW}:M{actual_last})'
    ws['N36'] = f'=SUM(N{STUDENT_START_ROW}:N{actual_last})'
    ws['O36'] = f'=SUM(O{STUDENT_START_ROW}:O{actual_last})'

    for sc in SUBJECT_COLS[:n_subjects]:
        cidx = col_index(sc)
        ws.cell(row=37, column=cidx).value = f'=IFERROR(AVERAGE({sc}{STUDENT_START_ROW}:{sc}{actual_last}),"")'
        ws.cell(row=38, column=cidx).value = f'=COUNTIF({sc}{STUDENT_START_ROW}:{sc}{actual_last},5)'
        ws.cell(row=39, column=cidx).value = f'=COUNTIF({sc}{STUDENT_START_ROW}:{sc}{actual_last},4)'
        ws.cell(row=40, column=cidx).value = f'=COUNTIF({sc}{STUDENT_START_ROW}:{sc}{actual_last},3)'
        ws.cell(row=41, column=cidx).value = f'=COUNTIF({sc}{STUDENT_START_ROW}:{sc}{actual_last},2)'

    for i in range(n_subjects, 6):
        sc = SUBJECT_COLS[i]
        cidx = col_index(sc)
        for rr in [37, 38, 39, 40, 41]:
            ws.cell(row=rr, column=cidx).value = None

    ws['I39'] = f'=IFERROR(AVERAGE(C37:{last_subj_col}37),"")'
    ws['C43'] = f'=IFERROR((I36*5+J36*4+K36*3)/(I36*5+J36*4+K36*3+L36*2)*100,"")'
    ws['C44'] = f'=IFERROR((I36*5+J36*4)/(I36*5+J36*4+K36*3+L36*2)*100,"")'
    ws['C45'] = f'=COUNTIF(L{STUDENT_START_ROW}:L{actual_last},">0")'
    ws['M39'] = f'=IFERROR((100-(O36/(C46*C47))*100),"")'

    wb.save(output_path)
    print(output_path)


if __name__ == '__main__':
    try:
        main()
    except Exception as e:
        print(f'ERROR: {e}', file=sys.stderr)
        sys.exit(1)