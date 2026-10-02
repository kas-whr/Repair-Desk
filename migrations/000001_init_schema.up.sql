CREATE TABLE categories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(100) NOT NULL UNIQUE,
    sla_hours  INT          NOT NULL CHECK (sla_hours > 0),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE equipment (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name             VARCHAR(255) NOT NULL,
    inventory_number VARCHAR(100) NOT NULL UNIQUE,
    location         VARCHAR(255) NOT NULL,
    status           VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE'
        CHECK (status IN ('ACTIVE', 'BROKEN', 'RETIRED'))
);

CREATE TABLE tickets (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title        VARCHAR(255) NOT NULL,
    description  TEXT         NOT NULL,
    status       VARCHAR(20)  NOT NULL DEFAULT 'NEW'
        CHECK (status IN ('NEW', 'IN_PROGRESS', 'RESOLVED', 'CLOSED')),
    priority     VARCHAR(20)  NOT NULL DEFAULT 'MEDIUM'
        CHECK (priority IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    category_id  UUID         NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    equipment_id UUID         REFERENCES equipment (id) ON DELETE SET NULL,
    due_at       TIMESTAMPTZ  NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_tickets_status ON tickets (status);
CREATE INDEX idx_tickets_category_id ON tickets (category_id);
CREATE INDEX idx_tickets_equipment_id ON tickets (equipment_id);

CREATE TABLE comments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id  UUID        NOT NULL REFERENCES tickets (id) ON DELETE CASCADE,
    content    TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_comments_ticket_id ON comments (ticket_id);

INSERT INTO categories (name, sla_hours) VALUES
    ('Hardware', 24),
    ('Software', 48),
    ('Network', 4),
    ('Printer', 12),
    ('Other', 72);
