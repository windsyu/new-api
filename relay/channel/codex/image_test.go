package codex

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodexImagesHTTPRelay(t *testing.T) {
	for _, endpoint := range []struct {
		path    string
		mode    int
		model   string
		quality string
	}{
		{path: "generations", mode: relayconstant.RelayModeImagesGenerations, model: "gpt-image-2", quality: "low"},
		{path: "edits", mode: relayconstant.RelayModeImagesEdits, model: "gpt-image-2", quality: "low"},
		{path: "generations", mode: relayconstant.RelayModeImagesGenerations, model: "gpt-image-2.5-sunburst", quality: "xhigh"},
		{path: "edits", mode: relayconstant.RelayModeImagesEdits, model: "gpt-image-2.5-sunburst", quality: "max"},
		{path: "generations", mode: relayconstant.RelayModeImagesGenerations, model: "gpt-image-2.5-flare", quality: "max"},
		{path: "edits", mode: relayconstant.RelayModeImagesEdits, model: "gpt-image-2.5-flare", quality: "xhigh"},
	} {
		t.Run(endpoint.model+"/"+endpoint.path, func(t *testing.T) {
			const responseJSON = `{"created":123,"quality":"low","size":"1024x1024","output_format":"png","data":[{"b64_json":"aW1hZ2U="}],"usage":{"input_tokens":3,"output_tokens":4,"input_tokens_details":{"image_tokens":2,"text_tokens":1}}}`
			type receivedRequest struct {
				method, path string
				header       http.Header
				body         []byte
				err          error
			}
			received := make(chan receivedRequest, 1)
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				received <- receivedRequest{r.Method, r.URL.Path, r.Header.Clone(), body, err}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, responseJSON)
			}))
			defer upstream.Close()

			requestJSON := fmt.Sprintf(`{"model":%q,"prompt":"a red cube","quality":%q,"size":"1024x1024","output_format":"png","output_compression":0,"stream":false}`, endpoint.model, endpoint.quality)
			if endpoint.mode == relayconstant.RelayModeImagesEdits {
				requestJSON = fmt.Sprintf(`{"model":%q,"prompt":"make the cube blue","quality":%q,"size":"1024x1024","output_format":"png","output_compression":0,"stream":false,"images":[{"image_url":"data:image/png;base64,aW1hZ2U="}]}`, endpoint.model, endpoint.quality)
			}
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/"+endpoint.path, strings.NewReader(requestJSON))
			c.Request.Header.Set("Content-Type", "application/json; charset=utf-8")
			request, err := helper.GetAndValidOpenAIImageRequest(c, endpoint.mode)
			require.NoError(t, err)
			copied, err := common.DeepCopy(request)
			require.NoError(t, err)
			info := &relaycommon.RelayInfo{
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelType:    constant.ChannelTypeCodex,
					ChannelBaseUrl: upstream.URL,
					ApiKey:         `{"access_token":"test-access","account_id":"test-account"}`,
				},
				RelayMode:       endpoint.mode,
				OriginModelName: endpoint.model,
			}
			info.PriceData.UsePrice = true
			adaptor := &Adaptor{}
			converted, err := adaptor.ConvertImageRequest(c, info, *copied)
			require.NoError(t, err)
			body, err := common.Marshal(converted)
			require.NoError(t, err)
			response, err := adaptor.DoRequest(c, info, bytes.NewReader(body))
			require.NoError(t, err)
			resp, ok := response.(*http.Response)
			require.True(t, ok)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			got := <-received
			require.NoError(t, got.err)
			assert.Equal(t, http.MethodPost, got.method)
			assert.Equal(t, "/backend-api/codex/images/"+endpoint.path, got.path)
			assert.Equal(t, "Bearer test-access", got.header.Get("Authorization"))
			assert.Equal(t, "test-account", got.header.Get("chatgpt-account-id"))
			assert.Equal(t, "codex_cli_rs", got.header.Get("originator"))
			assert.Equal(t, "application/json", got.header.Get("Content-Type"))
			assert.Equal(t, "application/json", got.header.Get("Accept"))
			var expected map[string]any
			require.NoError(t, common.UnmarshalJsonStr(requestJSON, &expected))
			expected["n"] = float64(1)
			var actual map[string]any
			require.NoError(t, common.Unmarshal(got.body, &actual))
			assert.Equal(t, expected, actual)

			usageValue, apiErr := adaptor.DoResponse(c, resp, info)
			require.Nil(t, apiErr)
			usage, ok := usageValue.(*dto.Usage)
			require.True(t, ok)
			assert.Equal(t, 3, usage.PromptTokens)
			assert.Equal(t, 4, usage.CompletionTokens)
			assert.Equal(t, 7, usage.TotalTokens)
			assert.Equal(t, 2, usage.PromptTokensDetails.ImageTokens)
			assert.Equal(t, float64(1), info.PriceData.OtherRatios()["n"])
			assert.Equal(t, responseJSON, recorder.Body.String())
		})
	}
}

func TestCodexImagesStreamResponse(t *testing.T) {
	savedTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = savedTimeout })
	for _, endpoint := range []struct {
		mode  int
		event string
	}{
		{relayconstant.RelayModeImagesGenerations, "image_generation.completed"},
		{relayconstant.RelayModeImagesEdits, "image_edit.completed"},
	} {
		t.Run(endpoint.event, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", nil)
			body := fmt.Sprintf("data: {\"type\":%q,\"b64_json\":\"aW1hZ2U=\",\"usage\":{\"input_tokens\":3,\"output_tokens\":4}}\n\ndata: [DONE]\n\n", endpoint.event)
			resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(body))}
			info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}, RelayMode: endpoint.mode, IsStream: true}
			value, apiErr := (&Adaptor{}).DoResponse(c, resp, info)
			require.Nil(t, apiErr)
			usage, ok := value.(*dto.Usage)
			require.True(t, ok)
			assert.Equal(t, 7, usage.TotalTokens)
			assert.Contains(t, recorder.Body.String(), "event: "+endpoint.event)
			assert.Contains(t, recorder.Body.String(), `"b64_json":"aW1hZ2U="`)
			assert.Contains(t, recorder.Body.String(), "data: [DONE]")
		})
	}
}

func TestCodexImagesRejectErrorResponse(t *testing.T) {
	for _, body := range []string{`{"error":{"type":"invalid_request_error","message":"image rejected"}}`, "not JSON"} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
		resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
		info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}, RelayMode: relayconstant.RelayModeImagesGenerations}
		_, apiErr := (&Adaptor{}).DoResponse(c, resp, info)
		require.NotNil(t, apiErr)
	}
}
