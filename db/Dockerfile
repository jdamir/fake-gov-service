FROM postgres:15-alpine

# Копируем наш файл инициализации структуры и мастер-данных в системную папку Postgres
COPY init-db.sql /docker-entrypoint-initdb.d/init-db.sql

# Предоставляем права на чтение для автоматического наката скрипта при старте
RUN chmod 644 /docker-entrypoint-initdb.d/init-db.sql

# Внутренний порт контейнера остается стандартным
EXPOSE 5432
