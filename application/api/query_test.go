package api

import (
	"net/url"
	"testing"

	"github.com/gobuffalo/buffalo"
)

func (ts *TestSuite) TestNewQuery() {
	tests := []struct {
		name           string
		qs             string
		wantLimit      int
		wantSearchName string
	}{
		{
			name:           "default",
			qs:             "",
			wantLimit:      10,
			wantSearchName: "",
		},
		{
			name:           "limit and name",
			qs:             "limit=2&search=name:john",
			wantLimit:      2,
			wantSearchName: "john",
		},
		{
			name:           "spaces",
			qs:             "limit= 2 &search= name : john smith ",
			wantLimit:      2,
			wantSearchName: "john smith",
		},
	}
	for _, tt := range tests {
		ts.T().Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.qs)

			got := NewQuery(buffalo.ParamValues(values))
			ts.Equal(tt.wantLimit, got.Limit(), "limit is incorrect")
			ts.Equal(tt.wantSearchName, got.Search("name"), "search name is incorrect")
		})
	}
}
