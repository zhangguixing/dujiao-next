package uploadhttp

import (
	"mime/multipart"

	uploadcontract "github.com/dujiao-next/internal/modules/upload/contract"
	"github.com/dujiao-next/internal/platform/http/ginutil"
	"github.com/dujiao-next/internal/platform/http/response"
	"github.com/gin-gonic/gin"
)

type UserFileUploader interface {
	SaveFileWithMeta(*multipart.FileHeader, string) (*uploadcontract.Result, error)
}
type UserHandler struct{ uploader UserFileUploader }

func NewUserHandler(uploader UserFileUploader) *UserHandler {
	if uploader == nil {
		panic("upload user handler: required dependency is nil")
	}
	return &UserHandler{uploader: uploader}
}
func (h *UserHandler) UploadManualRechargeProof(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		ginutil.RespondError(c, response.CodeBadRequest, "error.file_missing", nil)
		return
	}
	result, err := h.uploader.SaveFileWithMeta(file, "manual_recharge")
	if err != nil {
		if isUploadValidationError(err) {
			ginutil.RespondErrorWithMsg(c, response.CodeBadRequest, err.Error(), nil)
		} else {
			ginutil.RespondError(c, response.CodeInternal, "error.upload_failed", err)
		}
		return
	}
	response.Success(c, gin.H{"url": result.URL, "filename": result.Filename, "size": result.Size})
}
