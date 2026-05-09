package spider

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidSpec_Valid(t *testing.T) {
	cases := []string{
		"0 */20 * * * ?",
		"0 0 0 * * *",
		"30 0 0 * * *",
	}
	for _, s := range cases {
		require.NoError(t, ValidSpec(s), "expect valid: %q", s)
	}
}

func TestValidSpec_Invalid(t *testing.T) {
	cases := []string{
		"",
		"not-cron",
		"0 0",
		"99 0 0 * * *", // seconds field 越界
	}
	for _, s := range cases {
		require.Error(t, ValidSpec(s), "expect invalid: %q", s)
	}
}
