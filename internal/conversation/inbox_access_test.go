package conversation

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/authz"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/search"
	smodels "github.com/abhinavxd/libredesk/internal/search/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	vmodels "github.com/abhinavxd/libredesk/internal/view/models"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

// Exercise the real database predicates used by lists, views, search, drafts,
// unread counts, direct message access and WebSocket subscriptions.
func TestInboxAccessAcrossMailSurfaces(t *testing.T) {
	db := testutil.NewDB(t, "inbox_access_surfaces")
	lo := logf.New(logf.Opts{})
	i18n := testutil.NewI18n(t)
	inboxes, err := inbox.New(&lo, db, i18n, "")
	if err != nil {
		t.Fatal(err)
	}
	var viewer, contact, billing, support, role int
	mustID := func(dest *int, sql string) {
		t.Helper()
		if err := db.Get(dest, sql); err != nil {
			t.Fatal(err)
		}
	}
	mustID(&viewer, `INSERT INTO users(type,email,first_name) VALUES('agent','reader@example.test','Reader') RETURNING id`)
	mustID(&contact, `INSERT INTO users(type,email,first_name) VALUES('contact','sender@example.test','Sender') RETURNING id`)
	mustID(&billing, `INSERT INTO inboxes(name,channel) VALUES('Billing','email') RETURNING id`)
	mustID(&support, `INSERT INTO inboxes(name,channel) VALUES('Support','email') RETURNING id`)
	mustID(&role, `INSERT INTO roles(name) VALUES('Billing reader') RETURNING id`)
	db.MustExec(`INSERT INTO user_roles(user_id,role_id) VALUES($1,$2)`, viewer, role)
	db.MustExec(`INSERT INTO conversations(contact_id,inbox_id,status_id,subject,last_message_at)
   SELECT $1,id,(SELECT id FROM conversation_statuses WHERE name='Open'), name,now() FROM inboxes`, contact)
	db.MustExec(`INSERT INTO conversation_messages(conversation_id,sender_id,sender_type,type,status,text_content)
   SELECT id,$1,'contact','incoming','received','private mailbox example' FROM conversations`, contact)
	db.MustExec(`INSERT INTO conversation_drafts(conversation_id,user_id,type,content)
   SELECT id,$1,'reply','A draft' FROM conversations`, viewer)
	db.MustExec(`INSERT INTO conversation_mentions(conversation_id,message_id,mentioned_user_id,mentioned_by_user_id)
   SELECT conversation_id,id,$1,$1 FROM conversation_messages`, viewer)

	manager := newTestManager()
	manager.db, manager.lo, manager.i18n = db, &lo, i18n
	if err := dbutil.ScanSQLFile("queries.sql", &manager.q, db, efs); err != nil {
		t.Fatal(err)
	}
	searcher, err := search.New(search.Opts{DB: db, Lo: &lo, I18n: i18n, FilterLocation: func() string { return "UTC" }})
	if err != nil {
		t.Fatal(err)
	}
	enforcer, _ := authz.NewEnforcer(&lo, i18n, inboxes.CanAccess)
	reader := umodels.User{ID: viewer, Enabled: true, Permissions: []string{"conversations:read", "conversations:read_all"}}
	scope := smodels.ReadScope{UserID: viewer, Read: true, ReadAll: true}
	var uuids []string
	if err := db.Select(&uuids, `SELECT uuid::text FROM conversations`); err != nil {
		t.Fatal(err)
	}
	check := func(want int) {
		t.Helper()
		mailboxes, err := inboxes.GetMailboxes(viewer)
		if err != nil || len(mailboxes) != want {
			t.Fatalf("mailboxes=%v err=%v want=%d", mailboxes, err, want)
		}
		for _, lists := range [][]string{{models.AllConversations}, {models.MentionedConversations}, {models.AllConversations, models.AssignedConversations}} {
			rows, err := manager.GetConversations(viewer, viewer, nil, lists, "DESC", "", "[]", 1, 20)
			if err != nil || len(rows) != want {
				t.Fatalf("lists %v: rows=%d err=%v want=%d", lists, len(rows), err, want)
			}
		}
		count, err := manager.GetViewCount(viewer, reader.Permissions, nil, vmodels.View{ID: 1, Filters: []byte(`[]`)})
		if err != nil || count != want {
			t.Fatalf("view count=%d err=%v want=%d", count, err, want)
		}
		counts, err := manager.GetSidebarCounts(viewer, reader.Permissions, nil, nil)
		if err != nil || counts.All != want || counts.Mentioned != want || counts.Unread != want || len(counts.Inboxes) != want {
			t.Fatalf("counts=%+v err=%v want=%d", counts, err, want)
		}
		drafts, err := manager.GetAllUserDrafts(viewer)
		if err != nil || len(drafts) != want {
			t.Fatalf("drafts=%d err=%v want=%d", len(drafts), err, want)
		}
		var authorized []string
		err = manager.q.FilterAuthorizedListUUIDs.Select(&authorized, pq.Array(uuids), viewer, pq.Array([]int{}), true, true, false, false, false, false)
		if err != nil || len(authorized) != want {
			t.Fatalf("subscriptions=%v err=%v want=%d", authorized, err, want)
		}
		conversations, _, _, err := searcher.Conversations(smodels.Query{Term: "sender@example.test", PageSize: 20}, scope)
		if err != nil || len(conversations) != want {
			t.Fatalf("search conversations=%d err=%v want=%d", len(conversations), err, want)
		}
		messages, _, _, err := searcher.Messages(smodels.Query{Term: "mailbox example", PageSize: 20}, scope)
		if err != nil || len(messages) != want {
			t.Fatalf("search messages=%d err=%v want=%d", len(messages), err, want)
		}
	}
	check(2) // Existing inboxes stay available until restricted.
	save := func(id int, access inbox.InboxAccess) {
		t.Helper()
		if err := inboxes.UpdateAccess(id, access); err != nil {
			t.Fatal(err)
		}
	}
	save(billing, inbox.InboxAccess{Restricted: true, RoleIDs: []int{role}})
	save(support, inbox.InboxAccess{Restricted: true})
	check(1)
	allowed, err := enforcer.EnforceConversationAccess(reader, models.Conversation{InboxID: support})
	if err != nil || allowed {
		t.Fatalf("restricted direct access = %t, %v", allowed, err)
	}
	save(support, inbox.InboxAccess{Restricted: true, UserIDs: []int{viewer}})
	check(2) // A direct grant or a role grant is sufficient.
	db.MustExec(`DELETE FROM user_roles WHERE user_id=$1 AND role_id=$2`, viewer, role)
	check(1) // Role removal takes effect immediately, independently of cached permissions.
	save(support, inbox.InboxAccess{Restricted: true})
	check(0)
	// A bad save is atomic; it must not reopen an inbox or lose existing grants.
	if err := inboxes.UpdateAccess(support, inbox.InboxAccess{Restricted: false, UserIDs: []int{contact}}); err == nil {
		t.Fatal("contact accepted as an agent")
	}
	check(0)
	db.MustExec(`INSERT INTO user_roles(user_id,role_id) SELECT $1,id FROM roles WHERE name='Admin'`, viewer)
	check(2) // Administrators retain access.
	db.MustExec(`UPDATE users SET enabled=false WHERE id=$1`, viewer)
	check(0) // Even an administrator must be active.
}
