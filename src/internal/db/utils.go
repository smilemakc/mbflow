package db

import (
	"fmt"
	"github.com/uptrace/bun"
)

func SortByFieldQuery(field, sort string) func(q *bun.SelectQuery) *bun.SelectQuery {
	return func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Order(fmt.Sprintf("%s %s", field, sort))
	}
}

func CreatedAtSortQuery(sort string) func(q *bun.SelectQuery) *bun.SelectQuery {
	return SortByFieldQuery("created_at", sort)
}

func UpdatedAtSortQuery(sort string) func(q *bun.SelectQuery) *bun.SelectQuery {
	return SortByFieldQuery("updated_at", sort)
}
