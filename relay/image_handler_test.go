package relay

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodexImagesRejectMultipartBeforePassthrough(t *testing.T) {
	saved := model_setting.GetGlobalSettings().PassThroughRequestEnabled
	t.Cleanup(func() { model_setting.GetGlobalSettings().PassThroughRequestEnabled = saved })
	for _, tt := range []struct {
		name            string
		global, channel bool
	}{
		{name: "converted request"},
		{name: "channel passthrough", channel: true},
		{name: "global passthrough", global: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			model_setting.GetGlobalSettings().PassThroughRequestEnabled = tt.global
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			require.NoError(t, writer.WriteField("model", "gpt-image-2"))
			require.NoError(t, writer.WriteField("prompt", "make the cube blue"))
			part, err := writer.CreateFormFile("image", "cube.png")
			require.NoError(t, err)
			_, err = part.Write([]byte("test image"))
			require.NoError(t, err)
			require.NoError(t, writer.Close())
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
			c.Request.Header.Set("Content-Type", writer.FormDataContentType())
			common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeCodex)
			common.SetContextKey(c, constant.ContextKeyChannelSetting, dto.ChannelSettings{PassThroughBodyEnabled: tt.channel})
			request, err := helper.GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)
			require.NoError(t, err)
			info := &relaycommon.RelayInfo{Request: request, OriginModelName: request.Model, RelayMode: relayconstant.RelayModeImagesEdits}
			apiErr := ImageHelper(c, info)
			require.NotNil(t, apiErr)
			assert.Equal(t, http.StatusUnsupportedMediaType, apiErr.StatusCode)
			assert.Equal(t, types.ErrorCodeInvalidRequest, apiErr.GetErrorCode())
			assert.True(t, types.IsSkipRetryError(apiErr))
			assert.Contains(t, apiErr.Error(), "application/json")
		})
	}
}
