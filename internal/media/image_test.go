package media

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPagination(t *testing.T) {
	tt := []struct{
		p      Pagination
		offset uint
		limit  uint
	}{
		{Pagination{1, 10}, 0, 10},
		{Pagination{0, 10}, 0, 10},
		{Pagination{2, 5}, 5, 5},
		{Pagination{10, 26}, 234, 26},
		{Pagination{10, 0}, 225, 25},
	}

	for _, tc := range tt {
		t.Run(fmt.Sprintf("%d:%d", tc.p.Page, tc.p.PerPage), func(t *testing.T) {
			assert.Equal(t, int(tc.offset), int(tc.p.Offset()))
			assert.Equal(t, int(tc.limit), int(tc.p.Limit()))
		})
	}
}