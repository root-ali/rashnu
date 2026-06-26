ALTER TABLE service_cost_reports
    DROP COLUMN IF EXISTS cpu_cost,
    DROP COLUMN IF EXISTS ram_cost,
    DROP COLUMN IF EXISTS ssd_cost,
    DROP COLUMN IF EXISTS hdd_cost;

ALTER TABLE service_cost_reports_by_dc
    DROP COLUMN IF EXISTS cpu_cost,
    DROP COLUMN IF EXISTS ram_cost,
    DROP COLUMN IF EXISTS ssd_cost,
    DROP COLUMN IF EXISTS hdd_cost;