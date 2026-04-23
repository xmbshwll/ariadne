package ariadne

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookupServiceName(t *testing.T) {
	t.Parallel()

	service, ok := LookupServiceName(" apple-music ")
	assert.True(t, ok)
	assert.Equal(t, ServiceAppleMusic, service)

	service, ok = LookupServiceName("yt_music")
	assert.True(t, ok)
	assert.Equal(t, ServiceYouTubeMusic, service)

	service, ok = LookupServiceName("amazon")
	assert.True(t, ok)
	assert.Equal(t, ServiceAmazonMusic, service)

	service, ok = LookupServiceName("napster")
	assert.False(t, ok)
	assert.Empty(t, service)
}
