package api

import (
	"strconv"
	"strings"

	"github.com/gobuffalo/buffalo"
)

// Query contains criteria to limit the results of List endpoints
type Query struct {
	// searchKeys is a map of field name to search text. Field name may be "meta" keys to search across
	// multiple fields, e.g. "Name" searches "FirstName" and "LastName".
	searchKeys map[string]string

	// recordLimit sets the number of records returned in a single page. Minimum is 1, maximum is 50
	recordLimit int
}

func (q Query) Limit() int {
	l := q.recordLimit
	if l < 1 {
		l = 1
	}
	if l > 50 {
		l = 50
	}
	return q.recordLimit
}

func (q Query) Search(key string) string {
	return q.searchKeys[key]
}

// NewQuery parses query string parameter values into valid query criteria.
//
// Example:
//   "search=name:John,description:MacBook" becomes Query{searchKeys:
//   map[string]string{"name":"John","description":"MacBook"}}
func NewQuery(values buffalo.ParamValues) Query {
	q := Query{recordLimit: 10, searchKeys: map[string]string{}}

	if search := values.Get("search"); search != "" {

		pairs := strings.Split(strings.TrimSpace(search), ",")
		for _, p := range pairs {
			split := strings.SplitN(p, ":", 2)
			if len(split) == 2 {
				q.searchKeys[strings.TrimSpace(split[0])] = strings.TrimSpace(split[1])
			}
		}
	}

	if limit := values.Get("limit"); limit != "" {
		i, err := strconv.Atoi(strings.TrimSpace(limit))
		if err == nil {
			q.recordLimit = i
		}
	}

	return q
}
