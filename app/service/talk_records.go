package service

import (
	"context"
	"go-chat/app/cache"
	"go-chat/app/dao"
	"go-chat/app/entity"
	"go-chat/app/http/dto"
	"go-chat/app/model"
	"go-chat/app/pkg/jsonutil"
	"go-chat/app/pkg/slice"
	"go-chat/app/pkg/timeutil"
	"sort"
	"time"
)

type QueryTalkRecordsOpts struct {
	TalkType   int `json:"talk_type"`   // 对话类型
	UserId     int `json:"user_id"`     // 获取消息的用户
	ReceiverId int `json:"receiver_id"` // 接收者ID
	RecordId   int `json:"record_id"`   // 上次查询的最小消息ID
	Limit      int `json:"limit"`       // 数据行数
}

type QueryTalkRecordsItem struct {
	ID         int       `json:"id"`
	TalkType   int       `json:"talk_type"`
	MsgType    int       `json:"msg_type"`
	UserId     int       `json:"user_id"`
	ReceiverId int       `json:"receiver_id"`
	IsRevoke   int       `json:"is_revoke"`
	IsMark     int       `json:"is_mark"`
	IsRead     int       `json:"is_read"`
	QuoteId    int       `json:"quote_id"`
	WarnUsers  string    `json:"warn_users"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
	Nickname   string    `json:"nickname"`
	Avatar     string    `json:"avatar"`
}

type TalkRecordsService struct {
	*BaseService
	talkVoteCache      *cache.TalkVote
	talkRecordsVoteDao *dao.TalkRecordsVoteDao
}

func NewTalkRecordsService(baseService *BaseService, talkVoteCache *cache.TalkVote, talkRecordsVoteDao *dao.TalkRecordsVoteDao) *TalkRecordsService {
	return &TalkRecordsService{BaseService: baseService, talkVoteCache: talkVoteCache, talkRecordsVoteDao: talkRecordsVoteDao}
}

// GetTalkRecords 获取对话消息
func (s *TalkRecordsService) GetTalkRecords(ctx context.Context, query *QueryTalkRecordsOpts) ([]*dto.TalkRecordsItem, error) {
	var (
		err    error
		items  = make([]*QueryTalkRecordsItem, 0)
		fields = []string{
			"talk_records.id",
			"talk_records.talk_type",
			"talk_records.msg_type",
			"talk_records.user_id",
			"talk_records.receiver_id",
			"talk_records.is_revoke",
			"talk_records.content",
			"talk_records.created_at",
			"users.nickname",
			"users.avatar as avatar",
		}
	)

	tx := s.db.Table("talk_records")

	tx.Joins("left join users on talk_records.user_id = users.id")

	if query.RecordId > 0 {
		tx.Where("talk_records.id < ?", query.RecordId)
	}

	if query.TalkType == entity.PrivateChat {
		subWhere := s.db.Where("talk_records.user_id = ? and talk_records.receiver_id = ?", query.UserId, query.ReceiverId)
		subWhere.Or("talk_records.user_id = ? and talk_records.receiver_id = ?", query.ReceiverId, query.UserId)

		tx.Where(subWhere)
	} else {
		tx.Where("talk_records.receiver_id = ?", query.ReceiverId)
	}

	tx.Where("talk_records.talk_type = ?", query.TalkType)
	tx.Where("NOT EXISTS (SELECT 1 FROM `talk_records_delete` WHERE talk_records_delete.record_id = talk_records.id AND talk_records_delete.user_id = ? LIMIT 1)", query.UserId)
	tx.Select(fields).Order("talk_records.id desc").Limit(query.Limit)

	if err = tx.Scan(&items).Error; err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return make([]*dto.TalkRecordsItem, 0), err
	}

	return s.HandleTalkRecords(ctx, items)
}

// SearchTalkRecords 对话搜索消息
func (s *TalkRecordsService) SearchTalkRecords() {

}

func (s *TalkRecordsService) GetTalkRecord(ctx context.Context, recordId int64) (*dto.TalkRecordsItem, error) {
	var (
		err    error
		item   *QueryTalkRecordsItem
		fields = []string{
			"talk_records.id",
			"talk_records.talk_type",
			"talk_records.msg_type",
			"talk_records.user_id",
			"talk_records.receiver_id",
			"talk_records.is_revoke",
			"talk_records.content",
			"talk_records.created_at",
			"users.nickname",
			"users.avatar as avatar",
		}
	)

	tx := s.db.Table("talk_records")
	tx.Joins("left join users on talk_records.user_id = users.id")
	tx.Where("talk_records.id = ?", recordId)

	if err = tx.Select(fields).Take(&item).Error; err != nil {
		return nil, err
	}

	items := make([]*QueryTalkRecordsItem, 0)
	items = append(items, item)

	list, err := s.HandleTalkRecords(ctx, items)
	if err != nil {
		return nil, err
	}

	return list[0], nil
}

func (s *TalkRecordsService) HandleTalkRecords(ctx context.Context, items []*QueryTalkRecordsItem) ([]*dto.TalkRecordsItem, error) {
	var (
		files     []int
		codes     []int
		forwards  []int
		invites   []int
		votes     []int
		logins    []int
		locations []int

		fileItems     []*model.TalkRecordsFile
		codeItems     []*model.TalkRecordsCode
		forwardItems  []*model.TalkRecordsForward
		inviteItems   []*model.TalkRecordsInvite
		voteItems     []*model.TalkRecordsVote
		loginItems    []*model.TalkRecordsLogin
		locationItems []*model.TalkRecordsLocation
	)

	for _, item := range items {
		switch item.MsgType {
		case entity.MsgTypeFile:
			files = append(files, item.ID)
		case entity.MsgTypeForward:
			forwards = append(forwards, item.ID)
		case entity.MsgTypeCode:
			codes = append(codes, item.ID)
		case entity.MsgTypeVote:
			votes = append(votes, item.ID)
		case entity.MsgTypeGroupNotice:
		case entity.MsgTypeFriendApply:
		case entity.MsgTypeUserLogin:
			logins = append(logins, item.ID)
		case entity.MsgTypeGroupInvite:
			invites = append(invites, item.ID)
		case entity.MsgTypeLocation:
			locations = append(locations, item.ID)
		}
	}

	hashFiles := make(map[int]*model.TalkRecordsFile)
	if len(files) > 0 {
		s.db.Model(&model.TalkRecordsFile{}).Where("record_id in ?", files).Scan(&fileItems)
		for i := range fileItems {
			hashFiles[fileItems[i].RecordId] = fileItems[i]
		}
	}

	hashForwards := make(map[int]*model.TalkRecordsForward)
	if len(forwards) > 0 {
		s.db.Model(&model.TalkRecordsForward{}).Where("record_id in ?", forwards).Scan(&forwardItems)
		for i := range forwardItems {
			hashForwards[forwardItems[i].RecordId] = forwardItems[i]
		}
	}

	hashCodes := make(map[int]*model.TalkRecordsCode)
	if len(codes) > 0 {
		s.db.Model(&model.TalkRecordsCode{}).Where("record_id in ?", codes).Select("record_id", "code_lang", "code").Scan(&codeItems)
		for i := range codeItems {
			hashCodes[codeItems[i].RecordId] = codeItems[i]
		}
	}

	hashVotes := make(map[int]*model.TalkRecordsVote)
	if len(votes) > 0 {
		s.db.Model(&model.TalkRecordsVote{}).Where("record_id in ?", votes).Scan(&voteItems)
		for i := range voteItems {
			hashVotes[voteItems[i].RecordId] = voteItems[i]
		}
	}

	hashLogins := make(map[int]*model.TalkRecordsLogin)
	if len(logins) > 0 {
		s.db.Model(&model.TalkRecordsLogin{}).Where("record_id in ?", votes).Scan(&loginItems)
		for i := range loginItems {
			hashLogins[loginItems[i].RecordId] = loginItems[i]
		}
	}

	hashInvites := make(map[int]*model.TalkRecordsInvite)
	if len(invites) > 0 {
		s.db.Model(&model.TalkRecordsInvite{}).Where("record_id in ?", invites).Scan(&inviteItems)
		for i := range inviteItems {
			hashInvites[inviteItems[i].RecordId] = inviteItems[i]
		}
	}

	hashLocations := make(map[int]*model.TalkRecordsLocation)
	if len(locations) > 0 {
		s.db.Model(&model.TalkRecordsLocation{}).Where("record_id in ?", locations).Scan(&locationItems)
		for i := range locationItems {
			hashLocations[locationItems[i].RecordId] = locationItems[i]
		}
	}

	newItems := make([]*dto.TalkRecordsItem, 0, len(items))

	for _, item := range items {
		data := &dto.TalkRecordsItem{
			Id:         item.ID,
			TalkType:   item.TalkType,
			MsgType:    item.MsgType,
			UserId:     item.UserId,
			ReceiverId: item.ReceiverId,
			Nickname:   item.Nickname,
			Avatar:     item.Avatar,
			IsRevoke:   item.IsRevoke,
			IsMark:     item.IsMark,
			IsRead:     item.IsRead,
			Content:    item.Content,
			CreatedAt:  timeutil.FormatDatetime(item.CreatedAt),
		}

		switch item.MsgType {
		case entity.MsgTypeFile:
			if value, ok := hashFiles[item.ID]; ok {
				data.File = value
			}
		case entity.MsgTypeForward:
			if value, ok := hashForwards[item.ID]; ok {
				list := make([]map[string]interface{}, 0)

				_ = jsonutil.JsonDecode(value.Text, &list)

				data.Forward = map[string]interface{}{
					"num":  len(slice.ParseIds(value.RecordsId)),
					"list": list,
				}
			}
		case entity.MsgTypeCode:
			if value, ok := hashCodes[item.ID]; ok {
				data.CodeBlock = value
			}
		case entity.MsgTypeVote:
			if value, ok := hashVotes[item.ID]; ok {
				options := make(map[string]interface{})
				opts := make([]interface{}, 0)

				if err := jsonutil.JsonDecode(value.AnswerOption, &options); err == nil {
					arr := make([]string, 0, len(options))
					for k := range options {
						arr = append(arr, k)
					}

					sort.Strings(arr)

					for _, v := range arr {
						opts = append(opts, map[string]interface{}{
							"key":   v,
							"value": options[v],
						})
					}
				}

				users := make([]int, 0)
				if uids, err := s.talkRecordsVoteDao.GetVoteAnswerUser(ctx, value.Id); err == nil {
					users = uids
				}

				var statistics interface{}

				if res, err := s.talkRecordsVoteDao.GetVoteStatistics(ctx, value.Id); err != nil {
					statistics = map[string]interface{}{
						"count":   0,
						"options": map[string]int{},
					}
				} else {
					statistics = res
				}

				data.Vote = map[string]interface{}{
					"detail": map[string]interface{}{
						"id":            value.Id,
						"record_id":     value.RecordId,
						"title":         value.Title,
						"answer_mode":   value.AnswerMode,
						"status":        value.Status,
						"answer_option": opts,
						"answer_num":    value.AnswerNum,
						"answered_num":  value.AnsweredNum,
					},
					"statistics": statistics,
					"vote_users": users, // 已投票成员
				}
			}
		case entity.MsgTypeGroupNotice:
		case entity.MsgTypeFriendApply:
		case entity.MsgTypeUserLogin:
			if value, ok := hashLogins[item.ID]; ok {
				data.Login = value
			}
		case entity.MsgTypeGroupInvite:
			if value, ok := hashInvites[item.ID]; ok {
				m := map[string]interface{}{
					"type": value.Type,
					"operate_user": map[string]interface{}{
						"id":       value.OperateUserId,
						"nickname": "sf",
					},
					"users": map[string]interface{}{},
				}

				if value.Type == 1 || value.Type == 3 {
					var results []map[string]interface{}
					s.db.Model(&model.Users{}).Select("id", "nickname").Where("id in ?", slice.ParseIds(value.UserIds)).Scan(&results)
					m["users"] = results
				} else {
					m["users"] = m["operate_user"]
				}

				data.Invite = m
			}
		case entity.MsgTypeLocation:
			if value, ok := hashLocations[item.ID]; ok {
				data.Location = value
			}
		}

		newItems = append(newItems, data)
	}

	return newItems, nil
}
