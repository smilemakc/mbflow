package db

import (
	"context"
	"fmt"
	"sync"
)

type ITables interface {
	Add(tableName string, model interface{}) error
	Register(model interface{})
	Get(name string) interface{}
}

var tables = map[string]interface{}{}

var bunTables *BunTables
var bunTablesOnce sync.Once

type BunTables struct {
	items map[string]interface{}
}

func (b *BunTables) Add(tableName string, model interface{}) error {
	if _, has := b.items[tableName]; has {
		return fmt.Errorf("%s alredy registered", tableName)
	}
	_, err := DB().NewCreateTable().
		IfNotExists().
		WithForeignKeys().
		Model(model).Exec(context.Background())
	if err != nil {
		return err
	}
	b.items[tableName] = model
	return nil
}

func (b *BunTables) Register(model interface{}) {
	DB().RegisterModel(model)

}

func (b *BunTables) Get(name string) interface{} {
	return b.items[name]
}

func GetTables() *BunTables {
	bunTablesOnce.Do(func() {
		bunTables = &BunTables{
			items: tables,
		}
	})
	return bunTables
}
