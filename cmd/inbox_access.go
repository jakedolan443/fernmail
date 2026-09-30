package main

import (
	"encoding/json"
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox"
	wsmodels "github.com/abhinavxd/libredesk/internal/ws/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func requireInboxAccess(app *App, userID, inboxID int) error {
	allowed, err := app.inbox.CanAccess(userID, inboxID)
	if err != nil {
		return envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if !allowed {
		return envelope.NewError(envelope.PermissionError, app.i18n.T("status.deniedPermission"), nil)
	}
	return nil
}

func handleGetMailboxes(r *fastglue.Request) error {
	app := r.Context.(*App)
	user := r.RequestCtx.UserValue("user").(amodels.User)
	mailboxes, err := app.inbox.GetMailboxes(user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(mailboxes)
}

func handleGetInboxAccess(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	access, err := app.inbox.GetAccess(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(access)
}

func handleUpdateInboxAccess(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	var access inbox.InboxAccess
	if err := r.Decode(&access, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid inbox access settings", nil, envelope.InputError)
	}
	if _, err := app.inbox.GetDBRecord(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.inbox.UpdateAccess(id, access); err != nil {
		app.lo.Error("updating inbox access", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Could not save inbox access. Check the selected users and roles.", nil, envelope.InputError)
	}
	// No mailbox metadata in this event: even users losing access must refresh.
	app.wsHub.BroadcastMessage(wsmodels.BroadcastMessage{Data: []byte(`{"type":"mailbox_access_updated"}`)})
	return r.SendEnvelope(true)
}

func handleGetMailboxConversations(r *fastglue.Request) error {
	app := r.Context.(*App)
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	id, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err := requireInboxAccess(app, auser.ID, id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	lists := conversation.ListsForUserPermissions(user.Permissions)
	if len(lists) == 0 {
		return sendErrorEnvelope(r, envelope.NewError(envelope.PermissionError, app.i18n.T("status.deniedPermission"), nil))
	}
	// This list endpoint accepts flat AND filters, with the selected inbox always enforced.
	filters := []dbutil.FilterNode{}
	raw := r.RequestCtx.QueryArgs().Peek("filters")
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &filters); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid mailbox filters", nil, envelope.InputError)
		}
	}
	filters = append(filters, dbutil.FilterNode{Model: "conversations", Field: "inbox_id", Operator: "equals", Value: strconv.Itoa(id)})
	filterJSON, _ := json.Marshal(filters)
	page, size := getPagination(r)
	rows, err := app.conversation.GetConversations(user.ID, user.ID, user.Teams.IDs(), lists,
		string(r.RequestCtx.QueryArgs().Peek("order")), string(r.RequestCtx.QueryArgs().Peek("order_by")), string(filterJSON), page, size)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	total := 0
	if len(rows) > 0 {
		total = rows[0].Total
	}
	return r.SendEnvelope(envelope.PageResults{Results: rows, Total: total, PerPage: size, TotalPages: (total + size - 1) / size, Page: page})
}
