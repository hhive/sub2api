package vendorhall

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseListParamsAcceptsSupportedWindowsAndBoundsPagination(t *testing.T) {
	for _, window := range []string{"3h", "24h", "3d"} {
		params, err := ParseListParams(url.Values{"window": {window}, "page": {"2"}, "page_size": {"100"}})
		require.NoError(t, err)
		require.Equal(t, window, params.Window)
		require.Equal(t, 2, params.Page)
		require.Equal(t, 100, params.PageSize)
	}

	params, err := ParseListParams(url.Values{"page_size": {"999"}})
	require.NoError(t, err)
	require.Equal(t, 100, params.PageSize)
}

func TestParseListParamsRejectsUnknownWindowSortAndOrder(t *testing.T) {
	for _, values := range []url.Values{
		{"window": {"6h"}},
		{"window": {"7d"}},
		{"window": {"30d"}},
		{"sort_by": {"sql_injection"}},
		{"sort_order": {"sideways"}},
		{"status": {"mystery"}},
	} {
		_, err := ParseListParams(values)
		require.Error(t, err)
	}
}
