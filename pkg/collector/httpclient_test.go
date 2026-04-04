package collector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseRetryAfter_Seconds(t *testing.T) {
	assert.Equal(t, 120*time.Second, parseRetryAfter("120"))
	assert.Equal(t, 300*time.Second, parseRetryAfter("300"))
	assert.Equal(t, 1*time.Second, parseRetryAfter("1"))
}

func TestParseRetryAfter_Empty(t *testing.T) {
	assert.Equal(t, 5*time.Minute, parseRetryAfter(""))
}

func TestParseRetryAfter_Invalid(t *testing.T) {
	assert.Equal(t, 5*time.Minute, parseRetryAfter("not-a-number"))
}

func TestParseRetryAfter_Zero(t *testing.T) {
	// Zero seconds is invalid, falls through to default
	assert.Equal(t, 5*time.Minute, parseRetryAfter("0"))
}

func TestErrRateLimited_Error(t *testing.T) {
	err := &ErrRateLimited{URL: "https://example.com", RetryAfter: 5 * time.Minute}
	assert.Contains(t, err.Error(), "rate limited")
	assert.Contains(t, err.Error(), "5m0s")
}
