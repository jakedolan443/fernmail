package migrations

import (
	"testing"

	"github.com/jakedolan443/fernmail/internal/testutil"
)

func TestAddressFirstUpgradePromotesAliasesAndRetiresViews(t *testing.T) {
	db := testutil.NewDB(t, "address_first_upgrade")
	db.MustExec(`
		CREATE TYPE view_visibility AS ENUM ('all', 'team', 'user');
		CREATE TABLE views (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			filters JSONB NOT NULL,
			visibility view_visibility NOT NULL,
			user_id BIGINT REFERENCES users(id),
			team_id INTEGER REFERENCES teams(id)
		);
		INSERT INTO inboxes(name, channel, "from", config)
		VALUES ('Support transport', 'email', 'director@example.test',
		'{"email_aliases":[
			{"address":"director@example.test","enabled":true,"default":true},
			{"address":"support@example.test","display_name":"Support","enabled":true},
			{"address":"press@example.test","display_name":"Press","enabled":true}
		]}'::jsonb);
		INSERT INTO users(type, email, first_name) VALUES
			('agent', 'owner@example.test', 'Owner'),
			('contact', 'customer@example.test', 'Customer');
		INSERT INTO teams(name, conversation_assignment_type) VALUES ('Support', 'Manual');
		INSERT INTO team_members(team_id, user_id)
		SELECT t.id, u.id FROM teams t CROSS JOIN users u
		WHERE t.name='Support' AND u.email='owner@example.test';
		INSERT INTO conversations(contact_id, inbox_id, status_id, meta)
		SELECT u.id, i.id, s.id, '{"email_alias":"support@example.test"}'::jsonb
		FROM users u CROSS JOIN inboxes i CROSS JOIN conversation_statuses s
		WHERE u.email='customer@example.test' AND s.name='Open';
		INSERT INTO views(name, filters, visibility, user_id)
		SELECT 'Director', '[{"model":"conversations","field":"email_alias","operator":"equals","value":"director@example.test"}]'::jsonb,
		'user', id FROM users WHERE email='owner@example.test';
		INSERT INTO views(name, filters, visibility, team_id)
		SELECT 'Support', '[{"model":"conversations","field":"email_alias","operator":"equals","value":"support@example.test"}]'::jsonb,
		'team', id FROM teams WHERE name='Support';
		INSERT INTO views(name, filters, visibility)
		VALUES ('Press', '[{"model":"conversations","field":"email_alias","operator":"equals","value":"press@example.test"}]'::jsonb, 'all');
	`)

	for range 2 {
		if err := V3_5_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}

	var ok bool
	for _, query := range []string{
		`SELECT count(*) = 3 FROM email_addresses`,
		`SELECT count(*) = 1 FROM email_address_users`,
		`SELECT count(*) = 1 FROM email_address_teams`,
		`SELECT address_id IS NOT NULL FROM conversations LIMIT 1`,
		`SELECT restricted = FALSE FROM email_addresses WHERE address='press@example.test'`,
		`SELECT to_regclass('views') IS NULL`,
		`SELECT NOT ('view:manage' = ANY(permissions)) AND NOT ('shared_views:manage' = ANY(permissions)) AND NOT ('conversations:write' = ANY(permissions)) FROM roles WHERE name='Admin'`,
	} {
		if err := db.Get(&ok, query); err != nil || !ok {
			t.Fatalf("%s: %v %v", query, ok, err)
		}
	}
}

func TestAddressFirstUpgradeCopiesLegacyRestrictedMailboxAccess(t *testing.T) {
	db := testutil.NewDB(t, "address_first_inbox_access")
	db.MustExec(`
		INSERT INTO inboxes(name, channel, "from") VALUES ('Mail transport', 'email', 'support@example.test');
		INSERT INTO users(type, email, first_name) VALUES ('agent', 'reader@example.test', 'Reader');
		INSERT INTO inbox_access(inbox_id, restricted) SELECT id, TRUE FROM inboxes WHERE name='Mail transport';
		INSERT INTO inbox_users(inbox_id, user_id)
		SELECT i.id, u.id FROM inboxes i CROSS JOIN users u
		WHERE i.name='Mail transport' AND u.email='reader@example.test';
	`)

	if err := V3_5_0(db, nil, nil); err != nil {
		t.Fatal(err)
	}

	var allowed, restricted bool
	if err := db.Get(&allowed, `SELECT can_access_email_address(a.id, u.id)
		FROM email_addresses a CROSS JOIN users u
		WHERE a.address='support@example.test' AND u.email='reader@example.test'`); err != nil || !allowed {
		t.Fatalf("copied legacy access: allowed=%v err=%v", allowed, err)
	}
	if err := db.Get(&restricted, `SELECT restricted FROM email_addresses WHERE address='support@example.test'`); err != nil || !restricted {
		t.Fatalf("restricted address: restricted=%v err=%v", restricted, err)
	}
}
