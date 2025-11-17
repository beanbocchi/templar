package response

import "github.com/beanbocchi/templar/internal/model"

type CommonResponse struct {
	Data  any          `json:"data,omitempty"`
	Error *model.Error `json:"error"`
}
