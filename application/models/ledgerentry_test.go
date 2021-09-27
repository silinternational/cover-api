package models

import "testing"

func (ms *ModelSuite) Test_getFiscalPeriod() {
	tests := []struct {
		name  string
		month int
		want  int
	}{
		{
			name:  "September",
			month: 9,
			want:  1,
		},
		{
			name:  "August",
			month: 8,
			want:  12,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			ms.Equal(tt.want, getFiscalPeriod(tt.month))
		})
	}
}
