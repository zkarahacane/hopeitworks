package service

// clampPagination constrains page and perPage to valid ranges.
// page defaults to 1, perPage defaults to 20, capped at 100.
func clampPagination(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

// paginationToLimitOffset converts clamped page/perPage into limit and offset
// values suitable for database queries.
func paginationToLimitOffset(page, perPage int) (limit int32, offset int32) {
	page, perPage = clampPagination(page, perPage)
	return int32(perPage), int32((page - 1) * perPage)
}
