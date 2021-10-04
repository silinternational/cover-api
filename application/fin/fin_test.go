package fin

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/silinternational/cover-api/domain"
)

func Test_getFiscalPeriod(t *testing.T) {
	domain.Env.FiscalStartMonth = 9

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
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, getFiscalPeriod(tt.month))
		})
	}
}
