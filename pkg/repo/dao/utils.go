package dao

type Query struct {
	// Pagination
	Page, PageSize int
	// Time range
	StartTime, EndTime int64
	// Order
	Order string
	Desc  bool
}
