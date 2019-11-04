package box

import "mime/multipart"

// CommonResp with code, message and data
type CommonResp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	RouteID string      `json:"route_id,omitempty"`
}

const FileSignature = "github.com/tuotoo/biu/box.File"

type File struct {
	multipart.File
	Header *multipart.FileHeader
}
