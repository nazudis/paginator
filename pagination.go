package paginator

import (
	"gorm.io/gorm"
	"math"
	"reflect"
	"sync"
)

type Param struct {
	DB      *gorm.DB
	Page    int
	Limit   int
	OrderBy []string
	ShowSQL bool
}

type Pagination[T any] struct {
	From        int   `json:"from"`
	To          int   `json:"to"`
	Total       int64 `json:"total"`
	Data        T     `json:"data"`
	PerPage     int   `json:"per_page"`
	CurrentPage int   `json:"current_page"`
	Offset      int   `json:"-"`
	PrevPage    *int  `json:"prev_page"`
	NextPage    *int  `json:"next_page"`
	LastPage    int   `json:"last_page"`
}

func Paginate[T any](p Param, result T) Pagination[T] {
	db := p.DB

	if p.ShowSQL {
		db = db.Debug()
	}
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit == 0 {
		p.Limit = 25
	}
	if len(p.OrderBy) > 0 {
		for _, o := range p.OrderBy {
			db = db.Order(o)
		}
	}

	var paginate Pagination[T]
	var countInPage int
	var count int64
	var offset int

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go countRecords[T](wg, db, result, &count)

	if p.Page == 1 {
		offset = 0
	} else {
		offset = (p.Page - 1) * p.Limit
	}
	db.Session(&gorm.Session{}).Limit(p.Limit).Offset(offset).Find(result)

	indirect := reflect.ValueOf(result)
	if indirect.IsValid() && indirect.Elem().Kind() == reflect.Slice {
		countInPage = indirect.Elem().Len()
	}

	wg.Wait()

	paginate.Total = count
	paginate.Data = result
	paginate.CurrentPage = p.Page

	paginate.Offset = offset
	paginate.PerPage = p.Limit
	paginate.LastPage = int(math.Ceil(float64(count) / float64(p.Limit)))
	if countInPage > 0 {
		paginate.From = offset + 1
		paginate.To = offset + countInPage
	} else {
		paginate.From = 0
		paginate.To = 0
	}

	if p.Page > 1 {
		prevPage := p.Page - 1
		paginate.PrevPage = &prevPage
	}

	if p.Page < paginate.LastPage {
		nextPage := p.Page + 1
		paginate.NextPage = &nextPage
	}
	return paginate
}

func countRecords[T any](wg *sync.WaitGroup, db *gorm.DB, anyType T, count *int64) {
	db.Session(&gorm.Session{}).Model(anyType).Count(count)
	wg.Done()
}
