-- +goose Up

-- System-wide permission definitions
CREATE TABLE permissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key         TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Roles belong to an org (org_id NULL = system template role)
CREATE TABLE roles (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id         UUID        REFERENCES organizations(id) ON DELETE CASCADE,
    name           TEXT        NOT NULL,
    is_system_role BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, name)
);

CREATE INDEX idx_roles_org_id ON roles (org_id);

CREATE TABLE role_permissions (
    role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- Seed system permission keys
INSERT INTO permissions (key, description) VALUES
    ('tickets:create',           'Create tickets'),
    ('tickets:view',             'View tickets'),
    ('tickets:update_own',       'Update own tickets'),
    ('tickets:update_any',       'Update any ticket'),
    ('tickets:delete',           'Delete tickets'),
    ('tickets:assign',           'Assign tickets to users or teams'),
    ('tickets:change_status',    'Change ticket status'),
    ('tickets:set_priority',     'Set ticket priority'),
    ('tickets:add_tags',         'Add or remove tags on tickets'),
    ('comments:create_public',   'Post public comments'),
    ('comments:create_internal', 'Post internal (agent-only) comments'),
    ('comments:edit_own',        'Edit own comments'),
    ('comments:delete_own',      'Delete own comments'),
    ('teams:view',               'View teams'),
    ('teams:manage',             'Create, update, delete teams'),
    ('users:view',               'View organization members'),
    ('users:invite',             'Invite new users'),
    ('users:manage_roles',       'Assign or change user roles'),
    ('users:deactivate',         'Deactivate users'),
    ('organization:view_settings',   'View organization settings'),
    ('organization:manage_settings', 'Update organization settings'),
    ('sla:view',               'View SLA policies'),
    ('sla:manage',             'Create and update SLA policies'),
    ('reports:view',           'View reports and dashboards'),
    ('api_keys:manage',        'Create and revoke API keys');

-- +goose Down
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS permissions;
