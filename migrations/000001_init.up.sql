-- Rashnu — full schema initialisation
-- Consolidated from migrations 001-010. Run on a fresh database.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS datacenters (
    id               UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    name             TEXT          NOT NULL UNIQUE,
    short            TEXT          NOT NULL DEFAULT '',
    provider         TEXT          NOT NULL DEFAULT '',
    region           TEXT          NOT NULL,
    racks            INT           NOT NULL DEFAULT 0,
    power_kw         NUMERIC(10,2) NOT NULL DEFAULT 0,
    tier             TEXT          NOT NULL DEFAULT '',
    monthly_colo_fee BIGINT        NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS infrastructure_hardwares (
    id               UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    datacenter_id    UUID        NOT NULL REFERENCES datacenters(id) ON DELETE CASCADE,
    hostname         TEXT        NOT NULL,
    hardware_type    TEXT        NOT NULL,
    purchase_price   BIGINT      NOT NULL,
    purchase_date    DATE        NOT NULL,
    region           TEXT        NOT NULL,
    model            TEXT        NOT NULL,
    specs            JSONB       NOT NULL DEFAULT '{}',
    amort_months     INT         NOT NULL DEFAULT 60,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (hostname, region, datacenter_id)
);

CREATE TABLE IF NOT EXISTS servers (
    id               UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    datacenter_id    UUID        NOT NULL REFERENCES datacenters(id),
    hostname         TEXT        NOT NULL,
    vendor           TEXT        NOT NULL DEFAULT '',
    model            TEXT        NOT NULL DEFAULT '',
    purchase_price   BIGINT      NOT NULL,
    purchase_date    DATE        NOT NULL,
    amort_months     INT         NOT NULL DEFAULT 60,
    vcpus            INT         NOT NULL DEFAULT 0,
    memory_gb        INT         NOT NULL DEFAULT 0,
    ssd_capacity_gb  INT         NOT NULL DEFAULT 0,
    hdd_capacity_gb  INT         NOT NULL DEFAULT 0,
    gpus             INT         NOT NULL DEFAULT 0,
    power_watts      INT         NOT NULL DEFAULT 0,
    rack_units       INT         NOT NULL DEFAULT 1,
    role             TEXT        NOT NULL DEFAULT 'bare_metal',
    region           TEXT        NOT NULL,
    status           TEXT        NOT NULL DEFAULT 'active',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (hostname, region, datacenter_id)
);

CREATE INDEX IF NOT EXISTS idx_servers_datacenter ON servers(datacenter_id);
CREATE INDEX IF NOT EXISTS idx_servers_status     ON servers(status);

CREATE TABLE IF NOT EXISTS pricing (
    id            UUID        PRIMARY KEY,
    datacenter_id UUID        NOT NULL,
    month         DATE        NOT NULL,
    cpu_price     BIGINT      NOT NULL,
    memory_price  BIGINT      NOT NULL,
    ssd_price     BIGINT      NOT NULL,
    hdd_price     BIGINT      NOT NULL,
    gpu_price     BIGINT      NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (datacenter_id, month)
);

CREATE TABLE IF NOT EXISTS daily_pricing (
    id            UUID        PRIMARY KEY,
    datacenter_id UUID        NOT NULL,
    date          DATE        NOT NULL,
    cpu_price     BIGINT      NOT NULL,
    memory_price  BIGINT      NOT NULL,
    ssd_price     BIGINT      NOT NULL,
    hdd_price     BIGINT      NOT NULL,
    gpu_price     BIGINT      NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (datacenter_id, date)
);

CREATE TABLE IF NOT EXISTS users (
    id         UUID      PRIMARY KEY,
    full_name  TEXT      NOT NULL,
    email      TEXT      NOT NULL UNIQUE,
    role       TEXT      NOT NULL DEFAULT 'user',
    status     TEXT      NOT NULL DEFAULT 'active',
    password   TEXT      NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    deleted_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

CREATE TABLE IF NOT EXISTS services (
    id          UUID      PRIMARY KEY,
    name        VARCHAR   NOT NULL UNIQUE,
    description TEXT      NOT NULL DEFAULT '',
    platform    VARCHAR   NOT NULL CHECK (platform IN ('kubernetes', 'vm')),
    team        VARCHAR   NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMP
);

CREATE TABLE IF NOT EXISTS service_pods (
    id              UUID         PRIMARY KEY,
    service_id      UUID         NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    name            VARCHAR      NOT NULL,
    namespace       VARCHAR      NOT NULL,
    vcpu            INTEGER      NOT NULL DEFAULT 0,
    memory_gb       INTEGER      NOT NULL DEFAULT 0,
    ssd_gb          INTEGER      NOT NULL DEFAULT 0,
    hdd_gb          INTEGER      NOT NULL DEFAULT 0,
    gpus            INTEGER      NOT NULL DEFAULT 0,
    cpu_usage       NUMERIC(8,3) NOT NULL DEFAULT 0,
    memory_usage_gb NUMERIC(8,3) NOT NULL DEFAULT 0,
    datacenter_id   UUID         NOT NULL REFERENCES datacenters(id),
    created_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS service_vms (
    id              UUID         PRIMARY KEY,
    service_id      UUID         NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    name            VARCHAR      NOT NULL,
    vcpu            INTEGER      NOT NULL DEFAULT 0,
    memory_gb       INTEGER      NOT NULL DEFAULT 0,
    ssd_gb          INTEGER      NOT NULL DEFAULT 0,
    hdd_gb          INTEGER      NOT NULL DEFAULT 0,
    gpus            INTEGER      NOT NULL DEFAULT 0,
    cpu_usage       NUMERIC(8,3) NOT NULL DEFAULT 0,
    memory_usage_gb NUMERIC(8,3) NOT NULL DEFAULT 0,
    datacenter_id   UUID         NOT NULL REFERENCES datacenters(id),
    created_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_service_pods_service_id ON service_pods(service_id);
CREATE INDEX IF NOT EXISTS idx_service_vms_service_id  ON service_vms(service_id);

CREATE TABLE IF NOT EXISTS prometheus_config (
    id         INTEGER   PRIMARY KEY DEFAULT 1,
    url        VARCHAR   NOT NULL DEFAULT '',
    enabled    BOOLEAN   NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT single_row CHECK (id = 1)
);

INSERT INTO prometheus_config (id, url, enabled)
VALUES (1, '', false)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS service_cost_reports (
    id             UUID        PRIMARY KEY,
    service_id     UUID        NOT NULL,
    service_name   TEXT        NOT NULL,
    team           TEXT        NOT NULL,
    platform       TEXT        NOT NULL,
    workload_count INTEGER     NOT NULL DEFAULT 0,
    date           DATE        NOT NULL,
    compute_cost   BIGINT      NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (service_id, date)
);

CREATE INDEX IF NOT EXISTS idx_service_cost_reports_date       ON service_cost_reports(date);
CREATE INDEX IF NOT EXISTS idx_service_cost_reports_service_id ON service_cost_reports(service_id);

CREATE TABLE IF NOT EXISTS service_cost_reports_by_dc (
    id              UUID        PRIMARY KEY,
    service_id      UUID        NOT NULL,
    datacenter_id   UUID        NOT NULL,
    datacenter_name TEXT        NOT NULL,
    workload_count  INTEGER     NOT NULL DEFAULT 0,
    date            DATE        NOT NULL,
    compute_cost    BIGINT      NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (service_id, datacenter_id, date)
);

CREATE INDEX IF NOT EXISTS idx_svc_cost_by_dc_date       ON service_cost_reports_by_dc(date);
CREATE INDEX IF NOT EXISTS idx_svc_cost_by_dc_service_id ON service_cost_reports_by_dc(service_id);
CREATE INDEX IF NOT EXISTS idx_svc_cost_by_dc_dc_id      ON service_cost_reports_by_dc(datacenter_id);
