package api

import (
	"net/url"
	"testing"

	"github.com/gobuffalo/buffalo"
)

func (ts *TestSuite) TestNewQuery() {
	tests := []struct {
		name             string
		qs               string
		wantLimit        int
		wantFilterActive string
		wantSearchText   string
	}{
		{
			name:             "default",
			qs:               "",
			wantLimit:        10,
			wantFilterActive: "",
		},
		{
			name:             "limit and active:true",
			qs:               "limit=2&filter=active:true",
			wantLimit:        2,
			wantFilterActive: "true",
		},
		{
			name:           "search",
			qs:             "search=john",
			wantLimit:      10,
			wantSearchText: "john",
		},
		{
			name:             "spaces",
			qs:               "limit= 2 &filter= active : true ",
			wantLimit:        2,
			wantFilterActive: "true",
		},
	}
	for _, tt := range tests {
		ts.T().Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.qs)

			got := NewQuery(buffalo.ParamValues(values))
			ts.Equal(tt.wantLimit, got.Limit(), "limit is incorrect")
			ts.Equal(tt.wantFilterActive, got.Filter("active"), "filter active is incorrect")
		})
	}
}
