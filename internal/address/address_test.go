package address

import (
	"strings"
	"testing"

	"github.com/jakedolan443/fernmail/internal/testutil"
)

func TestAddressLifecyclePreservesHistoricalRouting(t *testing.T) {
	db := testutil.NewDB(t, "address_lifecycle")
	manager, err := New(db)
	if err != nil {
		t.Fatal(err)
	}

	var inboxID, contactID, agentID int
	if err := db.Get(&inboxID, `INSERT INTO inboxes(name, channel, "from") VALUES ('Mail transport', 'email', 'director@example.test') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&contactID, `INSERT INTO users(type, email, first_name) VALUES ('contact', 'customer@example.test', 'Customer') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&agentID, `INSERT INTO users(type, email, first_name) VALUES ('agent', 'agent@example.test', 'Agent') RETURNING id`); err != nil {
		t.Fatal(err)
	}

	primary, err := manager.EnsureMailboxAddress(inboxID, "director@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Create(Address{InboxID: inboxID, Address: "duplicate@example.test", Kind: KindMailbox, Enabled: true, Restricted: true}); err == nil {
		t.Fatal("expected direct primary-address creation to be rejected")
	}

	alias, err := manager.Create(Address{InboxID: inboxID, Address: "support@example.test", Kind: KindAlias, Enabled: true, Restricted: true, UserIDs: []int{agentID}})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Get(new(int), `INSERT INTO conversations(contact_id, inbox_id, address_id, status_id)
		VALUES ($1, $2, $3, (SELECT id FROM conversation_statuses WHERE name='Open')) RETURNING id`, contactID, inboxID, alias.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := manager.Update(alias.ID, Address{InboxID: inboxID, Address: "help@example.test", Kind: KindAlias, Enabled: true, Restricted: true}); err == nil || !strings.Contains(err.Error(), "cannot be moved or renamed") {
		t.Fatalf("renaming historical alias error = %v", err)
	}
	if _, err := manager.Delete(alias.ID); err == nil || !strings.Contains(err.Error(), "must be disabled") {
		t.Fatalf("deleting historical alias error = %v", err)
	}
	if _, err := manager.Update(alias.ID, Address{InboxID: inboxID, Address: alias.Address, Kind: KindAlias, Enabled: false, Restricted: true, UserIDs: []int{agentID}}); err != nil {
		t.Fatalf("disabling historical alias: %v", err)
	}
	// Address grants are canonical. A stale transport restriction must not
	// silently undo the explicit policy configured on this endpoint.
	db.MustExec(`INSERT INTO inbox_access(inbox_id, restricted) VALUES ($1, TRUE)`, inboxID)
	if allowed, err := manager.CanAccess(agentID, alias.ID); err != nil || !allowed {
		t.Fatalf("disabled historical alias with an explicit grant should stay readable: allowed=%v err=%v", allowed, err)
	}
	if _, err := manager.Update(primary.ID, Address{InboxID: inboxID, Address: "renamed@example.test", Kind: KindMailbox, Enabled: true, Restricted: true}); err == nil || !strings.Contains(err.Error(), "owned by its transport") {
		t.Fatalf("renaming primary error = %v", err)
	}
}
