-- MBFlow File Storage Migration - Rollback
-- Removes files and storage_configs tables

DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS storage_configs;
