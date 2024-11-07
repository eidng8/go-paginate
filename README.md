# go-paginate

A simple module providing pagination feature to use with Ent.

## Usage

```golang
package main

import (
	"context"

	"github.com/eidng8/go-paginate"
	"github.com/gin-gonic/gin"

	"your_project/ent"
	"your_project/ent/your_model"
)

func getPage(ctx context.Context, query *ent.your_query) (*paginate.PaginatedList[ent.your_model], error) { 
    // get the gin context to be used for the pagination
    gc := ctx.(*gin.Context)
    // creates a context to be used for ent query execution, 
    // e.g. if soft delete from the official site is used
    qc := SkipSoftDelete(context.Background())
    pageParams := paginate.GetPaginationParams(gc)
    // MUST be explicitly sorted, doesn't need this line if the query is already sorted
    query.Order(your_model.ByID())
    // optionally, add more predicates if needed
    query.Where(predicate1, predicate2, ...)
    // call `paginate.GetPage()` function to get the paginated result
    page, err := paginate.GetPage[ent.your_model](gc, qc, query, pageParams)
    if err != nil {
        return nil, err
    }
    return page, nil
}
```

## Functions

`GetRequestBase(req *http.Request) *url.URL`
