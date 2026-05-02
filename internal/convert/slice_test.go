package convert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloneSlice_contracts(t *testing.T) {
	identity := func(v int) int { return v }

	// nil → nil
	assert.Nil(t, CloneSlice[int](nil))
	assert.Nil(t, TranslateSlice[int, int](nil, identity))
	assert.Nil(t, TranslateNonEmptySlice[int, int](nil, identity))
	assert.Nil(t, TranslateNonEmptySlice[int, int]([]int{}, identity))

	// non-nil empty → cloned empty (non-nil)
	empty := []int{}
	clonedEmpty := CloneSlice(empty)
	assert.NotNil(t, clonedEmpty)
	assert.Empty(t, clonedEmpty)

	// non-nil non-empty → deep copy
	values := []int{1, 2, 3}
	cloned := CloneSlice(values)
	values[0] = 999
	assert.Equal(t, []int{1, 2, 3}, cloned)
}

func TestCloneStrings_contracts(t *testing.T) {
	assert.Nil(t, CloneStrings(nil))

	empty := []string{}
	cloned := CloneStrings(empty)
	assert.NotNil(t, cloned)
	assert.Empty(t, cloned)

	original := []string{"a", "b"}
	cloned = CloneStrings(original)
	original[0] = "mutated"
	assert.Equal(t, []string{"a", "b"}, cloned)
}

func TestTranslateSlice_preservesNil(t *testing.T) {
	double := func(v int) int { return v * 2 }

	assert.Nil(t, TranslateSlice[int, int](nil, double))

	empty := TranslateSlice([]int{}, double)
	assert.NotNil(t, empty)
	assert.Empty(t, empty)

	translated := TranslateSlice([]int{1, 2}, double)
	assert.Equal(t, []int{2, 4}, translated)
}

func TestTranslateSliceToEmpty_neverNil(t *testing.T) {
	double := func(v int) int { return v * 2 }

	fromNil := TranslateSliceToEmpty[int, int](nil, double)
	assert.NotNil(t, fromNil)
	assert.Empty(t, fromNil)

	fromEmpty := TranslateSliceToEmpty([]int{}, double)
	assert.NotNil(t, fromEmpty)
	assert.Empty(t, fromEmpty)

	fromValues := TranslateSliceToEmpty([]int{1, 2}, double)
	assert.Equal(t, []int{2, 4}, fromValues)
}

func TestTranslateNonEmptySlice_nilOrEmptyReturnsNil(t *testing.T) {
	double := func(v int) int { return v * 2 }

	assert.Nil(t, TranslateNonEmptySlice[int, int](nil, double))
	assert.Nil(t, TranslateNonEmptySlice[int, int]([]int{}, double))

	translated := TranslateNonEmptySlice([]int{1, 2}, double)
	assert.Equal(t, []int{2, 4}, translated)
}
