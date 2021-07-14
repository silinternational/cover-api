package api

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// TestSuite establishes a test suite for domain tests
type TestSuite struct {
	suite.Suite
}

// Test_TestSuite runs the test suite
func Test_TestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (ts *TestSuite) Test_keyToReadableString() {
	t := ts.T()

	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "all lowercase",
			key:  "lower",
			want: "lower",
		},
		{
			name: "one word",
			key:  "Single",
			want: "Single",
		},
		{
			name: "multiple words",
			key:  "ThisHasManyWords",
			want: "This has many words",
		},
		{
			name: "initial lowercase gets lost",
			key:  "initialLowerGetsLost",
			want: "Lower gets lost",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := keyToReadableString(tt.key)
			ts.Equal(tt.want, got)
		})
	}
}
