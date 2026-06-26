-- Drop all tables in reverse dependency order.
DROP TABLE IF EXISTS service_cost_reports_by_dc;
DROP TABLE IF EXISTS service_cost_reports;
DROP TABLE IF EXISTS prometheus_config;
DROP TABLE IF EXISTS service_vms;
DROP TABLE IF EXISTS service_pods;
DROP TABLE IF EXISTS services;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS daily_pricing;
DROP TABLE IF EXISTS pricing;
DROP TABLE IF EXISTS infrastructure_hardwares;
DROP TABLE IF EXISTS servers;
DROP TABLE IF EXISTS datacenters;
DROP EXTENSION IF EXISTS "uuid-ossp";