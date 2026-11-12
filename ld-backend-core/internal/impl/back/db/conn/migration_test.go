package ldconn_test

import (
	ldconn "ld-backend/internal/impl/back/db/conn"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0", "1.0.0", 0},
		{"0.9.0", "1.0.0", -1},
		{"1.9.0", "1.10.0", -1},
		{"1.10.0", "1.9.0", 1},
		{"2.0.0", "1.99.99", 1},
	}
	for _, c := range cases {
		got, err := ldconn.CompareVersions(c.a, c.b)
		require.NoError(t, err)
		assert.Equal(t, c.want, got, "%s vs %s", c.a, c.b)
	}

	_, err := ldconn.CompareVersions("1.x.0", "1.0.0")
	assert.Error(t, err)
}
