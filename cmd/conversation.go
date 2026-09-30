package main

import (
	"encoding/json"
	"fmt"
	"mime"
	"strconv"
	"strings"
	"time"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/conversation"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	vmodels "github.com/abhinavxd/libredesk/internal/view/models"
	wmodels "github.com/abhinavxd/libredesk/internal/webhook/models"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

type assigneeChangeReq struct {
	AssigneeID int `json:"assignee_id"`
}

type teamAssigneeChangeReq struct {
	AssigneeID int `json:"assignee_id"`
}

type priorityUpdateReq struct {
	Priority string `json:"priority"`
}

type statusUpdateReq struct {
	Status       string `json:"status"`
	SnoozedUntil string `json:"snoozed_until,omitempty"`
}

type createConversationRequest struct {
	InboxID     int    `json:"inbox_id"`
	Email       string `json:"email"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Subject     string `json:"subject"`
	Content     string `json:"content"`
	Attachments []int  `json:"attachments"`
	Initiator   string `json:"initiator"` // "contact" | "agent"
	SourceID    string `json:"source_id"` // RFC 5322 Message-ID of the inbound message; stored on the created contact message so replies thread on it. Contact-initiated only.
}

// handleGetAllConversations retrieves all conversations.
func handleGetAllConversations(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		user    = r.RequestCtx.UserValue("user").(amodels.User)
		order   = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		filters = string(r.RequestCtx.QueryArgs().Peek("filters"))
		total   = 0
	)
	page, pageSize := getPagination(r)

	conversations, err := app.conversation.GetAllConversationsList(user.ID, order, orderBy, filters, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if len(conversations) > 0 {
		total = conversations[0].Total
	}

	return r.SendEnvelope(envelope.PageResults{
		Results:    conversations,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetMentionedConversations retrieves conversations where the current user is mentioned.
func handleGetMentionedConversations(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		user    = r.RequestCtx.UserValue("user").(amodels.User)
		order   = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		filters = string(r.RequestCtx.QueryArgs().Peek("filters"))
		total   = 0
	)
	page, pageSize := getPagination(r)

	conversations, err := app.conversation.GetMentionedConversationsList(user.ID, order, orderBy, filters, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(conversations) > 0 {
		total = conversations[0].Total
	}

	return r.SendEnvelope(envelope.PageResults{
		Results:    conversations,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetViewConversations retrieves conversations for a view.
func handleGetViewConversations(r *fastglue.Request) error {
	var (
		app       = r.Context.(*App)
		auser     = r.RequestCtx.UserValue("user").(amodels.User)
		viewID, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		order     = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy   = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		total     = 0
	)
	page, pageSize := getPagination(r)
	if viewID < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	// Check if user has access to the view.
	view, err := app.view.Get(viewID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if !conversation.UserCanAccessView(view, auser.ID, user.Teams.IDs()) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("conversation.viewPermissionDenied"), nil, envelope.PermissionError)
	}

	lists := conversation.ListsForUserPermissions(user.Permissions)
	// No lists found, user doesn't have access to any conversations.
	if len(lists) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("status.deniedPermission"), nil, envelope.PermissionError)
	}

	conversations, err := app.conversation.GetViewConversationsList(user.ID, user.ID, user.Teams.IDs(), lists, order, orderBy, string(view.Filters), page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(conversations) > 0 {
		total = conversations[0].Total
	}

	return r.SendEnvelope(envelope.PageResults{
		Results:    conversations,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetSidebarCounts returns open-conversation counts for inbox sidebar badges.
func handleGetSidebarCounts(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)

	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	personalViews, err := app.view.GetUsersViews(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	sharedViews, err := app.view.GetSharedViewsForUser(user.Teams.IDs())
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	allViews := make([]vmodels.View, 0, len(personalViews)+len(sharedViews))
	allViews = append(allViews, personalViews...)
	allViews = append(allViews, sharedViews...)

	counts, err := app.conversation.GetSidebarCounts(user.ID, user.Permissions, user.Teams.IDs(), allViews)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(counts)
}

// handleGetViewCount returns the sidebar badge count for one view.
func handleGetViewCount(r *fastglue.Request) error {
	var (
		app       = r.Context.(*App)
		auser     = r.RequestCtx.UserValue("user").(amodels.User)
		viewID, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if viewID < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	view, err := app.view.Get(viewID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if !conversation.UserCanAccessView(view, auser.ID, user.Teams.IDs()) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("conversation.viewPermissionDenied"), nil, envelope.PermissionError)
	}

	count, err := app.conversation.GetViewCount(user.ID, user.Permissions, user.Teams.IDs(), view)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(map[string]int{"count": count})
}

// handleGetConversation retrieves a single conversation by it's UUID.
func handleGetConversation(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)

	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	conv, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(conv)
}

// handleDownloadConversationTranscript sends the conversation transcript as a text file download.
func handleDownloadConversationTranscript(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)

	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	conversation, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	private := false
	messages, err := app.conversation.GetAllConversationMessages(uuid, &private, []string{cmodels.MessageIncoming, cmodels.MessageOutgoing}, 0)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	transcript := app.conversation.BuildTranscript(*conversation, messages, time.Now())
	safeRef := stringutil.SanitizeFilename(conversation.ReferenceNumber)
	filename := fmt.Sprintf("transcript-%s.txt", safeRef)
	r.RequestCtx.Response.Header.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	r.RequestCtx.Response.Header.Set("X-Content-Type-Options", "nosniff")
	r.RequestCtx.SetContentType("text/plain; charset=utf-8")
	r.RequestCtx.SetBody(transcript)
	return nil
}

// handleUpdateConversationAssigneeLastSeen updates the current user's last seen timestamp for a conversation.
func handleUpdateConversationAssigneeLastSeen(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err = app.conversation.UpdateUserLastSeen(uuid, auser.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleMarkConversationAsUnread marks a conversation as unread for the current user.
func handleMarkConversationAsUnread(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err = app.conversation.MarkAsUnread(uuid, auser.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleGetConversationParticipants retrieves participants of a conversation.
func handleGetConversationParticipants(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	p, err := app.conversation.GetConversationParticipants(uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(p)
}

// handleUpdateConversationStatus updates the status of a conversation.
func handleUpdateConversationStatus(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		req   = statusUpdateReq{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error decoding status update request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}

	status := req.Status
	snoozedUntil := req.SnoozedUntil

	// Validate inputs
	if status == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`status`"), nil, envelope.InputError)
	}
	if snoozedUntil == "" && status == cmodels.StatusSnoozed {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`snoozed_until`"), nil, envelope.InputError)
	}
	if status == cmodels.StatusSnoozed {
		_, err := time.ParseDuration(snoozedUntil)
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
		}
	}

	// Enforce conversation access.
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Update conversation status.
	if err := app.conversation.UpdateConversationStatus(uuid, 0 /**status_id**/, status, snoozedUntil, user); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// enforceConversationAccess fetches the conversation and checks if the user has access to it.
func enforceConversationAccess(app *App, uuid string, user umodels.User) (*cmodels.Conversation, error) {
	conversation, err := app.conversation.GetConversation(0, uuid, "")
	if err != nil {
		return nil, err
	}
	allowed, err := app.authz.EnforceConversationAccess(user, conversation)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, envelope.NewError(envelope.PermissionError, "Permission denied", nil)
	}
	return &conversation, nil
}

// handleCreateConversation creates a new conversation and sends a message to it.
func handleCreateConversation(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		req   = createConversationRequest{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error decoding create conversation request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}

	if err := requireInboxAccess(app, auser.ID, req.InboxID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if err := validateCreateConversationRequest(req, app); err != nil {
		return sendErrorEnvelope(r, err)
	}

	email := req.Email
	to := []string{email}

	contact := umodels.User{
		Email:            null.StringFrom(email),
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		CustomAttributes: json.RawMessage(`{}`),
	}
	if err := app.user.ResolveEmailSender(&contact); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Create conversation first.
	conversationID, conversationUUID, err := app.conversation.CreateConversation(
		contact.ID,
		req.InboxID,
		"",         /** last_message **/
		time.Now(), /** last_message_at **/
		req.Subject,
		true, /** append reference number to subject? **/
		nil,
		nil,
		0, 0,
	)
	if err != nil {
		app.lo.Error("error creating conversation", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}

	// Get media for the attachment ids, skip any already associated with a model.
	media, err := getUnassociatedMedia(app, req.Attachments)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	// Send initial message based on the initiator of conversation.
	switch req.Initiator {
	case umodels.UserTypeAgent:
		// Queue reply.
		if _, err := app.conversation.QueueReply(media, req.InboxID, auser.ID /**sender_id**/, contact.ID, conversationUUID, req.Content, to, nil /**cc**/, nil /**bcc**/, map[string]any{} /**meta**/); err != nil {
			// Delete the conversation if msg queue fails.
			if err := app.conversation.DeleteConversation(conversationUUID); err != nil {
				app.lo.Error("error deleting conversation", "error", err)
			}
			return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.errorSendingMessage"), nil))
		}
		// Trigger webhook for agent-initiated conversation, for contact intitiated the incoming message hooks handle it.
		if c, err := app.conversation.GetConversation(0, conversationUUID, ""); err == nil {
			app.webhook.TriggerEvent(wmodels.EventConversationCreated, c)
		}
	case umodels.UserTypeContact:
		// Create contact message.
		if _, err := app.conversation.CreateContactMessage(media, contact.ID, conversationUUID, req.Content, cmodels.ContentTypeHTML, true, req.SourceID); err != nil {
			// Delete the conversation if message creation fails.
			if err := app.conversation.DeleteConversation(conversationUUID); err != nil {
				app.lo.Error("error deleting conversation", "error", err)
			}
			return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.errorSendingMessage"), nil))
		}
	default:
		// Guard anyway.
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	conversation, _ := app.conversation.GetConversation(conversationID, "", "")
	return r.SendEnvelope(conversation)
}

func validateCreateConversationRequest(req createConversationRequest, app *App) error {
	if req.InboxID <= 0 {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.required", "name", "`inbox_id`"), nil)
	}
	if req.Content == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.required", "name", "`content`"), nil)
	}
	if req.Email == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.required", "name", "`email`"), nil)
	}

	if !stringutil.ValidEmail(req.Email) {
		return envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidEmail"), nil)
	}
	if req.Initiator != umodels.UserTypeContact && req.Initiator != umodels.UserTypeAgent {
		return envelope.NewError(envelope.InputError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	// Check if inbox exists and is enabled.
	inbox, err := app.inbox.GetDBRecord(req.InboxID)
	if err != nil {
		return err
	}
	if !inbox.Enabled {
		return envelope.NewError(envelope.InputError, app.i18n.T("globals.messages.disabled"), nil)
	}
	if inbox.Channel != "email" {
		return envelope.NewError(envelope.InputError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	return nil
}
