package query

type Query struct {
	// Pagination
	Page, PageSize int
	// Time range
	StartTime, EndTime int64
	// Order
	Order string
	Desc  bool
}

type EdgeQuery struct {
	Query
	// Condition fields
	Meta string
	Addr string
	RPC  string
}

type EdgeRPCQuery struct {
	Query
	// Condition fields
	Meta   string
	EdgeID uint64
	RPC    string
}

type EdgeDelete struct {
	EdgeID uint64
	Addr   string
}

type ServiceQuery struct {
	Query
	// Condition fields
	Service   string
	Addr      string
	RPC       string
	Topic     string
	ServiceID uint64
}

type ServiceDelete struct {
	ServiceID uint64
	Addr      string
}

type ServiceRPCQuery struct {
	Query
	// Condition fields
	Service   string
	ServiceID uint64
}

type ServiceTopicQuery struct {
	Query
	// Condition fields
	Service   string
	ServiceID uint64
}
