package operations

import (
	"testing"

	_ "github.com/go-openapi/analysis/internal/antest"
	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/require"
)

var _ Provider = mockOperationsProvider{}

type mockOperationsProvider struct {
	give map[string]map[string]*spec.Operation
}

func (m mockOperationsProvider) Operations() map[string]map[string]*spec.Operation {
	return m.give
}

func TestGatherOperations(t *testing.T) {
	t.Run("should handle empty operation IDs", func(_ *testing.T) {
		m := mockOperationsProvider{
			give: map[string]map[string]*spec.Operation{
				"get": {
					"/pth1": {
						OperationProps: spec.OperationProps{
							ID:          "",
							Description: "ok",
						},
					},
				},
			},
		}

		res := GatherOperations(m, nil)
		require.Contains(t, res, "GetPth1")
	})

	t.Run("should handle duplicate operation IDs (when spec validation is skipped)", func(_ *testing.T) {
		m := mockOperationsProvider{
			give: map[string]map[string]*spec.Operation{
				"get": {
					"/pth1": {
						OperationProps: spec.OperationProps{
							ID:          "id1",
							Description: "ok",
						},
					},
				},
				"post": {
					"/pth2": {
						OperationProps: spec.OperationProps{
							ID:          "id1",
							Description: "ok",
						},
					},
				},
			},
		}

		res := GatherOperations(m, nil)
		require.Contains(t, res, "id1")
		require.NotContains(t, res, "GetPth1")
		require.Contains(t, res, "PostPth2")
	})
}
