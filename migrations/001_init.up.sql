-- Группы
CREATE TABLE IF NOT EXISTS groups (
    id   SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Предметы
CREATE TABLE IF NOT EXISTS subjects (
    id       SERIAL PRIMARY KEY,
    name     TEXT NOT NULL,
    group_id INTEGER REFERENCES groups(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (name, group_id)
    );

-- Студенты
CREATE TABLE IF NOT EXISTS students (
    id        SERIAL PRIMARY KEY,
    full_name TEXT NOT NULL,
    group_id  INTEGER NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (full_name, group_id)
    );

-- Оценки
CREATE TABLE IF NOT EXISTS ocenki (
    id          SERIAL PRIMARY KEY,
    student_id  INTEGER NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    subject_id  INTEGER NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    lesson_date DATE NOT NULL,
    grade       INT CHECK (grade >= 2 AND grade <= 5),
    status      TEXT DEFAULT 'present',
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (student_id, subject_id, lesson_date)
    );

-- Пользователи
CREATE TABLE IF NOT EXISTS users (
    id        SERIAL PRIMARY KEY,
    full_name TEXT NOT NULL,
    role      TEXT NOT NULL CHECK (role IN ('teacher', 'student')),
    group_id  INTEGER REFERENCES groups(id) ON DELETE SET NULL,
    password  TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (full_name, role, group_id)
    );

-- Индексы для производительности
CREATE INDEX IF NOT EXISTS idx_ocenki_student_id ON ocenki(student_id);
CREATE INDEX IF NOT EXISTS idx_ocenki_subject_id ON ocenki(subject_id);
CREATE INDEX IF NOT EXISTS idx_ocenki_lesson_date ON ocenki(lesson_date);
CREATE INDEX IF NOT EXISTS idx_students_group_id ON students(group_id);
CREATE INDEX IF NOT EXISTS idx_subjects_group_id ON subjects(group_id);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Триггер для обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_groups_updated_at BEFORE UPDATE ON groups
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_subjects_updated_at BEFORE UPDATE ON subjects
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_students_updated_at BEFORE UPDATE ON students
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_ocenki_updated_at BEFORE UPDATE ON ocenki
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();