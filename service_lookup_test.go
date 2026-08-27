package ariadne_test

import (
	"testing"

	ariadne "github.com/xmbshwll/ariadne"

	"github.com/stretchr/testify/assert"
)

func TestLookupServiceName(t *testing.T) {
	t.Parallel()

	service, ok := ariadne.LookupServiceName(" apple-music ")
	assert.True(t, ok)
	assert.Equal(t, ariadne.ServiceAppleMusic, service)

	service, ok = ariadne.LookupServiceName("yt_music")
	assert.True(t, ok)
	assert.Equal(t, ariadne.ServiceYouTubeMusic, service)

	service, ok = ariadne.LookupServiceName("amazon")
	assert.True(t, ok)
	assert.Equal(t, ariadne.ServiceAmazonMusic, service)

	service, ok = ariadne.LookupServiceName("napster")
	assert.False(t, ok)
	assert.Empty(t, service)
}
