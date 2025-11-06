package db

import "context"

type ForeignKey struct {
	TableName        string `bun:"table_name"`
	ConstraintName   string `bun:"constraint_name"`
	ColumnName       string `bun:"column_name"`
	ReferencedTable  string `bun:"referenced_table"`
	ReferencedColumn string `bun:"referenced_column"`
}

func GetForeignKeys(tableName string) ([]ForeignKey, error) {
	var foreignKeys []ForeignKey
	rows, err := DB().QueryContext(context.Background(), `SELECT
    conname AS constraint_name,
    conrelid::regclass AS table_name,
    a.attname AS column_name,
    confrelid::regclass AS referenced_table,
    af.attname AS referenced_column
FROM
    pg_constraint c
        JOIN pg_attribute a ON a.attnum = ANY(c.conkey) AND a.attrelid = c.conrelid
        JOIN pg_attribute af ON af.attnum = ANY(c.confkey) AND af.attrelid = c.confrelid
WHERE
    c.contype = 'f'  -- Foreign key constraint
  AND c.confrelid = ?::regclass`, tableName)
	if err != nil {
		return nil, err
	}

	if err := DB().ScanRows(context.Background(), rows, &foreignKeys); err != nil {
		return nil, err
	}
	return foreignKeys, nil
}
