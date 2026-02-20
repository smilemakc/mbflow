-- MBFlow File Storage Migration - Rollback
-- Removes files and storage_configs tables

DROP TABLE IF EXISTS mbflow_files;
DROP TABLE IF EXISTS mbflow_storage_configs;
