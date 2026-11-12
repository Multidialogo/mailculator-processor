#!/bin/bash
set -e

echo "Running MySQL migrations..."

for f in /docker-entrypoint-initdb.d/migrations/*.sql; do
    echo "Executing $f..."
    mysql -u root -p"$MYSQL_ROOT_PASSWORD" "$MYSQL_DATABASE" < "$f"
done

echo "Migrations completed. Creating finish marker..."
touch /var/lib/mysql/zz-finish

echo "MySQL initialization complete."
