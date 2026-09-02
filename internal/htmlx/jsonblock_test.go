package htmlx_test

import (
	"errors"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/htmlx"
)

var (
	errPageExtractionNotFound  = errors.New("page extraction not found")
	errPageExtractionMalformed = errors.New("page extraction malformed")
)

func TestFirstRegexpGroup(t *testing.T) {
	pattern := regexp.MustCompile(`<script>(.*?)</script>`)

	payload, err := htmlx.FirstRegexpGroup([]byte(`<script>payload</script>`), pattern, errPageExtractionNotFound)
	require.NoError(t, err)
	assert.Equal(t, []byte("payload"), payload)

	_, err = htmlx.FirstRegexpGroup([]byte(`<html></html>`), pattern, errPageExtractionNotFound)
	require.ErrorIs(t, err, errPageExtractionNotFound)
}

func TestDecodeJSONBlock(t *testing.T) {
	pattern := regexp.MustCompile(`<script>(.*?)</script>`)

	payload, err := htmlx.DecodeJSONBlock[struct {
		Name string `json:"name"`
	}](
		[]byte(`<script>{"name":"ariadne"}</script>`),
		pattern,
		errPageExtractionNotFound,
		"decode page json",
		errPageExtractionMalformed,
	)
	require.NoError(t, err)
	assert.Equal(t, "ariadne", payload.Name)

	_, err = htmlx.DecodeJSONBlock[struct{}](
		[]byte(`<script>{</script>`),
		pattern,
		errPageExtractionNotFound,
		"decode page json",
		errPageExtractionMalformed,
	)
	require.ErrorIs(t, err, errPageExtractionMalformed)
}
