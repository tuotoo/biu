package ctx

// CommonResp with code, message and data
type CommonResp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	RouteID string      `json:"route_id,omitempty"`
}
