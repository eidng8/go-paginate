package paginate

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"

	eu "github.com/eidng8/go-url"
	"github.com/gin-gonic/gin"
)

const (
	// ParamPage is the query parameter name for the page number.
	ParamPage = "page"

	// ParamPerPage is the query parameter name for the number of items per page.
	ParamPerPage = "per_page"
)

// PaginatedParams is a struct that contains the page and per_page parameters.
type PaginatedParams struct {
	// Page is the current page number.
	Page int `form:"page"`
	// PerPage is the number of items per page.
	PerPage int `form:"per_page"`
}

// GetPage returns the current page number.
func (pp *PaginatedParams) GetPage() int {
	if pp.Page < 1 {
		return 1
	}
	return pp.Page
}

// GetPerPage returns the number of items per page.
func (pp *PaginatedParams) GetPerPage() int {
	if pp.PerPage < 1 {
		return 10
	}
	return pp.PerPage
}

// GetPaginationParams returns the PaginatedParams from the gin.Context,
// with default values of page `1` and `10` items per page.
func GetPaginationParams(gc *gin.Context) PaginatedParams {
	return GetPaginationParamsWithDefault(gc, 1, 10)
}

// GetPaginationParamsWithDefault returns the PaginatedParams from the
// gin.Context with given default values.
func GetPaginationParamsWithDefault(
	gc *gin.Context, defaultPage, defaultPerPage int,
) PaginatedParams {
	var page PaginatedParams
	if gc.ShouldBind(&page) != nil {
		page.Page = defaultPage
		page.PerPage = defaultPerPage
	}
	if page.Page < 1 {
		page.Page = defaultPage
	}
	if page.PerPage < 1 {
		page.PerPage = defaultPerPage
	}
	return page
}

// PaginatedList is a struct that contains the paginated list of items.
type PaginatedList[T any] struct {
	// Total is the total number of items.
	Total int `json:"total" bson:"total" xml:"total" yaml:"total"`
	// PerPage is the number of items per page.
	PerPage int `json:"per_page" bson:"per_page" xml:"per_page" yaml:"per_page"`
	// CurrentPage is the current page number.
	CurrentPage int `json:"current_page" bson:"current_page" xml:"current_page" yaml:"current_page"`
	// LastPage is the last page number.
	LastPage int `json:"last_page" bson:"last_page" xml:"last_page" yaml:"last_page"`
	// FirstPageUrl is the URL of the first page.
	FirstPageUrl string `json:"first_page_url" bson:"first_page_url" xml:"first_page_url" yaml:"first_page_url"`
	// LastPageUrl is the URL of the last page. It is an empty string if there
	// is only one page.
	LastPageUrl string `json:"last_page_url" bson:"last_page_url" xml:"last_page_url" yaml:"last_page_url"`
	// NextPageUrl is the URL of the next page. It is an empty string if the
	// current page is the last page.
	NextPageUrl string `json:"next_page_url" bson:"next_page_url" xml:"next_page_url" yaml:"next_page_url"`
	// PrevPageUrl is the URL of the previous page. It is an empty string if
	// the current page is the first page.
	PrevPageUrl string `json:"prev_page_url" bson:"prev_page_url" xml:"prev_page_url" yaml:"prev_page_url"`
	// Path is the fully qualified URL without query string.
	Path string `json:"path" bson:"path" xml:"path" yaml:"path"`
	// From is the starting 1-based index of the items.
	From int `json:"from" bson:"from" xml:"from" yaml:"from"`
	// To is the ending 1-based index of the items.
	To int `json:"to" bson:"to" xml:"to" yaml:"to"`
	// Data is the list of items.
	Data []*T `json:"data" bson:"data" xml:"data" yaml:"data"`
}

// PQ is an interface that defines the methods for queries to be paginated.
type PQ[I any, Q any] interface {
	Offset(int) *Q
	Limit(int) *Q
	Count(context.Context) (int, error)
	All(context.Context) ([]*I, error)
}

// GetPage returns a paginated list of items. `V` is the type of items in the
// paginated list. `Q` is the query type to be used to retrieve items, which in
// most cases can be inferred. So in most cases, only the `V` needs to be
// provided.
//
// The `gc` parameter is the gin.Context to be used to generate various links in
// the paginated list; `qc` is the context to be used in query execution;
// `query` is the ent query instance to be executed; and `params` is the
// PaginatedParams to be used in pagination.
//
// Please remember to explicitly add the `ORDER` clause to the query before
// calling this function.
func GetPage[V any, Q any, T PQ[V, Q]](
	gc *gin.Context, qc context.Context, query T, params PaginatedParams,
) (*PaginatedList[V], error) {
	var next, prev string
	fi := 1
	ni := params.Page + 1
	pi := params.Page - 1
	req := gc.Request
	count, err := query.Count(qc)
	if err != nil {
		return nil, err
	}
	if 0 == count {
		return &PaginatedList[V]{
			Total:        0,
			PerPage:      params.PerPage,
			CurrentPage:  1,
			LastPage:     1,
			FirstPageUrl: UrlWithPage(req, 1, params.PerPage).String(),
			LastPageUrl:  "",
			NextPageUrl:  "",
			PrevPageUrl:  "",
			Path:         eu.RequestBaseUrl(req).String(),
			From:         0,
			To:           0,
			Data:         []*V{},
		}, nil
	}
	from := pi*params.PerPage + 1
	to := int(math.Min(float64(params.Page*params.PerPage), float64(count)))
	query.Offset(pi * params.PerPage)
	query.Limit(params.PerPage)
	rows, err := query.All(qc)
	if err != nil {
		return nil, err
	}
	li := int(math.Ceil(float64(count) / float64(params.PerPage)))
	first := UrlWithPage(req, fi, params.PerPage).String()
	var last string
	if li <= 1 {
		li = 1
		last = ""
	} else {
		last = UrlWithPage(req, li, params.PerPage).String()
	}
	if ni > li {
		ni = li
		next = ""
	} else {
		next = UrlWithPage(req, ni, params.PerPage).String()
	}
	if pi < 1 {
		pi = 1
		prev = ""
	} else {
		prev = UrlWithPage(req, pi, params.PerPage).String()
	}
	return &PaginatedList[V]{
		Total:        count,
		PerPage:      params.PerPage,
		CurrentPage:  params.Page,
		LastPage:     li,
		FirstPageUrl: first,
		LastPageUrl:  last,
		NextPageUrl:  next,
		PrevPageUrl:  prev,
		Path:         eu.RequestBaseUrl(req).String(),
		From:         from,
		To:           to,
		Data:         rows,
	}, nil
}

// GetPageMapped returns a paginated list of items. `I` is the type of items
// returned by the query, `V` is the type of items in the paginated list. `Q` is
// the query type to be used to retrieve items, which in most cases can be
// inferred. So in most cases, only the `I` and `V` types need to be provided.
//
// The `gc` parameter is the gin.Context to be used to generate various links in
// the paginated list; `qc` is the context to be used in query execution;
// `query` is the ent query instance to be executed; and `params` is the
// PaginatedParams to be used in pagination; the `mapper` is a function that
// maps the one query result row to an item in the paginated list, the 2nd
// parameter is the index of the item in the result set.
//
// Please remember to explicitly add the `ORDER` clause to the query before
// calling this function.
func GetPageMapped[I any, V any, Q any, T PQ[I, Q]](
	gc *gin.Context, qc context.Context, query T, page PaginatedParams,
	mapper func(*I, int) *V,
) (*PaginatedList[V], error) {
	list, err := GetPage[I, Q, T](gc, qc, query, page)
	if err != nil {
		return nil, err
	}
	data := make([]*V, len(list.Data))
	for i, row := range list.Data {
		data[i] = mapper(row, i)
	}
	return &PaginatedList[V]{
		Total:        list.Total,
		PerPage:      list.PerPage,
		CurrentPage:  list.CurrentPage,
		LastPage:     list.LastPage,
		FirstPageUrl: list.FirstPageUrl,
		LastPageUrl:  list.LastPageUrl,
		NextPageUrl:  list.NextPageUrl,
		PrevPageUrl:  list.PrevPageUrl,
		Path:         list.Path,
		From:         list.From,
		To:           list.To,
		Data:         data,
	}, nil
}

// UrlWithPage returns a URL with the page and per_page query parameters set.
func UrlWithPage(request *http.Request, page int, perPage int) *url.URL {
	return eu.RequestUrlWithQueryParams(request, PageQueryParams(page, perPage))
}

// UrlWithoutPageParams returns a URL without the page and per_page query
// parameters.
func UrlWithoutPageParams(req *http.Request) *url.URL {
	return eu.RequestUrlWithoutQueryParams(req, ParamPage, ParamPerPage)
}

// PageQueryParams sets the page and per_page query parameters.
func PageQueryParams(page int, perPage int) map[string]string {
	params := make(map[string]string, 2)
	params[ParamPage] = fmt.Sprintf("%d", page)
	params[ParamPerPage] = fmt.Sprintf("%d", perPage)
	return params
}
