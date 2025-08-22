-- Пример вывода конвертера Cronos to ClickHouse
-- Таблица: Отдел_кадров_Сотрудники -> otdel_kadrov_sotrudniki

CREATE TABLE IF NOT EXISTS `otdel_kadrov_sotrudniki` (
    `sistemnyy_nomer` UInt32,
    `familiya` String,
    `imya` String,
    `otchestvo` String,
    `data_rozhdeniya` Date,
    `vremya_sozdaniya` DateTime,
    `zarplata` Int32,
    `primechanie` String
) ENGINE = MergeTree() ORDER BY tuple();

INSERT INTO `otdel_kadrov_sotrudniki` (`sistemnyy_nomer`, `familiya`, `imya`, `otchestvo`, `data_rozhdeniya`, `vremya_sozdaniya`, `zarplata`, `primechanie`) VALUES (1, 'Иванов', 'Иван', 'Иванович', '1990-01-15', '2025-08-22 12:30:00', 50000, 'Хороший сотрудник');
INSERT INTO `otdel_kadrov_sotrudniki` (`sistemnyy_nomer`, `familiya`, `imya`, `otchestvo`, `data_rozhdeniya`, `vremya_sozdaniya`, `zarplata`, `primechanie`) VALUES (2, 'Петров', 'Петр', 'Петрович', '1985-03-22', '2025-08-22 12:31:00', 60000, 'Опытный специалист');

