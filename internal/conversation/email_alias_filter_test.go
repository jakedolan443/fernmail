package conversation

import (
	"strings"
	"testing"

	"github.com/jakedolan443/fernmail/internal/dbutil"
)

func TestEmailAliasFilterIsSafeAndParameterized(t *testing.T) {
	filters := `[{"model":"conversations","field":"email_alias","operator":"equals","value":"billing@example.com"}]`
	query, args, err := dbutil.BuildFilterQuery(
		"SELECT 1 FROM conversations WHERE TRUE",
		nil,
		filters,
		ListFilterAllowedFields,
		ListFilterRenderers,
		"UTC",
	)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(query, "billing@example.com") {
		t.Fatalf("alias value was interpolated into SQL: %s", query)
	}
	if len(args) != 1 || args[0] != "billing@example.com" {
		t.Fatalf("args = %#v, want one parameterized alias", args)
	}
}

func TestEmailAliasFilterRejectsUnknownOperator(t *testing.T) {
	filters := `[{"model":"conversations","field":"email_alias","operator":"greater than","value":"billing@example.com"}]`
	if err := dbutil.ValidateFilters(filters, ListFilterAllowedFields, ListFilterRenderers); err == nil {
		t.Fatal("expected unsupported email alias operator to be rejected")
	}
}
