#!/bin/bash

# Скрипт для исправления "dirty" миграций в production
# Использование: ./scripts/fix-migrations.sh

set -e

echo "🔧 Исправление dirty миграций..."

# Переменные
DB_HOST=${DB_HOST:-"localhost"}
DB_PORT=${DB_PORT:-"5434"}
DB_USER=${DB_USER:-"driver_service"}
DB_PASSWORD=${DB_PASSWORD:-"driver_service_password"}
DB_NAME=${DB_NAME:-"driver_service"}

# Формируем connection string
DB_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"

echo "📊 Подключение к базе данных: ${DB_HOST}:${DB_PORT}/${DB_NAME}"

# Проверяем подключение к БД
echo "🔍 Проверка подключения к PostgreSQL..."
until pg_isready -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${DB_NAME}; do
  echo "⏳ Ожидание PostgreSQL..."
  sleep 2
done

echo "✅ PostgreSQL доступен"

# Проверяем состояние миграций
echo "📋 Проверка состояния миграций..."
MIGRATION_STATUS=$(psql "${DB_URL}" -t -c "SELECT dirty FROM schema_migrations ORDER BY version DESC LIMIT 1;" 2>/dev/null || echo "error")

if [ "$MIGRATION_STATUS" = "error" ]; then
  echo "❌ Ошибка подключения к базе данных"
  exit 1
fi

if [ "$MIGRATION_STATUS" = "t" ]; then
  echo "⚠️  Обнаружена dirty миграция. Исправляем..."
  
  # Получаем текущую версию
  CURRENT_VERSION=$(psql "${DB_URL}" -t -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;" 2>/dev/null | tr -d ' ')
  
  echo "📌 Текущая версия миграции: ${CURRENT_VERSION}"
  
  # Сбрасываем dirty флаг
  echo "🔧 Сброс dirty флага..."
  psql "${DB_URL}" -c "UPDATE schema_migrations SET dirty = false WHERE version = ${CURRENT_VERSION};"
  
  echo "✅ Dirty флаг сброшен"
  
  # Принудительно устанавливаем версию
  echo "🔧 Принудительная установка версии ${CURRENT_VERSION}..."
  psql "${DB_URL}" -c "SELECT setval('schema_migrations_version_seq', ${CURRENT_VERSION});"
  
  echo "✅ Версия установлена"
  
else
  echo "✅ Миграции в чистом состоянии"
fi

# Проверяем финальное состояние
echo "🔍 Финальная проверка..."
FINAL_STATUS=$(psql "${DB_URL}" -t -c "SELECT dirty FROM schema_migrations ORDER BY version DESC LIMIT 1;" 2>/dev/null)
CURRENT_VERSION=$(psql "${DB_URL}" -t -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;" 2>/dev/null | tr -d ' ')

echo "📊 Состояние миграций:"
echo "   Версия: ${CURRENT_VERSION}"
echo "   Dirty: ${FINAL_STATUS}"

if [ "$FINAL_STATUS" = "f" ]; then
  echo "🎉 Миграции исправлены успешно!"
  exit 0
else
  echo "❌ Ошибка исправления миграций"
  exit 1
fi
