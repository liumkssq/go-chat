package talk

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go-chat/internal/entity"
	"go-chat/internal/http/internal/request"
	"go-chat/internal/http/internal/response"
	"go-chat/internal/pkg/filesystem"
	"go-chat/internal/pkg/jwtutil"
	"go-chat/internal/pkg/sliceutil"
	"go-chat/internal/service"
)

type Records struct {
	service            *service.TalkRecordsService
	groupMemberService *service.GroupMemberService
	fileSystem         *filesystem.Filesystem
}

func NewTalkRecordsHandler(service *service.TalkRecordsService, groupMemberService *service.GroupMemberService, fileSystem *filesystem.Filesystem) *Records {
	return &Records{
		service:            service,
		groupMemberService: groupMemberService,
		fileSystem:         fileSystem,
	}
}

// GetRecords 获取会话记录
func (c *Records) GetRecords(ctx *gin.Context) {
	params := &request.TalkRecordsRequest{}
	if err := ctx.ShouldBindQuery(params); err != nil {
		response.InvalidParams(ctx, err)
		return
	}

	records, err := c.service.GetTalkRecords(ctx, &service.QueryTalkRecordsOpts{
		TalkType:   params.TalkType,
		UserId:     jwtutil.GetUid(ctx),
		ReceiverId: params.ReceiverId,
		RecordId:   params.RecordId,
		Limit:      params.Limit,
	})

	if err != nil {
		response.BusinessError(ctx, err)
		return
	}

	rid := 0
	if length := len(records); length > 0 {
		rid = records[length-1].Id
	}

	response.Success(ctx, gin.H{
		"limit":     params.Limit,
		"record_id": rid,
		"rows":      records,
	})
}

// SearchHistoryRecords 查询下会话记录
func (c *Records) SearchHistoryRecords(ctx *gin.Context) {
	params := &request.TalkRecordsRequest{}
	if err := ctx.ShouldBindQuery(params); err != nil {
		response.InvalidParams(ctx, err)
		return
	}

	m := []int{
		entity.MsgTypeText,
		entity.MsgTypeFile,
		entity.MsgTypeForward,
		entity.MsgTypeCode,
		entity.MsgTypeVote,
	}

	if sliceutil.InInt(params.MsgType, m) {
		m = []int{params.MsgType}
	}

	records, err := c.service.GetTalkRecords(ctx, &service.QueryTalkRecordsOpts{
		TalkType:   params.TalkType,
		MsgType:    m,
		UserId:     jwtutil.GetUid(ctx),
		ReceiverId: params.ReceiverId,
		RecordId:   params.RecordId,
		Limit:      params.Limit,
	})

	if err != nil {
		response.BusinessError(ctx, err)
		return
	}

	rid := 0
	if length := len(records); length > 0 {
		rid = records[length-1].Id
	}

	response.Success(ctx, gin.H{
		"limit":     params.Limit,
		"record_id": rid,
		"rows":      records,
	})
}

// GetForwardRecords 获取转发记录
func (c *Records) GetForwardRecords(ctx *gin.Context) {
	params := &request.TalkForwardRecordsRequest{}
	if err := ctx.ShouldBind(params); err != nil {
		response.InvalidParams(ctx, err)
		return
	}

	records, err := c.service.GetForwardRecords(ctx.Request.Context(), jwtutil.GetUid(ctx), int64(params.RecordId))
	if err != nil {
		response.BusinessError(ctx, err)
		return
	}

	response.Success(ctx, gin.H{
		"rows": records,
	})
}

// Download 聊天文件下载
func (c *Records) Download(ctx *gin.Context) {
	params := &request.DownloadChatFileRequest{}
	if err := ctx.ShouldBindQuery(params); err != nil {
		response.InvalidParams(ctx, err)
		return
	}

	resp, err := c.service.Dao().FindFileRecord(ctx.Request.Context(), params.RecordId)
	if err != nil {
		return
	}

	uid := jwtutil.GetUid(ctx)
	if uid != resp.Record.UserId {
		if resp.Record.TalkType == entity.ChatPrivateMode {
			if resp.Record.ReceiverId != uid {
				response.Unauthorized(ctx, "无访问权限！")
				return
			}
		} else {
			if !c.groupMemberService.Dao().IsMember(resp.Record.ReceiverId, uid, false) {
				response.Unauthorized(ctx, "无访问权限！")
				return
			}
		}
	}

	switch resp.FileInfo.Drive {
	case entity.FileDriveLocal:
		ctx.FileAttachment(c.fileSystem.Local.Path(resp.FileInfo.Path), resp.FileInfo.OriginalName)
	case entity.FileDriveCos:
		ctx.Redirect(http.StatusFound, c.fileSystem.Cos.PrivateUrl(resp.FileInfo.Path, 60))
	default:
		response.BusinessError(ctx, "未知文件驱动类型")
	}
}
