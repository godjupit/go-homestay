package admin

import (
	"reflect"
	"testing"
)

func TestPracticeValidScope(t *testing.T) {
	for _, scope := range []int64{DataScopeAll, DataScopeBusiness, DataScopeCustom, DataScopeSelf} {
		if !validScope(scope) {
			t.Errorf("scope %d should be valid", scope)
		}
	}
	for _, scope := range []int64{-1, 0, 5, 99} {
		if validScope(scope) {
			t.Errorf("scope %d should be invalid", scope)
		}
	}
}

func TestPracticeScopeCondition(t *testing.T) {
	tests := []struct {
		name     string
		auth     *Authorization
		wantSQL  string
		wantArgs []any
	}{
		{"all", &Authorization{AllData: true}, "", nil},
		{"deny by default", &Authorization{}, " AND 1=0", nil},
		{"nil deny", nil, " AND 1=0", nil},
		{"business", &Authorization{BusinessIDs: []int64{2, 3}}, " AND (homestay_business_id IN (?,?))", []any{int64(2), int64(3)}},
		{"self", &Authorization{LinkedUserID: 9}, " AND (user_id = ?)", []any{int64(9)}},
		{"combined", &Authorization{BusinessIDs: []int64{2}, LinkedUserID: 9}, " AND (homestay_business_id IN (?) OR user_id = ?)", []any{int64(2), int64(9)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args := scopeCondition(tt.auth)
			if sql != tt.wantSQL || !reflect.DeepEqual(args, tt.wantArgs) {
				t.Fatalf("scopeCondition() = (%q, %#v), want (%q, %#v)", sql, args, tt.wantSQL, tt.wantArgs)
			}
		})
	}
}
