package api

import (
	"net/url"
	"testing"

	"github.com/gobuffalo/buffalo"
)

func (ts *TestSuite) TestNewQuery() {
	tests := []struct {
		name string
		qs   string
		want Query
	}{
		{
			name: "default",
			qs:   "",
			want: Query{Limit: 10, Search: map[string]string{}},
		},
		{
			name: "limit and name",
			qs:   "limit=2&search=name:john",
			want: Query{Limit: 2, Search: map[string]string{"name": "john"}},
		},
		{
			name: "spaces",
			qs:   "limit= 2 &search= name : john smith ",
			want: Query{Limit: 2, Search: map[string]string{"name": "john smith"}},
		},
	}
	for _, tt := range tests {
		ts.T().Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.qs)

			got := NewQuery(buffalo.ParamValues(values))
			ts.Equal(tt.want, got)
		})
	}
}
