package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodexImageModelEndpoint(t *testing.T) {
	assert.True(t, IsImageGenerationModel("gpt-image-2"))
	endpoints := GetEndpointTypesByChannelType(constant.ChannelTypeCodex, "gpt-image-2")
	require.NotEmpty(t, endpoints)
	assert.Equal(t, constant.EndpointTypeImageGeneration, endpoints[0])
	assert.False(t, IsImageGenerationModel("gpt-6-astra"))
}
