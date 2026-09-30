package conversation

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	authzModels "github.com/jakedolan443/fernmail/internal/authz/models"
	"github.com/jakedolan443/fernmail/internal/conversation/models"
	"github.com/jakedolan443/fernmail/internal/dbutil"
	"github.com/jakedolan443/fernmail/internal/envelope"
	vmodels "github.com/jakedolan443/fernmail/internal/view/models"
)

const sidebarCountsViewBatchSize = 50
const sidebarCountsQueryTimeout = 10 * time.Second

// sidebarCountsScanCap bounds each view count's scan; the UI shows anything above 99 as "99+".
const sidebarCountsScanCap = 100

// GetSidebarCounts returns open counts for the standard inboxes and a count per accessible view.
func (c *Manager) GetSidebarCounts(viewingUserID int, permissions []string, teamIDs []int, views []vmodels.View) (models.SidebarCounts, error) {
	out := models.SidebarCounts{Views: map[int]int{}, Inboxes: map[int]int{}}
	ctx, cancel := context.WithTimeout(context.Background(), sidebarCountsQueryTimeout)
	defer cancel()

	if err := c.fillStandardSidebarCounts(ctx, &out, viewingUserID, permissions); err != nil {
		return out, err
	}

	lists := ListsForUserPermissions(permissions)
	if len(lists) == 0 {
		return out, nil
	}

	unread, err := c.getUnreadMessageCount(ctx, viewingUserID, teamIDs, lists)
	if err != nil {
		return out, err
	}
	out.Unread = unread
	args := []any{}
	conditions, err := appendListTypeConditions(lists, viewingUserID, viewingUserID, teamIDs, &args)
	if err != nil {
		return out, err
	}
	var mailboxCounts []struct {
		InboxID int `db:"inbox_id"`
		Count   int `db:"count"`
	}
	if err := c.db.SelectContext(ctx, &mailboxCounts, `SELECT conversations.inbox_id, COUNT(*) AS count
        FROM conversations WHERE status_id IN (SELECT id FROM conversation_statuses WHERE category='open') `+listTypeWhereClause(conditions)+` GROUP BY conversations.inbox_id`, args...); err != nil {
		return out, err
	}
	for _, row := range mailboxCounts {
		out.Inboxes[row.InboxID] = row.Count
	}

	accessible := make([]vmodels.View, 0, len(views))
	for _, view := range views {
		if UserCanAccessView(view, viewingUserID, teamIDs) {
			accessible = append(accessible, view)
		}
	}
	for start := 0; start < len(accessible); start += sidebarCountsViewBatchSize {
		batch := accessible[start:min(start+sidebarCountsViewBatchSize, len(accessible))]
		viewCounts, err := c.getViewCounts(ctx, viewingUserID, teamIDs, lists, batch)
		if err != nil {
			return out, err
		}
		maps.Copy(out.Views, viewCounts)
	}
	return out, nil
}

// getUnreadMessageCount counts across all accessible mail, regardless of the
// current list page, status or view. Use the same read cutoff as conversation rows.
func (c *Manager) getUnreadMessageCount(ctx context.Context, userID int, teamIDs []int, lists []string) (int, error) {
	if len(lists) == 0 {
		return 0, nil
	}
	args := []any{userID}
	conditions, err := appendListTypeConditions(lists, userID, userID, teamIDs, &args)
	if err != nil {
		return 0, err
	}
	query := `SELECT COUNT(*) FROM conversations
		JOIN inboxes ON inboxes.id = conversations.inbox_id
		JOIN conversation_messages AS messages ON messages.conversation_id = conversations.id
		LEFT JOIN conversation_last_seen AS seen ON seen.conversation_id = conversations.id AND seen.user_id = $1
		WHERE inboxes.channel = 'email'
		AND messages.created_at > COALESCE(seen.last_seen_at, '1970-01-01'::TIMESTAMPTZ)
		AND (messages.meta IS NULL OR NOT COALESCE((messages.meta->>'continuity_email')::boolean, false))
		` + listTypeWhereClause(conditions)
	var count int
	if err := c.db.GetContext(ctx, &count, query, args...); err != nil {
		return 0, fmt.Errorf("counting unread mail: %w", err)
	}
	return count, nil
}

// GetViewCount returns the capped open count for one view.
func (c *Manager) GetViewCount(viewingUserID int, permissions []string, teamIDs []int, view vmodels.View) (int, error) {
	lists := ListsForUserPermissions(permissions)
	if len(lists) == 0 {
		return 0, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), sidebarCountsQueryTimeout)
	defer cancel()

	counts, err := c.getViewCounts(ctx, viewingUserID, teamIDs, lists, []vmodels.View{view})
	if err != nil {
		return 0, err
	}
	return counts[view.ID], nil
}

func (c *Manager) fillStandardSidebarCounts(ctx context.Context, out *models.SidebarCounts, userID int, permissions []string) error {
	var row struct {
		Assigned   int `db:"assigned"`
		Unassigned int `db:"unassigned"`
		Mentioned  int `db:"mentioned"`
		All        int `db:"all"`
	}

	if err := c.q.GetSidebarStandardCounts.GetContext(ctx, &row, userID); err != nil {
		c.lo.Error("error fetching sidebar standard counts", "error", err)
		return envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	if slices.Contains(permissions, authzModels.PermConversationsReadAssigned) {
		out.Assigned = row.Assigned
	}
	if slices.Contains(permissions, authzModels.PermConversationsReadUnassigned) {
		out.Unassigned = row.Unassigned
	}
	if slices.Contains(permissions, authzModels.PermConversationsRead) {
		out.Mentioned = row.Mentioned
	}
	if slices.Contains(permissions, authzModels.PermConversationsReadAll) {
		out.All = row.All
	}
	return nil
}

// getViewCounts returns the conversation count per view ID in one round trip.
func (c *Manager) getViewCounts(ctx context.Context, userID int, teamIDs []int, listTypes []string, views []vmodels.View) (map[int]int, error) {
	counts := map[int]int{}
	if len(views) == 0 {
		return counts, nil
	}

	query, qArgs, err := c.makeViewCountsQuery(userID, teamIDs, listTypes, views)
	if err != nil {
		c.lo.Error("error making view counts query", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	var rows []struct {
		ViewID int `db:"view_id"`
		Count  int `db:"count"`
	}
	if err := c.db.SelectContext(ctx, &rows, query, qArgs...); err != nil {
		c.lo.Error("error fetching view counts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	for _, row := range rows {
		counts[row.ViewID] = row.Count
	}
	return counts, nil
}

// makeViewCountsQuery unions one counting subquery per view into a single (view_id, count) statement.
func (c *Manager) makeViewCountsQuery(userID int, teamIDs []int, listTypes []string, views []vmodels.View) (string, []any, error) {
	var (
		args  = []any{}
		parts = make([]string, 0, len(views))
		loc   = c.FilterLocation()
	)

	for _, view := range views {
		countQuery, nextArgs, err := c.makeConversationsCountQuery(args, userID, teamIDs, listTypes, string(view.Filters), loc)
		if err != nil {
			return "", nil, err
		}
		parts = append(parts, fmt.Sprintf("SELECT %d AS view_id, (SELECT COUNT(*) FROM (%s LIMIT %d) capped) AS count", view.ID, countQuery, sidebarCountsScanCap))
		args = nextArgs
	}

	return strings.Join(parts, " UNION ALL "), args, nil
}

// makeConversationsCountQuery builds a query selecting matching conversations, with placeholders continuing after existingArgs.
func (c *Manager) makeConversationsCountQuery(existingArgs []any, userID int, teamIDs []int, listTypes []string, filtersJSON, loc string) (string, []any, error) {
	if len(listTypes) == 0 {
		return "", nil, fmt.Errorf("no conversation list types specified")
	}

	qArgs := existingArgs
	conditions, err := appendListTypeConditions(listTypes, userID, userID, teamIDs, &qArgs)
	if err != nil {
		return "", nil, err
	}

	baseQuery := fmt.Sprintf(c.q.GetConversationsCountBase, listTypeWhereClause(conditions))

	return dbutil.BuildFilterQuery(baseQuery, qArgs, filtersJSON, ListFilterAllowedFields, ListFilterRenderers, loc)
}

// ListsForUserPermissions returns conversation list types the user may access.
func ListsForUserPermissions(permissions []string) []string {
	lists := []string{}
	hasTeamAll := slices.Contains(permissions, authzModels.PermConversationsReadTeamAll)

	for _, perm := range permissions {
		if perm == authzModels.PermConversationsReadAll {
			return []string{models.AllConversations}
		}
		if perm == authzModels.PermConversationsReadUnassigned {
			lists = append(lists, models.UnassignedConversations)
		}
		if perm == authzModels.PermConversationsReadAssigned {
			lists = append(lists, models.AssignedConversations)
		}
		if perm == authzModels.PermConversationsReadTeamInbox && !hasTeamAll {
			lists = append(lists, models.TeamUnassignedConversations)
		}
		if perm == authzModels.PermConversationsReadTeamAll {
			lists = append(lists, models.TeamAllConversations)
		}
	}
	return lists
}

// UserCanAccessView reports whether a user can access a view.
func UserCanAccessView(view vmodels.View, userID int, teamIDs []int) bool {
	switch view.Visibility {
	case vmodels.VisibilityUser:
		return view.UserID != nil && *view.UserID == userID
	case vmodels.VisibilityAll:
		return true
	case vmodels.VisibilityTeam:
		if view.TeamID == nil {
			return false
		}
		return slices.Contains(teamIDs, *view.TeamID)
	default:
		return false
	}
}
