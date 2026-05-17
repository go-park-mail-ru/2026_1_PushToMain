#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 \
     --username "$POSTGRES_USER" \
     --dbname "$POSTGRES_DBNAME" <<-EOSQL

CREATE USER smail_migrate WITH PASSWORD '$SMAIL_MIGRATE_PASSWORD';
GRANT CONNECT ON DATABASE $POSTGRES_DBNAME TO smail_migrate;
GRANT CREATE, USAGE ON SCHEMA public TO smail_migrate;

CREATE USER smail_user_svc WITH PASSWORD '$SMAIL_USER_SVC_PASSWORD';
CREATE USER smail_email_svc WITH PASSWORD '$SMAIL_EMAIL_SVC_PASSWORD';
CREATE USER smail_folder_svc WITH PASSWORD '$SMAIL_FOLDER_SVC_PASSWORD';

GRANT CONNECT ON DATABASE $POSTGRES_DBNAME TO smail_user_svc, smail_email_svc, smail_folder_svc;
GRANT USAGE ON SCHEMA public TO smail_user_svc, smail_email_svc, smail_folder_svc;

ALTER DEFAULT PRIVILEGES FOR ROLE smail_migrate IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO smail_user_svc;
ALTER DEFAULT PRIVILEGES FOR ROLE smail_migrate IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO smail_email_svc;
ALTER DEFAULT PRIVILEGES FOR ROLE smail_migrate IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO smail_folder_svc;
ALTER DEFAULT PRIVILEGES FOR ROLE smail_migrate IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO smail_user_svc, smail_email_svc, smail_folder_svc;

ALTER ROLE smail_user_svc SET statement_timeout = '30s';
ALTER ROLE smail_email_svc SET statement_timeout = '30s';
ALTER ROLE smail_folder_svc SET statement_timeout = '30s';

ALTER ROLE smail_user_svc SET lock_timeout = '5s';
ALTER ROLE smail_email_svc SET lock_timeout = '5s';
ALTER ROLE smail_folder_svc SET lock_timeout = '5s';

EOSQL
