CREATE TYPE user_type_enum AS ENUM (
    'user',
    'admin',
    'moderator',
    'support'
);

CREATE TYPE group_type_enum AS ENUM (
    'system',
    'custom',
    'auto_generated'
);
CREATE TYPE permission_resource_enum AS ENUM (
    'messages',
    'contacts',
    'settings',
    'admin_panel',
    'users',
    'folders',
    'attachments'
);

CREATE TYPE permission_action_enum AS ENUM (
    'create',
    'read',
    'update',
    'delete',
    'manage'
);

CREATE TYPE folder_type_enum AS ENUM (
    'system',
    'user_created'
);

CREATE TYPE recipient_type_enum AS ENUM (
    'to',
    'cc',
    'bcc'
);

CREATE TYPE importance_enum AS ENUM (
    'low',
    'normal',
    'high'
);

CREATE TYPE theme_enum AS ENUM (
    'light',
    'dark',
    'system'
);

CREATE TYPE contact_type_enum AS ENUM (
    'internal',
    'external'
);

CREATE TYPE contact_category_type_enum AS ENUM (
    'system',
    'user_defined',
    'auto_generated'
);
