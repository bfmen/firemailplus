package api

import (
	"firemail/internal/api/generated"
	"firemail/internal/handlers"
	"firemail/internal/services"

	"github.com/gin-gonic/gin"
)

var _ generated.ServerInterface = (*Server)(nil)

// Server adapts generated OpenAPI handlers to the existing handwritten
// handler layer. Business logic remains in services and handlers, not in
// generated code.
type Server struct {
	handler          *handlers.Handler
	attachmentRouter *handlers.AttachmentHandler
}

// NewServer creates a generated server adapter around the current handler set.
func NewServer(handler *handlers.Handler) *Server {
	attachmentStorageConfig := &services.AttachmentStorageConfig{
		BaseDir:      "attachments",
		MaxFileSize:  25 * 1024 * 1024,
		CompressText: false,
		CreateDirs:   true,
		ChecksumType: "md5",
	}
	attachmentStorage := services.NewLocalFileStorage(attachmentStorageConfig)
	attachmentService := services.NewAttachmentService(
		handler.GetDB(),
		attachmentStorage,
		handler.GetProviderFactory(),
	)

	return &Server{
		handler:          handler,
		attachmentRouter: handlers.NewAttachmentHandler(attachmentService, handler.GetDB()),
	}
}

// RegisterHandlers registers generated OpenAPI routes against a Gin router.
// The main binary keeps the existing handwritten registration until route-group
// replacement is performed deliberately, but this is the generated integration
// boundary for cutover tests and future route replacement.
func RegisterHandlers(router gin.IRouter, handler *handlers.Handler) {
	generated.RegisterHandlers(router, NewServer(handler))
}

func (s *Server) ListEmailAccounts(c *gin.Context)        { s.handler.GetEmailAccounts(c) }
func (s *Server) CreateEmailAccount(c *gin.Context)       { s.handler.CreateEmailAccount(c) }
func (s *Server) BatchDeleteEmailAccounts(c *gin.Context) { s.handler.BatchDeleteEmailAccounts(c) }
func (s *Server) BatchMarkAccountsAsRead(c *gin.Context)  { s.handler.BatchMarkAccountsAsRead(c) }
func (s *Server) BatchSyncEmailAccounts(c *gin.Context)   { s.handler.BatchSyncEmailAccounts(c) }
func (s *Server) CreateCustomEmailAccount(c *gin.Context) { s.handler.CreateCustomEmailAccount(c) }
func (s *Server) DeleteEmailAccount(c *gin.Context, _ generated.IdPath) {
	s.handler.DeleteEmailAccount(c)
}
func (s *Server) GetEmailAccount(c *gin.Context, _ generated.IdPath) { s.handler.GetEmailAccount(c) }
func (s *Server) UpdateEmailAccount(c *gin.Context, _ generated.IdPath) {
	s.handler.UpdateEmailAccount(c)
}
func (s *Server) MarkAccountAsRead(c *gin.Context, _ generated.IdPath) {
	s.handler.MarkAccountAsRead(c)
}
func (s *Server) SyncEmailAccount(c *gin.Context, _ generated.IdPath) { s.handler.SyncEmailAccount(c) }
func (s *Server) TestEmailAccount(c *gin.Context, _ generated.IdPath) { s.handler.TestEmailAccount(c) }

func (s *Server) UploadAttachment(c *gin.Context) { s.attachmentRouter.UploadAttachment(c) }
func (s *Server) DownloadAttachment(c *gin.Context, _ generated.IdPath) {
	s.attachmentRouter.DownloadAttachment(c)
}
func (s *Server) ForceDownloadAttachment(c *gin.Context, _ generated.IdPath) {
	s.attachmentRouter.ForceDownloadAttachment(c)
}
func (s *Server) PreviewAttachment(c *gin.Context, _ generated.IdPath) {
	s.attachmentRouter.PreviewAttachment(c)
}
func (s *Server) GetAttachmentDownloadProgress(c *gin.Context, _ generated.IdPath) {
	s.attachmentRouter.GetDownloadProgress(c)
}

func (s *Server) Login(c *gin.Context)          { s.handler.Login(c) }
func (s *Server) Logout(c *gin.Context)         { s.handler.Logout(c) }
func (s *Server) GetCurrentUser(c *gin.Context) { s.handler.GetCurrentUser(c) }
func (s *Server) RefreshToken(c *gin.Context)   { s.handler.RefreshToken(c) }
func (s *Server) ChangePassword(c *gin.Context) { s.handler.ChangePassword(c) }
func (s *Server) UpdateProfile(c *gin.Context)  { s.handler.UpdateProfile(c) }

func (s *Server) ListBackups(c *gin.Context)    { s.handler.ListBackups(c) }
func (s *Server) CreateBackup(c *gin.Context)   { s.handler.CreateBackup(c) }
func (s *Server) RestoreBackup(c *gin.Context)  { s.handler.RestoreBackup(c) }
func (s *Server) DeleteBackup(c *gin.Context)   { s.handler.DeleteBackup(c) }
func (s *Server) ValidateBackup(c *gin.Context) { s.handler.ValidateBackup(c) }
func (s *Server) CleanupBackups(c *gin.Context) { s.handler.CleanupOldBackups(c) }
func (s *Server) GetSoftDeleteStats(c *gin.Context) {
	s.handler.GetSoftDeleteStats(c)
}
func (s *Server) CleanupSoftDeletes(c *gin.Context) { s.handler.CleanupExpiredSoftDeletes(c) }
func (s *Server) RestoreSoftDeleted(c *gin.Context, _ generated.RestoreSoftDeletedParamsTable, _ generated.IdPath) {
	s.handler.RestoreSoftDeleted(c)
}
func (s *Server) PermanentlyDeleteSoftDeleted(c *gin.Context, _ generated.PermanentlyDeleteSoftDeletedParamsTable, _ generated.IdPath) {
	s.handler.PermanentlyDelete(c)
}

func (s *Server) DeduplicateAccount(c *gin.Context, _ generated.IdPath) {
	s.handler.DeduplicateAccount(c)
}
func (s *Server) DeduplicateUser(c *gin.Context) { s.handler.DeduplicateUser(c) }
func (s *Server) GetDeduplicationReport(c *gin.Context, _ generated.IdPath) {
	s.handler.GetDeduplicationReport(c)
}
func (s *Server) ScheduleDeduplication(c *gin.Context, _ generated.IdPath) {
	s.handler.ScheduleDeduplication(c)
}
func (s *Server) CancelScheduledDeduplication(c *gin.Context, _ generated.IdPath) {
	s.handler.CancelScheduledDeduplication(c)
}
func (s *Server) GetDeduplicationStats(c *gin.Context, _ generated.IdPath) {
	s.handler.GetDeduplicationStats(c)
}

func (s *Server) ListEmails(c *gin.Context, _ generated.ListEmailsParams) { s.handler.GetEmails(c) }
func (s *Server) BatchEmailOperations(c *gin.Context)                     { s.handler.BatchEmailOperations(c) }
func (s *Server) SearchEmails(c *gin.Context, _ generated.SearchEmailsParams) {
	s.handler.SearchEmails(c)
}
func (s *Server) SendEmail(c *gin.Context)                       { s.handler.SendEmail(c) }
func (s *Server) SendBulkEmails(c *gin.Context)                  { s.handler.SendBulkEmails(c) }
func (s *Server) GetSendStatus(c *gin.Context, _ string)         { s.handler.GetSendStatus(c) }
func (s *Server) ResendEmail(c *gin.Context, _ string)           { s.handler.ResendEmail(c) }
func (s *Server) SaveDraft(c *gin.Context)                       { s.handler.SaveDraft(c) }
func (s *Server) GetDraft(c *gin.Context, _ generated.IdPath)    { s.handler.GetDraft(c) }
func (s *Server) UpdateDraft(c *gin.Context, _ generated.IdPath) { s.handler.UpdateDraft(c) }
func (s *Server) DeleteDraft(c *gin.Context, _ generated.IdPath) { s.handler.DeleteDraft(c) }
func (s *Server) ListDrafts(c *gin.Context, _ generated.ListDraftsParams) {
	s.handler.ListDrafts(c)
}
func (s *Server) CreateTemplate(c *gin.Context) { s.handler.CreateTemplate(c) }
func (s *Server) GetTemplate(c *gin.Context, _ generated.IdPath) {
	s.handler.GetTemplate(c)
}
func (s *Server) UpdateTemplate(c *gin.Context, _ generated.IdPath) {
	s.handler.UpdateTemplate(c)
}
func (s *Server) DeleteTemplate(c *gin.Context, _ generated.IdPath) {
	s.handler.DeleteTemplate(c)
}
func (s *Server) ListTemplates(c *gin.Context, _ generated.ListTemplatesParams) {
	s.handler.ListTemplates(c)
}
func (s *Server) DeleteEmail(c *gin.Context, _ generated.IdPath)  { s.handler.DeleteEmail(c) }
func (s *Server) GetEmail(c *gin.Context, _ generated.IdPath)     { s.handler.GetEmail(c) }
func (s *Server) UpdateEmail(c *gin.Context, _ generated.IdPath)  { s.handler.UpdateEmail(c) }
func (s *Server) ArchiveEmail(c *gin.Context, _ generated.IdPath) { s.handler.ArchiveEmail(c) }
func (s *Server) ListEmailAttachments(c *gin.Context, _ generated.IdPath) {
	s.attachmentRouter.GetEmailAttachments(c)
}
func (s *Server) DownloadEmailAttachments(c *gin.Context, _ generated.IdPath) {
	s.attachmentRouter.DownloadEmailAttachments(c)
}
func (s *Server) ForwardEmail(c *gin.Context, _ generated.IdPath)    { s.handler.ForwardEmail(c) }
func (s *Server) MoveEmail(c *gin.Context, _ generated.IdPath)       { s.handler.MoveEmail(c) }
func (s *Server) MarkEmailAsRead(c *gin.Context, _ generated.IdPath) { s.handler.MarkEmailAsRead(c) }
func (s *Server) ReplyEmail(c *gin.Context, _ generated.IdPath)      { s.handler.ReplyEmail(c) }
func (s *Server) ReplyAllEmail(c *gin.Context, _ generated.IdPath)   { s.handler.ReplyAllEmail(c) }
func (s *Server) ToggleEmailStar(c *gin.Context, _ generated.IdPath) { s.handler.ToggleEmailStar(c) }
func (s *Server) MarkEmailAsUnread(c *gin.Context, _ generated.IdPath) {
	s.handler.MarkEmailAsUnread(c)
}

func (s *Server) ListFolders(c *gin.Context, _ generated.ListFoldersParams) { s.handler.GetFolders(c) }
func (s *Server) CreateFolder(c *gin.Context)                               { s.handler.CreateFolder(c) }
func (s *Server) DeleteFolder(c *gin.Context, _ generated.IdPath)           { s.handler.DeleteFolder(c) }
func (s *Server) GetFolder(c *gin.Context, _ generated.IdPath)              { s.handler.GetFolder(c) }
func (s *Server) UpdateFolder(c *gin.Context, _ generated.IdPath)           { s.handler.UpdateFolder(c) }
func (s *Server) MarkFolderAsRead(c *gin.Context, _ generated.IdPath)       { s.handler.MarkFolderAsRead(c) }
func (s *Server) SyncFolder(c *gin.Context, _ generated.IdPath)             { s.handler.SyncFolder(c) }

func (s *Server) ListEmailGroups(c *gin.Context)                      { s.handler.GetEmailGroups(c) }
func (s *Server) CreateEmailGroup(c *gin.Context)                     { s.handler.CreateEmailGroup(c) }
func (s *Server) ReorderEmailGroups(c *gin.Context)                   { s.handler.ReorderEmailGroups(c) }
func (s *Server) DeleteEmailGroup(c *gin.Context, _ generated.IdPath) { s.handler.DeleteEmailGroup(c) }
func (s *Server) UpdateEmailGroup(c *gin.Context, _ generated.IdPath) { s.handler.UpdateEmailGroup(c) }
func (s *Server) SetDefaultEmailGroup(c *gin.Context, _ generated.IdPath) {
	s.handler.SetDefaultEmailGroup(c)
}

func (s *Server) CreateOAuthAccount(c *gin.Context) { s.handler.CreateOAuth2Account(c) }
func (s *Server) InitGmailOAuth(c *gin.Context, _ generated.InitGmailOAuthParams) {
	s.handler.InitGmailOAuth(c)
}
func (s *Server) CreateManualOAuthAccount(c *gin.Context) {
	s.handler.CreateManualOAuth2Account(c)
}
func (s *Server) InitOutlookOAuth(c *gin.Context, _ generated.InitOutlookOAuthParams) {
	s.handler.InitOutlookOAuth(c)
}
func (s *Server) HandleOAuthCallback(c *gin.Context, _ generated.HandleOAuthCallbackParamsProvider, _ generated.HandleOAuthCallbackParams) {
	s.handler.HandleOAuth2Callback(c)
}

func (s *Server) ListProviders(c *gin.Context) { s.handler.GetProviders(c) }
func (s *Server) DetectProvider(c *gin.Context, _ generated.DetectProviderParams) {
	s.handler.DetectProvider(c)
}

func (s *Server) StreamSSE(c *gin.Context, _ generated.StreamSSEParams) { s.handler.HandleSSE(c) }
func (s *Server) StreamSSEEvents(c *gin.Context, _ generated.StreamSSEEventsParams) {
	s.handler.HandleSSE(c)
}
func (s *Server) GetSSEStats(c *gin.Context)      { s.handler.GetSSEStats(c) }
func (s *Server) SendTestSSEEvent(c *gin.Context) { s.handler.SendTestEvent(c) }
func (s *Server) GetHealth(c *gin.Context)        { s.handler.HealthCheck(c) }
