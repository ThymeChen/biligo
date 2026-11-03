package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/fdcs99/biligo/internal/applog"
	"github.com/fdcs99/biligo/internal/biliauth"
	"github.com/fdcs99/biligo/internal/biliticket"
	"github.com/fdcs99/biligo/internal/bws"
	"github.com/fdcs99/biligo/internal/bwsrunner"
	"github.com/fdcs99/biligo/internal/events"
	"github.com/fdcs99/biligo/internal/model"
	"github.com/fdcs99/biligo/internal/notify"
	proxynet "github.com/fdcs99/biligo/internal/proxy"
	"github.com/fdcs99/biligo/internal/runner"
	"github.com/fdcs99/biligo/internal/store"
	"github.com/fdcs99/biligo/internal/userauth"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	store     *store.Store
	auth      *biliauth.Client
	ticket    *biliticket.Client
	bws       *bws.Client
	hub       *events.Hub
	runner    *runner.Manager
	bwsRunner *bwsrunner.Manager
	user      *userauth.Manager
	logger    *applog.Logger
	notify    notify.Sender
}

type RouterOptions struct {
	WebFS              fs.FS
	NotificationSender notify.Sender
}

type Runtime struct {
	Router    *gin.Engine
	Runner    *runner.Manager
	BWSRunner *bwsrunner.Manager
}

const maxAccountImportJSONBytes = 2 * 1024 * 1024

var pullKuaidailiDPS = proxynet.PullKuaidailiDPS
var testProxyNode = proxynet.TestNode

type RouterOption func(*RouterOptions)

func WithWebFS(webFS fs.FS) RouterOption {
	return func(options *RouterOptions) {
		options.WebFS = webFS
	}
}

func WithNotificationSender(sender notify.Sender) RouterOption {
	return func(options *RouterOptions) {
		options.NotificationSender = sender
	}
}

func NewRouter(store *store.Store, user *userauth.Manager, logger *applog.Logger, opts ...RouterOption) *gin.Engine {
	return NewRuntime(store, user, logger, opts...).Router
}

func NewRuntime(store *store.Store, user *userauth.Manager, logger *applog.Logger, opts ...RouterOption) Runtime {
	gin.SetMode(gin.ReleaseMode)
	var options RouterOptions
	for _, opt := range opts {
		opt(&options)
	}

	router := gin.New()
	router.Use(devCORS())
	router.Use(recovery(logger))
	if user == nil {
		user = userauth.NewManager(24 * time.Hour)
	}

	hub := events.NewHub(logger)
	hub.SetTaskOwnerLookup(func(ctx context.Context, kind string, taskID int64) (int64, error) {
		if kind == "bws" {
			return store.GetBWSTaskOwnerUID(ctx, taskID)
		}
		return store.GetTaskOwnerUID(ctx, taskID)
	})
	ticket := biliticket.NewClient(nil)
	bwsClient := bws.NewClient(nil)
	notificationSender := options.NotificationSender
	if notificationSender == nil {
		notificationSender = notify.NewHTTPSender(nil)
	}
	runnerManager := runner.NewManager(store, ticket, hub)
	runnerManager.SetNotifier(notify.NewService(store, notificationSender, hub))
	bwsRunnerManager := bwsrunner.NewManager(store, bwsClient, hub)
	handler := &Handler{
		store:     store,
		auth:      biliauth.NewClient(nil),
		ticket:    ticket,
		bws:       bwsClient,
		hub:       hub,
		runner:    runnerManager,
		bwsRunner: bwsRunnerManager,
		user:      user,
		logger:    logger,
		notify:    notificationSender,
	}
	api := router.Group("/api")
	{
		api.GET("/health", handler.health)
		api.POST("/user-auth/qr/start", handler.userQRStart)
		api.POST("/user-auth/qr/poll", handler.userQRPoll)

		protected := api.Group("")
		protected.Use(handler.requireUserAuth())
		{
			protected.GET("/user-auth/session", handler.userSession)
			protected.POST("/user-auth/logout", handler.userLogout)

			protected.GET("/events", handler.events)
			protected.GET("/auth/session", handler.sessionSummary)

			protected.GET("/accounts", handler.listAccounts)
			protected.POST("/accounts", handler.createAccount)
			protected.POST("/accounts/export", handler.exportAccounts)
			protected.POST("/accounts/import", handler.importAccounts)
			protected.PUT("/accounts/:id", handler.updateAccount)
			protected.GET("/accounts/:id/cookie", handler.accountCookie)
			protected.POST("/accounts/:id/verify", handler.verifyAccount)
			protected.DELETE("/accounts/:id", handler.deleteAccount)

			protected.GET("/ticket-projects/history", handler.listTicketProjectHistory)
			protected.POST("/ticket-projects/fetch", handler.fetchTicketProject)
			protected.POST("/ticket-projects/account-context", handler.fetchTicketAccountContext)

			protected.GET("/tasks", handler.listTasks)
			protected.POST("/tasks", handler.createTask)
			protected.PUT("/tasks/:id", handler.updateTask)
			protected.DELETE("/tasks/:id", handler.deleteTask)
			protected.POST("/tasks/:id/dispatch", handler.dispatchTask)
			protected.POST("/tasks/:id/pause", handler.pauseTask)
			protected.GET("/tasks/:id/logs", handler.listTaskLogs)

			protected.POST("/bws/info", handler.fetchBWSInfo)
			protected.GET("/bws/my-reservations", handler.fetchBWSMyReservations)
			protected.GET("/bws/tasks", handler.listBWSTasks)
			protected.POST("/bws/tasks", handler.createBWSTask)
			protected.PUT("/bws/tasks/:id", handler.updateBWSTask)
			protected.DELETE("/bws/tasks/:id", handler.deleteBWSTask)
			protected.POST("/bws/tasks/:id/dispatch", handler.dispatchBWSTask)
			protected.POST("/bws/tasks/:id/pause", handler.pauseBWSTask)
			protected.GET("/bws/tasks/:id/logs", handler.listBWSTaskLogs)
			protected.GET("/bws/logs", handler.listBWSLogs)

			protected.GET("/notifications", handler.listNotifications)
			protected.POST("/notifications", handler.createNotification)
			protected.PUT("/notifications/:id", handler.updateNotification)
			protected.DELETE("/notifications/:id", handler.deleteNotification)
			protected.POST("/notifications/:id/test", handler.testNotification)
			protected.POST("/notifications/:id/enable", handler.enableNotification)
			protected.POST("/notifications/:id/disable", handler.disableNotification)

			protected.GET("/proxy-groups", handler.listProxyGroups)
			protected.POST("/proxy-groups", handler.createProxyGroup)
			protected.PUT("/proxy-groups/:id", handler.updateProxyGroup)
			protected.DELETE("/proxy-groups/:id", handler.deleteProxyGroup)
			protected.GET("/proxy-groups/:id/nodes", handler.listProxyNodes)
			protected.POST("/proxy-groups/:id/nodes", handler.createProxyNode)
			protected.PUT("/proxy-nodes/:id", handler.updateProxyNode)
			protected.DELETE("/proxy-nodes/:id", handler.deleteProxyNode)
			protected.POST("/proxy-groups/:id/test", handler.testProxyGroup)
			protected.POST("/proxy-groups/:id/pull", handler.pullProxyGroup)

			protected.GET("/logs", handler.listLogs)
		}
	}

	if options.WebFS != nil {
		registerWebUI(router, options.WebFS)
	}

	return Runtime{
		Router:    router,
		Runner:    runnerManager,
		BWSRunner: bwsRunnerManager,
	}
}

func registerWebUI(router *gin.Engine, webFS fs.FS) {
	httpFS := http.FS(webFS)
	serveIndex := func(c *gin.Context) {
		data, err := fs.ReadFile(webFS, "index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "web index not found")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	}

	router.GET("/", serveIndex)
	router.GET("/assets/*filepath", func(c *gin.Context) {
		c.FileFromFS(path.Clean(c.Request.URL.Path), httpFS)
	})
	router.NoRoute(func(c *gin.Context) {
		if c.Request.URL.Path == "/api" || strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "接口不存在"})
			return
		}

		name := strings.TrimPrefix(path.Clean(c.Request.URL.Path), "/")
		if webFileExists(webFS, name) {
			c.FileFromFS(name, httpFS)
			return
		}
		serveIndex(c)
	})
}

func webFileExists(webFS fs.FS, name string) bool {
	if name == "" || name == "." {
		return false
	}
	info, err := fs.Stat(webFS, name)
	return err == nil && !info.IsDir()
}

func devCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func recovery(logger *applog.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, recovered any) {
		if logger != nil {
			logger.Errorf("HTTP 请求处理异常：%v", recovered)
		}
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}

func (h *Handler) health(c *gin.Context) {
	status := model.Health{
		Status:   "ok",
		Database: "ok",
		Time:     time.Now().Format(time.RFC3339),
	}
	if err := h.store.Ping(c.Request.Context()); err != nil {
		status.Status = "error"
		status.Database = err.Error()
		c.JSON(http.StatusServiceUnavailable, status)
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h *Handler) requireUserAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := userTokenFromRequest(c)
		user, expiresAt, ok := h.user.Validate(token)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "登录已失效，请重新登录"})
			return
		}
		c.Set("userID", user.UID)
		c.Set("username", user.Username)
		c.Set("userAuthExpiresAt", expiresAt)
		c.Next()
	}
}

func (h *Handler) userSession(c *gin.Context) {
	expiresAt, _ := c.Get("userAuthExpiresAt")
	value, ok := expiresAt.(time.Time)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "登录已失效"})
		return
	}
	c.JSON(http.StatusOK, model.UserAuthResponse{
		ExpiresAt: value.Format(time.RFC3339),
		User: &model.UserSession{
			UID:      ownerUIDFromContext(c),
			Username: usernameFromContext(c),
		},
	})
}

func (h *Handler) userLogout(c *gin.Context) {
	revoked := h.user.Revoke(userTokenFromRequest(c))
	if h.logger != nil {
		if revoked {
			h.logger.Infof("用户退出登录成功，客户端 %s。", clientIP(c))
		} else {
			h.logger.Warnf("用户退出登录请求未匹配有效 token，客户端 %s。", clientIP(c))
		}
	}
	c.Status(http.StatusNoContent)
}

func userTokenFromRequest(c *gin.Context) string {
	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[len("bearer "):])
	}
	return strings.TrimSpace(c.Query("token"))
}

func ownerUIDFromContext(c *gin.Context) int64 {
	if value, ok := c.Get("userID"); ok {
		if uid, ok := value.(int64); ok {
			return uid
		}
	}
	return 0
}

func usernameFromContext(c *gin.Context) string {
	if value, ok := c.Get("username"); ok {
		if name, ok := value.(string); ok {
			return name
		}
	}
	return ""
}

func (h *Handler) requireResourceOwner(c *gin.Context, lookup func(context.Context, int64) (int64, error), id int64) bool {
	ownerUID, err := lookup(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return false
	}
	if ownerUID != ownerUIDFromContext(c) {
		c.JSON(http.StatusNotFound, gin.H{"error": "资源不存在"})
		return false
	}
	return true
}

func clientIP(c *gin.Context) string {
	if ip := strings.TrimSpace(c.ClientIP()); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return firstNonEmpty(c.Request.RemoteAddr, "unknown")
}

func (h *Handler) events(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	id, ch := h.hub.SubscribeForOwner(ownerUIDFromContext(c))
	defer h.hub.Unsubscribe(id)

	ownerUID := ownerUIDFromContext(c)
	tasks, taskErr := h.store.ListTasks(c.Request.Context(), ownerUID)
	logs, logErr := h.store.ListTaskLogs(c.Request.Context(), ownerUID, 0)
	bwsTasks, bwsTaskErr := h.store.ListBWSTasks(c.Request.Context(), ownerUID)
	bwsLogs, bwsLogErr := h.store.ListBWSTaskLogs(c.Request.Context(), ownerUID, 0)
	if taskErr == nil && logErr == nil && bwsTaskErr == nil && bwsLogErr == nil {
		writeSSE(c, "snapshot", model.EventSnapshot{Tasks: tasks, Logs: logs, BWSTasks: bwsTasks, BWSTaskLogs: bwsLogs})
	}

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event.Name, event.Data)
			c.Writer.Flush()
		case <-ticker.C:
			writeSSE(c, "heartbeat", gin.H{"time": time.Now().Format(time.RFC3339)})
		}
	}
}

func (h *Handler) sessionSummary(c *gin.Context) {
	summary, err := h.store.SessionSummary(c.Request.Context(), ownerUIDFromContext(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) userQRStart(c *gin.Context) {
	result, err := h.auth.StartQRLogin(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, model.QRLoginStartResponse{
		OK:               result.OK,
		LoginURL:         result.LoginURL,
		QRCodeKey:        result.QRCodeKey,
		QRImageDataURL:   result.QRImageDataURL,
		ExpiresInSeconds: result.ExpiresInSeconds,
		NextAction:       "show_qr_and_confirm_scan",
	})
}

func (h *Handler) userQRPoll(c *gin.Context) {
	var input model.QRLoginPollInput
	if !bindJSON(c, &input) {
		return
	}
	if strings.TrimSpace(input.QRCodeKey) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "qrcodeKey 不能为空"})
		return
	}

	result, err := h.auth.PollQRLogin(c.Request.Context(), input.QRCodeKey)
	if err != nil {
		respondError(c, err)
		return
	}

	response := model.QRLoginPollResponse{
		OK:       result.OK,
		Status:   result.Status,
		Message:  result.Message,
		Code:     result.Code,
		Username: result.Username,
	}
	if result.Status != "confirmed" || result.Cookie == "" || result.UID <= 0 {
		c.JSON(http.StatusOK, response)
		return
	}

	if err := h.store.UpsertUser(c.Request.Context(), result.UID, result.Username, result.Cookie); err != nil {
		respondError(c, err)
		return
	}
	account, err := h.store.GetAccountByOwner(c.Request.Context(), result.UID)
	if errors.Is(err, sql.ErrNoRows) {
		account, err = h.store.CreateAccountWithStatus(c.Request.Context(), result.UID, model.AccountInput{
			Name:   firstNonEmpty(result.Username, "Bilibili 账号"),
			Cookie: result.Cookie,
		}, "logged_in")
	} else if err == nil {
		account, err = h.store.UpdateAccount(c.Request.Context(), account.ID, model.AccountInput{
			Name:   firstNonEmpty(account.Name, result.Username, "Bilibili 账号"),
			Cookie: result.Cookie,
			Note:   account.Note,
		})
		if err == nil {
			account, err = h.store.UpdateAccountStatus(c.Request.Context(), account.ID, "logged_in")
		}
	}
	if err != nil {
		respondError(c, err)
		return
	}

	token, expiresAt, err := h.user.Login(userauth.User{UID: result.UID, Username: result.Username})
	if err != nil {
		respondError(c, err)
		return
	}
	if h.logger != nil {
		h.logger.Infof("用户登录成功：UID %d（%s），客户端 %s，token 有效期至 %s。", result.UID, result.Username, clientIP(c), expiresAt.Format(time.RFC3339))
	}

	response.Token = token
	response.User = &model.UserSession{UID: result.UID, Username: result.Username}
	response.ExpiresAt = expiresAt.Format(time.RFC3339)
	response.Account = &account
	c.JSON(http.StatusOK, response)
}

func (h *Handler) listAccounts(c *gin.Context) {
	accounts, err := h.store.ListAccounts(c.Request.Context(), ownerUIDFromContext(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, accounts)
}

func (h *Handler) createAccount(c *gin.Context) {
	var input model.AccountInput
	if !bindJSON(c, &input) || !requireName(c, input.Name, "账号名称不能为空") {
		return
	}
	account, err := h.store.CreateAccount(c.Request.Context(), ownerUIDFromContext(c), input)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, account)
}

func (h *Handler) exportAccounts(c *gin.Context) {
	var input model.AccountBatchExportInput
	if !bindJSON(c, &input) || !validateAccountBatchExportInput(c, input) {
		return
	}
	accounts, err := h.store.ExportAccounts(c.Request.Context(), ownerUIDFromContext(c), input.AccountIDs)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.AccountBatchExportResponse{
		Version:    1,
		ExportedAt: time.Now().Format(time.RFC3339),
		Accounts:   accounts,
	})
}

func (h *Handler) importAccounts(c *gin.Context) {
	accountInputs, ok := decodeAccountImportInput(c)
	if !ok || !validateAccountImportInput(c, accountInputs) {
		return
	}
	accounts, err := h.store.ImportAccounts(c.Request.Context(), ownerUIDFromContext(c), accountInputs)
	if err != nil {
		respondError(c, err)
		return
	}
	response := model.AccountImportResponse{
		Imported: len(accounts),
		Accounts: accounts,
	}
	c.JSON(http.StatusCreated, response)
}

func (h *Handler) updateAccount(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireResourceOwner(c, h.store.GetAccountOwnerUID, id) {
		return
	}
	var input model.AccountInput
	if !bindJSON(c, &input) || !requireName(c, input.Name, "账号名称不能为空") {
		return
	}
	account, err := h.store.UpdateAccount(c.Request.Context(), id, input)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, account)
}

func (h *Handler) accountCookie(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireResourceOwner(c, h.store.GetAccountOwnerUID, id) {
		return
	}

	account, cookie, err := h.store.GetAccountCookie(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	if strings.TrimSpace(cookie) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "账号未保存 Cookie"})
		return
	}

	c.JSON(http.StatusOK, model.AccountCookieResponse{
		AccountID:     account.ID,
		Cookie:        cookie,
		CookiePreview: account.CookiePreview,
	})
}

func (h *Handler) verifyAccount(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireResourceOwner(c, h.store.GetAccountOwnerUID, id) {
		return
	}

	_, cookie, err := h.store.GetAccountCookie(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	if strings.TrimSpace(cookie) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "账号未保存 Cookie"})
		return
	}

	_, username, loggedIn, err := h.auth.VerifyCookie(c.Request.Context(), cookie)
	if err != nil {
		respondError(c, err)
		return
	}

	status := "login_invalid"
	message := "Cookie 未通过 Bilibili 登录态验证。"
	if loggedIn {
		status = "logged_in"
		message = "登录态验证成功。"
	}
	account, err := h.store.UpdateAccountStatus(c.Request.Context(), id, status)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, model.AccountVerifyResponse{
		OK:        true,
		LoggedIn:  loggedIn,
		AccountID: id,
		Username:  username,
		Message:   message,
		Account:   &account,
	})
}

func (h *Handler) deleteAccount(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireResourceOwner(c, h.store.GetAccountOwnerUID, id) {
		return
	}
	if err := h.store.DeleteAccount(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listTicketProjectHistory(c *gin.Context) {
	history, err := h.store.ListTicketProjectHistory(c.Request.Context(), ownerUIDFromContext(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, history)
}

func (h *Handler) fetchTicketProject(c *gin.Context) {
	var input model.TicketProjectFetchInput
	if !bindJSON(c, &input) {
		return
	}
	if strings.TrimSpace(input.ProjectInput) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "项目 ID 不能为空"})
		return
	}
	if _, err := biliticket.ExtractProjectID(input.ProjectInput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	project, err := h.ticket.FetchProject(c.Request.Context(), input.ProjectInput, "")
	if err != nil {
		respondError(c, err)
		return
	}
	if err := h.store.UpsertTicketProjectHistory(c.Request.Context(), ownerUIDFromContext(c), project); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, project)
}

func (h *Handler) fetchTicketAccountContext(c *gin.Context) {
	var input model.TicketAccountContextInput
	if !bindJSON(c, &input) {
		return
	}
	if input.AccountID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先选择账号"})
		return
	}
	if strings.TrimSpace(input.ProjectInput) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "项目 ID 不能为空"})
		return
	}
	projectID, err := biliticket.ExtractProjectID(input.ProjectInput)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !h.requireResourceOwner(c, h.store.GetAccountOwnerUID, input.AccountID) {
		return
	}
	_, cookie, err := h.store.GetAccountCookie(c.Request.Context(), input.AccountID)
	if err != nil {
		respondError(c, err)
		return
	}
	if strings.TrimSpace(cookie) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "所选账号未保存 Cookie"})
		return
	}

	context, err := h.ticket.FetchAccountContext(c.Request.Context(), projectID, cookie)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, context)
}

func (h *Handler) listNotifications(c *gin.Context) {
	notifications, err := h.store.ListNotifications(c.Request.Context(), ownerUIDFromContext(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, notifications)
}

func (h *Handler) createNotification(c *gin.Context) {
	var input model.NotificationInput
	if !bindJSON(c, &input) || !validateNotificationInput(c, input) {
		return
	}
	notification, err := h.store.CreateNotification(c.Request.Context(), ownerUIDFromContext(c), input)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, notification)
}

func (h *Handler) updateNotification(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireResourceOwner(c, h.store.GetNotificationOwnerUID, id) {
		return
	}
	var input model.NotificationInput
	if !bindJSON(c, &input) || !validateNotificationInput(c, input) {
		return
	}
	notification, err := h.store.UpdateNotification(c.Request.Context(), id, input)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, notification)
}

func (h *Handler) deleteNotification(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireResourceOwner(c, h.store.GetNotificationOwnerUID, id) {
		return
	}
	if err := h.store.DeleteNotification(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) testNotification(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireResourceOwner(c, h.store.GetNotificationOwnerUID, id) {
		return
	}
	notification, err := h.store.GetNotification(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	sender := h.notify
	if sender == nil {
		sender = notify.NewHTTPSender(nil)
	}
	sendCtx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	err = sender.Send(sendCtx, notification, "Biligo 通知测试", "这是一条 Biligo 通知接口测试消息。")
	cancel()
	status := "success"
	message := "测试推送已发送。"
	if err != nil {
		status = "error"
		message = err.Error()
	}
	notification, updateErr := h.store.SetNotificationTestResult(c.Request.Context(), id, status, message)
	if updateErr != nil {
		respondError(c, updateErr)
		return
	}
	c.JSON(http.StatusOK, notification)
}

func (h *Handler) enableNotification(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireResourceOwner(c, h.store.GetNotificationOwnerUID, id) {
		return
	}
	notification, err := h.store.SetNotificationEnabled(c.Request.Context(), id, true)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, notification)
}

func (h *Handler) disableNotification(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireResourceOwner(c, h.store.GetNotificationOwnerUID, id) {
		return
	}
	notification, err := h.store.SetNotificationEnabled(c.Request.Context(), id, false)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, notification)
}

func (h *Handler) listProxyGroups(c *gin.Context) {
	groups, err := h.store.ListProxyGroups(c.Request.Context(), ownerUIDFromContext(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, groups)
}

func (h *Handler) createProxyGroup(c *gin.Context) {
	var input model.ProxyGroupInput
	if !bindJSON(c, &input) || !validateProxyGroupInput(c, input) {
		return
	}
	group, err := h.store.CreateProxyGroup(c.Request.Context(), ownerUIDFromContext(c), input)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, group)
}

func (h *Handler) updateProxyGroup(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireProxyGroupEditable(c, id) {
		return
	}
	var input model.ProxyGroupInput
	if !bindJSON(c, &input) || !validateProxyGroupInput(c, input) {
		return
	}
	group, err := h.store.UpdateProxyGroup(c.Request.Context(), id, input)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, group)
}

func (h *Handler) deleteProxyGroup(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireProxyGroupEditable(c, id) {
		return
	}
	if err := h.store.DeleteProxyGroup(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listProxyNodes(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireResourceOwner(c, h.store.GetProxyGroupOwnerUID, id) {
		return
	}
	nodes, err := h.store.ListProxyNodes(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, nodes)
}

func (h *Handler) createProxyNode(c *gin.Context) {
	groupID, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireProxyGroupEditable(c, groupID) {
		return
	}
	group, err := h.store.GetProxyGroup(c.Request.Context(), groupID)
	if err != nil {
		respondError(c, err)
		return
	}
	if group.Type == model.ProxyGroupTypeAPI {
		c.JSON(http.StatusBadRequest, gin.H{"error": "API 代理组不支持手动添加代理节点，请使用拉取"})
		return
	}
	var input model.ProxyNodeInput
	if !bindJSON(c, &input) || !validateProxyNodeInput(c, input) {
		return
	}
	node, err := h.store.CreateProxyNode(c.Request.Context(), groupID, input)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, node)
}

func (h *Handler) updateProxyNode(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	node, err := h.store.GetProxyNode(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	if !h.requireProxyGroupEditable(c, node.GroupID) {
		return
	}
	var input model.ProxyNodeInput
	if !bindJSON(c, &input) || !validateProxyNodeInput(c, input) {
		return
	}
	updated, err := h.store.UpdateProxyNode(c.Request.Context(), id, input)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) deleteProxyNode(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	node, err := h.store.GetProxyNode(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	if !h.requireProxyGroupEditable(c, node.GroupID) {
		return
	}
	if err := h.store.DeleteProxyNode(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) testProxyGroup(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireProxyGroupEditable(c, id) {
		return
	}
	group, err := h.testProxyGroupNodes(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, group)
}

func (h *Handler) pullProxyGroup(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireProxyGroupEditable(c, id) {
		return
	}
	group, err := h.store.GetProxyGroup(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	if group.Type != model.ProxyGroupTypeAPI || group.APIProvider != model.ProxyProviderKuaidailiDPS {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅快代理 API 代理组支持拉取"})
		return
	}
	pullCtx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	nodes, err := pullKuaidailiDPS(pullCtx, group)
	cancel()
	if err != nil {
		_, _ = h.store.SetProxyGroupPullResult(c.Request.Context(), id, "error", err.Error())
		respondError(c, err)
		return
	}
	if _, err := h.store.ReplaceAPIProxyNodes(c.Request.Context(), id, nodes); err != nil {
		respondError(c, err)
		return
	}
	if _, err := h.store.SetProxyGroupPullResult(c.Request.Context(), id, "success", fmt.Sprintf("已拉取 %d 个代理节点。", len(nodes))); err != nil {
		respondError(c, err)
		return
	}
	group, err = h.store.ClearProxyGroupTestResult(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, group)
}

func (h *Handler) testProxyGroupNodes(ctx context.Context, groupID int64) (model.ProxyGroup, error) {
	nodes, err := h.store.ListProxyNodes(ctx, groupID)
	if err != nil {
		return model.ProxyGroup{}, err
	}
	successCount := 0
	totalLatencyMillis := int64(0)
	for _, node := range nodes {
		testCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		result, err := testProxyNode(testCtx, node)
		cancel()
		status := "success"
		message := "代理检测通过。"
		if err != nil {
			status = "error"
			message = err.Error()
		} else {
			successCount++
			totalLatencyMillis += result.LatencyMillis
			if result.IPLocationErr != "" {
				message = fmt.Sprintf("代理检测通过，IP 归属地获取失败：%s", result.IPLocationErr)
			}
		}
		if _, updateErr := h.store.SetProxyNodeTestResult(ctx, node.ID, status, message, result.LatencyMillis, result.IPLocation); updateErr != nil {
			return model.ProxyGroup{}, updateErr
		}
	}
	status := "success"
	if successCount == 0 {
		status = "error"
	}
	message := fmt.Sprintf("检测完成：%d/%d 个节点可用。", successCount, len(nodes))
	if successCount > 0 {
		message = fmt.Sprintf("%s 平均延时 %dms。", message, totalLatencyMillis/int64(successCount))
	}
	return h.store.SetProxyGroupTestResult(ctx, groupID, status, message)
}

func (h *Handler) requireProxyGroupEditable(c *gin.Context, id int64) bool {
	if !h.requireResourceOwner(c, h.store.GetProxyGroupOwnerUID, id) {
		return false
	}
	inUse, err := h.store.ProxyGroupInUse(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return false
	}
	if inUse {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该代理组正在被运行中的任务使用，无法编辑"})
		return false
	}
	return true
}

func (h *Handler) listTasks(c *gin.Context) {
	tasks, err := h.store.ListTasks(c.Request.Context(), ownerUIDFromContext(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, tasks)
}

func (h *Handler) createTask(c *gin.Context) {
	var input model.TaskInput
	if !bindJSON(c, &input) || !requireName(c, input.Name, "任务名称不能为空") || !validateTaskInput(c, input) {
		return
	}
	if !h.requireTaskResourcesOwned(c, input.AccountID, input.ProxyGroupID) {
		return
	}
	ownerUID := ownerUIDFromContext(c)
	task, err := h.store.CreateTask(c.Request.Context(), ownerUID, input)
	if err != nil {
		respondError(c, err)
		return
	}
	h.hub.PublishForOwner("task.updated", task, ownerUID)
	c.JSON(http.StatusCreated, task)
}

func (h *Handler) requireTaskResourcesOwned(c *gin.Context, accountID int64, proxyGroupID int64) bool {
	if accountID > 0 && !h.requireResourceOwner(c, h.store.GetAccountOwnerUID, accountID) {
		return false
	}
	if proxyGroupID > 0 && !h.requireResourceOwner(c, h.store.GetProxyGroupOwnerUID, proxyGroupID) {
		return false
	}
	return true
}

func (h *Handler) updateTask(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireResourceOwner(c, h.store.GetTaskOwnerUID, id) {
		return
	}
	var input model.TaskInput
	if !bindJSON(c, &input) || !requireName(c, input.Name, "任务名称不能为空") || !validateTaskInput(c, input) {
		return
	}
	if !h.requireTaskResourcesOwned(c, input.AccountID, input.ProxyGroupID) {
		return
	}
	ownerUID := ownerUIDFromContext(c)
	task, err := h.store.UpdateTask(c.Request.Context(), id, input)
	if err != nil {
		respondError(c, err)
		return
	}
	h.hub.PublishForOwner("task.updated", task, ownerUID)
	c.JSON(http.StatusOK, task)
}

func validateTaskInput(c *gin.Context, input model.TaskInput) bool {
	taskMode := model.NormalizeTaskMode(input.TaskMode)
	hasRushStage := taskMode == model.TaskModeRush || taskMode == model.TaskModeHybrid
	if hasRushStage && model.NormalizeProxyMode(input.ProxyMode) == model.ProxyModeConcurrent && input.ProxyGroupID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "并发代理需要选择代理组"})
		return false
	}
	return true
}

func (h *Handler) deleteTask(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	task, err := h.store.GetTask(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	if task.OwnerUID != ownerUIDFromContext(c) {
		c.JSON(http.StatusNotFound, gin.H{"error": "资源不存在"})
		return
	}
	h.runner.Cancel(id)
	logMessage := "任务已删除。"
	if name := strings.TrimSpace(task.Name); name != "" {
		logMessage = fmt.Sprintf("任务已删除：%s。", name)
	}
	if log, err := h.store.AddTaskLog(c.Request.Context(), id, "warn", logMessage); err == nil && log.ID > 0 {
		h.hub.PublishForOwner("log.created", log, task.OwnerUID)
	}
	if err := h.store.DeleteTask(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	h.hub.PublishForOwner("task.deleted", gin.H{"id": id}, task.OwnerUID)
	c.Status(http.StatusNoContent)
}

func (h *Handler) dispatchTask(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireResourceOwner(c, h.store.GetTaskOwnerUID, id) {
		return
	}
	task, err := h.runner.Dispatch(c.Request.Context(), id)
	if err != nil {
		respondTaskActionError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *Handler) pauseTask(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireResourceOwner(c, h.store.GetTaskOwnerUID, id) {
		return
	}
	task, err := h.runner.Pause(c.Request.Context(), id)
	if err != nil {
		respondTaskActionError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *Handler) listTaskLogs(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	logs, err := h.store.ListTaskLogs(c.Request.Context(), ownerUIDFromContext(c), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, logs)
}

func (h *Handler) fetchBWSInfo(c *gin.Context) {
	var input model.BWSInfoInput
	if !bindJSON(c, &input) {
		return
	}
	if input.AccountID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择账号"})
		return
	}
	if !h.requireResourceOwner(c, h.store.GetAccountOwnerUID, input.AccountID) {
		return
	}
	_, cookie, err := h.store.GetAccountCookie(c.Request.Context(), input.AccountID)
	if err != nil {
		respondError(c, err)
		return
	}
	info, err := h.bws.FetchInfo(c.Request.Context(), cookie, input)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, info)
}

func (h *Handler) fetchBWSMyReservations(c *gin.Context) {
	accountID, err := strconv.ParseInt(strings.TrimSpace(c.Query("accountId")), 10, 64)
	if err != nil || accountID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accountId 必须是有效账号 ID"})
		return
	}
	year := strings.TrimSpace(c.Query("year"))
	if !h.requireResourceOwner(c, h.store.GetAccountOwnerUID, accountID) {
		return
	}
	_, cookie, err := h.store.GetAccountCookie(c.Request.Context(), accountID)
	if err != nil {
		respondError(c, err)
		return
	}
	result, err := h.bws.FetchMyReservations(c.Request.Context(), cookie, year)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) listBWSTasks(c *gin.Context) {
	tasks, err := h.store.ListBWSTasks(c.Request.Context(), ownerUIDFromContext(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, tasks)
}

func (h *Handler) createBWSTask(c *gin.Context) {
	var input model.BWSTaskInput
	if !bindJSON(c, &input) || !requireName(c, input.Name, "BWS 任务名称不能为空") || !validateBWSTaskInput(c, input) {
		return
	}
	if !h.requireTaskResourcesOwned(c, input.AccountID, input.ProxyGroupID) {
		return
	}
	ownerUID := ownerUIDFromContext(c)
	task, err := h.store.CreateBWSTask(c.Request.Context(), ownerUID, input)
	if err != nil {
		respondError(c, err)
		return
	}
	h.hub.PublishForOwner("bws.task.updated", task, ownerUID)
	c.JSON(http.StatusCreated, task)
}

func (h *Handler) updateBWSTask(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireResourceOwner(c, h.store.GetBWSTaskOwnerUID, id) {
		return
	}
	var input model.BWSTaskInput
	if !bindJSON(c, &input) || !requireName(c, input.Name, "BWS 任务名称不能为空") || !validateBWSTaskInput(c, input) {
		return
	}
	if !h.requireTaskResourcesOwned(c, input.AccountID, input.ProxyGroupID) {
		return
	}
	ownerUID := ownerUIDFromContext(c)
	task, err := h.store.UpdateBWSTask(c.Request.Context(), id, input)
	if err != nil {
		respondError(c, err)
		return
	}
	h.hub.PublishForOwner("bws.task.updated", task, ownerUID)
	c.JSON(http.StatusOK, task)
}

func validateBWSTaskInput(c *gin.Context, input model.BWSTaskInput) bool {
	if input.AccountID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BWS 预约任务需要选择账号"})
		return false
	}
	if strings.TrimSpace(input.Year) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BWS 预约任务需要年份"})
		return false
	}
	if strings.TrimSpace(input.TicketNo) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BWS 预约任务需要票号"})
		return false
	}
	if input.ReserveID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BWS 预约任务需要预约 ID"})
		return false
	}
	if strings.TrimSpace(input.Title) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BWS 预约任务需要标题"})
		return false
	}
	if model.NormalizeBWSTaskMode(input.Mode) == model.BWSTaskModeScheduled && input.ReserveBeginTime <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BWS 预约任务需要预约开始时间"})
		return false
	}
	if input.PollIntervalMillis <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BWS 循环间隔必须大于 0"})
		return false
	}
	if model.NormalizeProxyMode(input.ProxyMode) == model.ProxyModeConcurrent && input.ProxyGroupID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BWS 并发代理需要选择代理组"})
		return false
	}
	return true
}

func (h *Handler) deleteBWSTask(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	task, err := h.store.GetBWSTask(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	if task.OwnerUID != ownerUIDFromContext(c) {
		c.JSON(http.StatusNotFound, gin.H{"error": "资源不存在"})
		return
	}
	h.bwsRunner.Cancel(id)
	logMessage := "BWS 预约任务已删除。"
	if name := strings.TrimSpace(task.Name); name != "" {
		logMessage = fmt.Sprintf("BWS 预约任务已删除：%s。", name)
	}
	if log, err := h.store.AddBWSTaskLog(c.Request.Context(), id, "warn", logMessage); err == nil && log.ID > 0 {
		h.hub.PublishForOwner("bws.log.created", log, task.OwnerUID)
	}
	if err := h.store.DeleteBWSTask(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	h.hub.PublishForOwner("bws.task.deleted", gin.H{"id": id}, task.OwnerUID)
	c.Status(http.StatusNoContent)
}

func (h *Handler) dispatchBWSTask(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireResourceOwner(c, h.store.GetBWSTaskOwnerUID, id) {
		return
	}
	task, err := h.bwsRunner.Dispatch(c.Request.Context(), id)
	if err != nil {
		respondTaskActionError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *Handler) pauseBWSTask(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.requireResourceOwner(c, h.store.GetBWSTaskOwnerUID, id) {
		return
	}
	task, err := h.bwsRunner.Pause(c.Request.Context(), id)
	if err != nil {
		respondTaskActionError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *Handler) listBWSTaskLogs(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	logs, err := h.store.ListBWSTaskLogs(c.Request.Context(), ownerUIDFromContext(c), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, logs)
}

func (h *Handler) listBWSLogs(c *gin.Context) {
	var taskID int64
	if raw := c.Query("task_id"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "task_id 必须是数字"})
			return
		}
		taskID = parsed
	}
	logs, err := h.store.ListBWSTaskLogs(c.Request.Context(), ownerUIDFromContext(c), taskID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, logs)
}

func (h *Handler) listLogs(c *gin.Context) {
	var taskID int64
	if raw := c.Query("task_id"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "task_id 必须是数字"})
			return
		}
		taskID = parsed
	}
	logs, err := h.store.ListTaskLogs(c.Request.Context(), ownerUIDFromContext(c), taskID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, logs)
}

func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体不是有效 JSON"})
		return false
	}
	return true
}

func decodeAccountImportInput(c *gin.Context) ([]model.AccountExportItem, bool) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxAccountImportJSONBytes+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取导入 JSON 失败"})
		return nil, false
	}
	if len(body) > maxAccountImportJSONBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "导入 JSON 不能超过 2MB"})
		return nil, false
	}
	if strings.TrimSpace(string(body)) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "导入 JSON 不能为空"})
		return nil, false
	}

	var accounts []model.AccountExportItem
	if err := json.Unmarshal(body, &accounts); err == nil && accounts != nil {
		return accounts, true
	}
	var input model.AccountImportInput
	if err := json.Unmarshal(body, &input); err == nil && input.Accounts != nil {
		return input.Accounts, true
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "账号导入 JSON 需要包含 accounts 数组"})
	return nil, false
}

func validateAccountImportInput(c *gin.Context, accounts []model.AccountExportItem) bool {
	if len(accounts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "导入账号不能为空"})
		return false
	}
	if len(accounts) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "一次最多导入 200 个账号"})
		return false
	}
	for index, account := range accounts {
		if strings.TrimSpace(account.Name) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("第 %d 个账号名称不能为空", index+1)})
			return false
		}
	}
	return true
}

func validateAccountBatchExportInput(c *gin.Context, input model.AccountBatchExportInput) bool {
	if len(input.AccountIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要导出的账号"})
		return false
	}
	if len(input.AccountIDs) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "一次最多导出 200 个账号"})
		return false
	}
	seen := make(map[int64]struct{}, len(input.AccountIDs))
	for _, id := range input.AccountIDs {
		if id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "账号 ID 必须是正整数"})
			return false
		}
		if _, ok := seen[id]; ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "导出账号不能重复选择"})
			return false
		}
		seen[id] = struct{}{}
	}
	return true
}

func validateNotificationInput(c *gin.Context, input model.NotificationInput) bool {
	if !requireName(c, input.Name, "通知名称不能为空") {
		return false
	}
	provider := model.NormalizeNotificationProvider(input.Provider)
	if provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "通知类型必须是 pushplus 或 bark"})
		return false
	}
	if strings.TrimSpace(input.Config["token"]) == "" {
		if provider == model.NotificationProviderPushPlus {
			c.JSON(http.StatusBadRequest, gin.H{"error": "PushPlus Token 不能为空"})
			return false
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bark Token 或完整推送地址不能为空"})
		return false
	}
	return true
}

func validateProxyGroupInput(c *gin.Context, input model.ProxyGroupInput) bool {
	if !requireName(c, input.Name, "代理组名称不能为空") {
		return false
	}
	groupType := model.NormalizeProxyGroupType(input.Type)
	if groupType == model.ProxyGroupTypeAPI {
		if model.NormalizeProxyProvider(input.APIProvider) != model.ProxyProviderKuaidailiDPS {
			c.JSON(http.StatusBadRequest, gin.H{"error": "API 代理组目前仅支持快代理私密代理"})
			return false
		}
		if strings.TrimSpace(input.APIConfig["secretId"]) == "" || strings.TrimSpace(input.APIConfig["secretKey"]) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "快代理 SecretId 和 SecretKey 不能为空"})
			return false
		}
		if _, ok := model.ParseProxyAPIPullTimes(input.APIConfig["pullTimes"]); !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("提取次数必须是 %d-%d 的整数", model.DefaultProxyAPIPullTimes, model.MaxProxyAPIPullTimes)})
			return false
		}
	}
	return true
}

func validateProxyNodeInput(c *gin.Context, input model.ProxyNodeInput) bool {
	if strings.TrimSpace(input.Host) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "代理地址不能为空"})
		return false
	}
	if input.Port <= 0 || input.Port > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "代理端口必须是 1-65535"})
		return false
	}
	return true
}

func requireName(c *gin.Context, value string, message string) bool {
	if strings.TrimSpace(value) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return false
	}
	return true
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID 必须是正整数"})
		return 0, false
	}
	return id, true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func respondError(c *gin.Context, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "资源不存在"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func respondTaskActionError(c *gin.Context, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

func writeSSE(c *gin.Context, name string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", name, data)
	c.Writer.Flush()
}
