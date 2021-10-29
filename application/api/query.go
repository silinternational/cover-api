package api

import (
	"strconv"
	"strings"

	"github.com/gobuffalo/buffalo"
)

// Query contains criteria to limit the results of List endpoints
type Query struct {
	// Search is a map of field name to search text. Field name may be "meta" keys to search across
	// multiple fields, e.g. "Name" searches "FirstName" and "LastName".
	Search map[string]string

	// Limit sets the number of records returned in a single page. Minimum is 1, maximum is 50
	Limit int
}

// NewQuery parses query string parameter values into valid query criteria.
//
// Example:
//   "search=name:John,description:MacBook" becomes Query{Search:
//   map[string]string{"name":"John","description":"MacBook"}}
func NewQuery(values buffalo.ParamValues) Query {
	q := Query{Limit: 10}

	if search := values.Get("search"); search != "" {
		pairs := strings.Split(search, ",")
		for _, p := range pairs {
			split := strings.SplitN(p, ":", 2)
			if len(split) == 2 {
				q.Search[split[0]] = split[1]
			}
		}
	}

	if limit := values.Get("limit"); limit != "" {
		i, err := strconv.Atoi(limit)
		if err != nil && i > 0 && i <= 50 {
			q.Limit = i
		}
	}

	return q
}
