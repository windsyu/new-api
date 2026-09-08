package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageRelayInvalidCountReturnsBadRequest(t *testing.T) {
	for _, path := range []string{"/v1/images/generations", "/v1/images/edits"} {
		t.Run(path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			body := fmt.Sprintf(`{"model":"gpt-image-2","prompt":"a cube","n":%d}`, dto.MaxImageN+1)
			c.Request = httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")
			Relay(c, types.RelayFormatOpenAIImage)
			assert.Equal(t, http.StatusBadRequest, recorder.Code)
			var response struct {
				Error types.OpenAIError `json:"error"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			assert.Equal(t, string(types.ErrorCodeInvalidRequest), response.Error.Code)
			assert.Contains(t, response.Error.Message, fmt.Sprintf("n must be an integer between 1 and %d", dto.MaxImageN))
		})
	}
}
