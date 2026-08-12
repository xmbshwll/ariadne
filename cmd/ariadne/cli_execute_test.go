package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errAllTargetsAlpha = errors.New("alpha boom")
	errAllTargetsBeta  = errors.New("beta boom")
)

func TestAllTargetsFailedErrorExposesEveryFailure(t *testing.T) {
	failures := map[string]error{
		"beta":  errAllTargetsBeta,
		"alpha": errAllTargetsAlpha,
	}

	err := allTargetsFailedError(2, failures)

	require.Error(t, err)
	assert.ErrorIs(t, err, errAllTargetSearchesFailed)
	assert.ErrorIs(t, err, errAllTargetsAlpha)
	assert.ErrorIs(t, err, errAllTargetsBeta)
	assert.Contains(t, err.Error(), "alpha, beta")
}

func TestAllTargetsFailedErrorReturnsNilWhenSomeTargetSucceeded(t *testing.T) {
	err := allTargetsFailedError(2, map[string]error{"alpha": errAllTargetsAlpha})

	assert.NoError(t, err)
}
