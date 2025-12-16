#!/bin/bash
set -e

echo "Running MariaDB migrations..."

for f in /docker-entrypoint-initdb.d/migrations/*.sql; do
    echo "Executing $f..."
    mysql -u root -p"$MARIADB_ROOT_PASSWORD" "$MARIADB_DATABASE" < "$f"
done

echo "Migrations completed. Creating finish marker..."
touch /var/lib/mysql/zz-finish

echo "MariaDB initialization complete."
