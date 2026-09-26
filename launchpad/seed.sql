-- Тестовые данные, чтобы проект можно было сразу посмотреть.

INSERT INTO users (name, email, role) VALUES
    ('Родион Берсенев', 'rodion@launchpad.dev', 'lead'),
    ('Сергей Охотников', 'sergey@launchpad.dev', 'producer'),
    ('Анна Гурьева', 'anna@launchpad.dev', 'methodist'),
    ('Марат Юсупов', 'marat@launchpad.dev', 'marketer');

INSERT INTO projects (title, description, status, owner_id, budget) VALUES
    ('Курс по Go для бэкендеров', 'С нуля до продакшена: HTTP, БД, докер', 'in_progress', 2, 150000.00),
    ('Интенсив по продуктовой аналитике', 'Метрики, воронки, A/B-тесты за 4 недели', 'search', 1, NULL),
    ('Курс по дизайну интерфейсов', 'Уже отснят, готовим к запуску', 'launched', 2, 220000.00);

INSERT INTO launches (project_id, title, status, start_date, end_date) VALUES
    (1, 'Осенний поток', 'running', '2026-09-01', '2026-11-15'),
    (1, 'Зимний поток', 'planned', '2026-12-01', '2027-02-01'),
    (3, 'Первый запуск', 'finished', '2026-06-01', '2026-07-15');

INSERT INTO tasks (launch_id, title, status, assignee_id, due_date) VALUES
    (1, 'Согласовать программу с методистом', 'done', 3, '2026-08-20'),
    (1, 'Собрать лендинг', 'in_progress', 2, '2026-09-10'),
    (1, 'Настроить рассылку для потока', 'todo', 4, '2026-09-25'),
    (2, 'Обновить программу под зимний поток', 'todo', 3, '2026-11-01');

INSERT INTO comments (task_id, user_id, text) VALUES
    (1, 3, 'Программу утвердили, можно верстать лендинг'),
    (2, 2, 'Лендинг почти готов, жду тексты от методиста'),
    (2, 1, 'Дедлайн не двигаем, зовите, если нужна помощь');
