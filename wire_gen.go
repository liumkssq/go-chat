// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//+build !wireinject

package main

import (
	"context"
	"github.com/google/wire"
	"go-chat/app/cache"
	"go-chat/app/dao"
	"go-chat/app/http/handler"
	"go-chat/app/http/handler/api/v1"
	"go-chat/app/http/handler/open"
	"go-chat/app/http/handler/ws"
	"go-chat/app/http/router"
	"go-chat/app/pkg/filesystem"
	"go-chat/app/process"
	"go-chat/app/process/handle"
	"go-chat/app/service"
	"go-chat/provider"
)

import (
	_ "github.com/urfave/cli/v2"
	_ "go-chat/app/pkg/validation"
)

// Injectors from wire.go:

func Initialize(ctx context.Context) *provider.Providers {
	config := provider.NewConfig()
	client := provider.NewRedisClient(ctx, config)
	smsCodeCache := &cache.SmsCodeCache{
		Redis: client,
	}
	smsService := service.NewSmsService(smsCodeCache)
	db := provider.NewMySQLClient(config)
	baseDao := dao.NewBaseDao(db, client)
	usersDao := dao.NewUserDao(baseDao)
	userService := service.NewUserService(usersDao)
	common := v1.NewCommonHandler(config, smsService, userService)
	session := cache.NewSession(client)
	redisLock := cache.NewRedisLock(client)
	auth := v1.NewAuthHandler(config, userService, smsService, session, redisLock)
	user := v1.NewUserHandler(userService, smsService)
	baseService := service.NewBaseService(db, client)
	unreadTalkCache := cache.NewUnreadTalkCache(client)
	talkMessageForwardService := service.NewTalkMessageForwardService(baseService)
	lastMessage := cache.NewLastMessage(client)
	talkVote := cache.NewTalkVote(client)
	talkRecordsVoteDao := dao.NewTalkRecordsVoteDao(baseDao, talkVote)
	groupMemberDao := dao.NewGroupMemberDao(baseDao)
	sidServer := cache.NewSid(client)
	wsClientSession := cache.NewWsClientSession(client, config, sidServer)
	filesystemFilesystem := filesystem.NewFilesystem(config)
	talkMessageService := service.NewTalkMessageService(baseService, config, unreadTalkCache, talkMessageForwardService, lastMessage, talkRecordsVoteDao, groupMemberDao, sidServer, wsClientSession, filesystemFilesystem)
	groupMemberService := service.NewGroupMemberService(groupMemberDao)
	talkService := service.NewTalkService(baseService, groupMemberService)
	fileSplitUploadDao := dao.NewFileSplitUploadDao(baseDao)
	splitUploadService := service.NewSplitUploadService(baseService, fileSplitUploadDao, config, filesystemFilesystem)
	talkMessage := v1.NewTalkMessageHandler(talkMessageService, talkService, talkRecordsVoteDao, talkMessageForwardService, splitUploadService)
	talkListDao := dao.NewTalkListDao(baseDao)
	talkListService := service.NewTalkListService(baseService, talkListDao)
	usersFriendsDao := dao.NewUsersFriends(baseDao)
	contactService := service.NewContactService(baseService, usersFriendsDao)
	talk := v1.NewTalkHandler(talkService, talkListService, redisLock, userService, wsClientSession, lastMessage, unreadTalkCache, contactService)
	talkRecordsService := service.NewTalkRecordsService(baseService, talkVote, talkRecordsVoteDao, filesystemFilesystem, groupMemberDao)
	talkRecords := v1.NewTalkRecordsHandler(talkRecordsService)
	talkRecordsDao := &dao.TalkRecordsDao{
		BaseDao: baseDao,
	}
	download := v1.NewDownloadHandler(filesystemFilesystem, talkRecordsDao, groupMemberService)
	emoticonDao := dao.NewEmoticonDao(baseDao)
	emoticonService := service.NewEmoticonService(baseService, emoticonDao)
	emoticon := v1.NewEmoticonHandler(emoticonService, filesystemFilesystem, redisLock)
	upload := v1.NewUploadHandler(config, filesystemFilesystem, splitUploadService)
	index := open.NewIndexHandler(client)
	clientService := service.NewClientService(wsClientSession)
	room := cache.NewGroupRoom(client)
	defaultWebSocket := ws.NewDefaultWebSocket(client, config, clientService, room, groupMemberService)
	groupDao := dao.NewGroupDao(baseDao)
	groupService := service.NewGroupService(baseService, groupDao, groupMemberDao)
	group := v1.NewGroupHandler(groupService, groupMemberService, talkListService, redisLock, contactService, userService)
	groupNoticeDao := &dao.GroupNoticeDao{
		BaseDao: baseDao,
	}
	groupNoticeService := service.NewGroupNoticeService(groupNoticeDao)
	groupNotice := v1.NewGroupNoticeHandler(groupNoticeService, groupMemberService)
	contact := v1.NewContactHandler(contactService, wsClientSession, userService)
	contactApplyService := service.NewContactsApplyService(baseService)
	contactApply := v1.NewContactsApplyHandler(contactApplyService, userService)
	handlerHandler := &handler.Handler{
		Common:           common,
		Auth:             auth,
		User:             user,
		TalkMessage:      talkMessage,
		Talk:             talk,
		TalkRecords:      talkRecords,
		Download:         download,
		Emoticon:         emoticon,
		Upload:           upload,
		Index:            index,
		DefaultWebSocket: defaultWebSocket,
		Group:            group,
		GroupNotice:      groupNotice,
		Contact:          contact,
		ContactsApply:    contactApply,
	}
	engine := router.NewRouter(config, handlerHandler, session)
	server := provider.NewHttpServer(config, engine)
	processServer := process.NewServerRun(config, sidServer)
	subscribeConsume := handle.NewSubscribeConsume(config, wsClientSession, room, talkRecordsService, contactService)
	wsSubscribe := process.NewWsSubscribe(client, config, subscribeConsume)
	heartbeat := process.NewImHeartbeat()
	clearGarbage := process.NewClearGarbage(client, redisLock, sidServer)
	processProcess := process.NewProcessManage(processServer, wsSubscribe, heartbeat, clearGarbage)
	providers := &provider.Providers{
		Config:     config,
		HttpServer: server,
		Process:    processProcess,
	}
	return providers
}

// wire.go:

var providerSet = wire.NewSet(provider.NewConfig, provider.NewMySQLClient, provider.NewRedisClient, provider.NewHttpClient, provider.NewHttpServer, router.NewRouter, filesystem.NewFilesystem, cache.NewSession, cache.NewSid, cache.NewUnreadTalkCache, cache.NewRedisLock, cache.NewWsClientSession, cache.NewLastMessage, cache.NewTalkVote, cache.NewGroupRoom, wire.Struct(new(cache.SmsCodeCache), "*"), dao.NewBaseDao, dao.NewUsersFriends, dao.NewGroupMemberDao, dao.NewUserDao, dao.NewGroupDao, wire.Struct(new(dao.TalkRecordsDao), "*"), wire.Struct(new(dao.TalkRecordsCodeDao), "*"), wire.Struct(new(dao.TalkRecordsLoginDao), "*"), wire.Struct(new(dao.TalkRecordsFileDao), "*"), wire.Struct(new(dao.GroupNoticeDao), "*"), dao.NewTalkListDao, dao.NewEmoticonDao, dao.NewTalkRecordsVoteDao, dao.NewFileSplitUploadDao, service.NewBaseService, service.NewUserService, service.NewSmsService, service.NewTalkService, service.NewTalkMessageService, service.NewClientService, service.NewGroupService, service.NewGroupMemberService, service.NewGroupNoticeService, service.NewTalkListService, service.NewTalkMessageForwardService, service.NewEmoticonService, service.NewTalkRecordsService, service.NewContactService, service.NewContactsApplyService, service.NewSplitUploadService, v1.NewAuthHandler, v1.NewCommonHandler, v1.NewUserHandler, v1.NewContactHandler, v1.NewContactsApplyHandler, v1.NewGroupHandler, v1.NewGroupNoticeHandler, v1.NewTalkHandler, v1.NewTalkMessageHandler, v1.NewUploadHandler, v1.NewDownloadHandler, v1.NewEmoticonHandler, v1.NewTalkRecordsHandler, open.NewIndexHandler, ws.NewDefaultWebSocket, wire.Struct(new(handler.Handler), "*"), wire.Struct(new(provider.Providers), "*"), process.NewWsSubscribe, process.NewServerRun, process.NewProcessManage, process.NewImHeartbeat, process.NewClearGarbage, handle.NewSubscribeConsume)
