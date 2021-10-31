package service

import (
	"context"
	"errors"
	"fmt"
	"go-chat/app/cache"
	"go-chat/app/entity"
	"go-chat/app/http/request"
	"go-chat/app/model"
	"go-chat/app/pkg/jsonutil"
	"go-chat/app/pkg/strutil"
	"go-chat/config"
	"gorm.io/gorm"
	"time"
)

type TalkMessageService struct {
	*BaseService
	config             *config.Config
	groupMemberService *GroupMemberService
	unreadTalkCache    *cache.UnreadTalkCache
}

func NewTalkMessageService(
	base *BaseService,
	config *config.Config,
	groupMemberService *GroupMemberService,
) *TalkMessageService {
	return &TalkMessageService{
		BaseService:        base,
		config:             config,
		groupMemberService: groupMemberService,
	}
}

func (s *TalkMessageService) SendTextMessage(ctx context.Context, uid int, params *request.TextMessageRequest) error {
	record := &model.TalkRecords{
		TalkType:   params.TalkType,
		MsgType:    entity.MsgTypeText,
		UserId:     uid,
		ReceiverId: params.ReceiverId,
		Content:    params.Text,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	result := s.db.Create(record)
	if result.Error != nil {
		return result.Error
	}

	s.handle(ctx, record, map[string]string{
		"text": strutil.MtSubstr(&record.Content, 0, 30),
	})

	return nil
}

func (s *TalkMessageService) SendCodeMessage(ctx context.Context, uid int, params *request.CodeMessageRequest) error {
	var (
		err    error
		record = &model.TalkRecords{
			TalkType:   params.TalkType,
			MsgType:    entity.MsgTypeCode,
			UserId:     uid,
			ReceiverId: params.ReceiverId,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
	)

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err = s.db.Create(record).Error; err != nil {
			return err
		}

		if err = s.db.Create(&model.TalkRecordsCode{
			RecordId:  record.ID,
			UserId:    uid,
			CodeLang:  params.Lang,
			Code:      params.Code,
			CreatedAt: time.Now(),
		}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	s.handle(ctx, record, map[string]string{
		"text": "[代码消息]",
	})

	return nil
}

func (s *TalkMessageService) SendImageMessage(ctx context.Context, uid int, params *request.ImageMessageRequest) error {
	return nil
}

func (s *TalkMessageService) SendFileMessage(ctx context.Context, params *request.FileMessageRequest) {

}

func (s *TalkMessageService) SendCardMessage(ctx context.Context, params *request.CardMessageRequest) {

}

func (s *TalkMessageService) SendVoteMessage(ctx context.Context, uid int, params *request.VoteMessageRequest) error {

	var (
		err    error
		record = &model.TalkRecords{
			TalkType:   entity.GroupChat,
			MsgType:    entity.MsgTypeVote,
			UserId:     uid,
			ReceiverId: params.ReceiverId,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
	)

	options := make(map[string]string)
	for i, value := range params.Options {
		options[fmt.Sprintf("%c", 65+i)] = value
	}

	num := s.groupMemberService.GetGroupMemberCount(params.ReceiverId)

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err = s.db.Create(record).Error; err != nil {
			return err
		}

		if err = s.db.Create(&model.TalkRecordsVote{
			RecordId:     record.ID,
			UserId:       uid,
			Title:        params.Title,
			AnswerMode:   params.Mode,
			AnswerOption: jsonutil.JsonEncode(options),
			AnswerNum:    int(num),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	s.handle(ctx, record, map[string]string{
		"text": "[投票消息]",
	})

	return nil
}

func (s *TalkMessageService) SendEmoticonMessage(ctx context.Context, uid int, params *request.EmoticonMessageRequest) error {
	var (
		err      error
		emoticon model.EmoticonItem
		record   = &model.TalkRecords{
			TalkType:   entity.GroupChat,
			MsgType:    entity.MsgTypeFile,
			UserId:     uid,
			ReceiverId: params.ReceiverId,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
	)

	if err = s.db.Model(&model.EmoticonItem{}).Where("id = ?", params.EmoticonId).First(&emoticon).Error; err != nil {
		return err
	}

	if emoticon.UserId > 0 && emoticon.UserId != uid {
		return errors.New("表情包不存在！")
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err = s.db.Create(record).Error; err != nil {
			return err
		}

		if err = s.db.Create(&model.TalkRecordsFile{
			RecordId:     record.ID,
			UserId:       uid,
			FileSource:   2,
			FileType:     entity.GetMediaType(emoticon.FileSuffix),
			OriginalName: "图片表情",
			FileSuffix:   emoticon.FileSuffix,
			FileSize:     emoticon.FileSize,
			SaveDir:      emoticon.Url,
			CreatedAt:    time.Now(),
		}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	s.handle(ctx, record, map[string]string{
		"text": "[图片消息]",
	})

	return nil
}

func (s *TalkMessageService) SendForwardMessage(ctx context.Context, params *request.ForwardMessageRequest) {

}

// SendLocationMessage 发送位置消息
func (s *TalkMessageService) SendLocationMessage(ctx context.Context, uid int, params *request.LocationMessageRequest) error {

	var (
		err    error
		record = &model.TalkRecords{
			TalkType:   params.TalkType,
			MsgType:    entity.MsgTypeLocation,
			UserId:     uid,
			ReceiverId: params.ReceiverId,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
	)

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err = s.db.Create(record).Error; err != nil {
			return err
		}

		if err = s.db.Create(&model.TalkRecordsLocation{
			RecordId:  record.ID,
			UserId:    uid,
			Longitude: params.Longitude,
			Latitude:  params.Latitude,
			CreatedAt: time.Now(),
		}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	s.handle(ctx, record, map[string]string{
		"text": "[位置消息]",
	})

	return nil
}

func (s *TalkMessageService) handle(ctx context.Context, record *model.TalkRecords, opts map[string]string) {

	if record.TalkType == entity.PrivateChat {
		s.unreadTalkCache.Increment(ctx, record.UserId, record.ReceiverId)
	}

	// 推送消息至 redis
}
