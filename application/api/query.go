package api

import (
	"strconv"
	"strings"

	"github.com/gobuffalo/buffalo"
)

// QueryParams contains criteria to limit the results of List endpoints
type QueryParams struct {
	// filterKeys is a map of field name to filter text.
	filterKeys map[string]string

	// searchText is text to search across multiple fields
	searchText string

	// recordLimit sets the number of records returned in a single page. Minimum is 1, maximum is 50
	recordLimit int

	// page sets the pagination slice for the query
	page int
}

func (q QueryParams) Limit() int {
	l := q.recordLimit
	if l < 1 {
		l = 1
	}
	if l > 50 {
		l = 50
	}
	return q.recordLimit
}

func (q QueryParams) Page() int {
	p := q.page
	if p < 1 {
		p = 1
	}
	return q.page
}

func (q QueryParams) Filter(key string) string {
	return q.filterKeys[key]
}

func (q QueryParams) Search() string {
	return q.searchText
}

// NewQueryParams parses query string parameter values into valid query criteria.
//
// Example:
//   "filter=name:John,description:MacBook" becomes Query{filterKeys:
//   map[string]string{"name":"John","description":"MacBook"}}
func NewQueryParams(values buffalo.ParamValues) QueryParams {
	q := QueryParams{recordLimit: 10, filterKeys: map[string]string{}}

	q.searchText = values.Get("search")

	if filter := values.Get("filter"); filter != "" {
		pairs := strings.Split(strings.TrimSpace(filter), ",")
		for _, p := range pairs {
			split := strings.SplitN(p, ":", 2)
			if len(split) == 2 {
				q.filterKeys[strings.TrimSpace(split[0])] = strings.TrimSpace(split[1])
			}
		}
	}

	if limit := values.Get("limit"); limit != "" {
		i, err := strconv.Atoi(strings.TrimSpace(limit))
		if err == nil {
			q.recordLimit = i
		}
	}

	if page := values.Get("page"); page != "" {
		i, err := strconv.Atoi(strings.TrimSpace(page))
		if err == nil {
			q.page = i
		}
	}

	return q
}
