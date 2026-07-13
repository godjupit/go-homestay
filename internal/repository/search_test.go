package repository

import (
	"reflect"
	"testing"

	"gin-looklook/internal/model"
)

func TestScopeCondition(t *testing.T) {
	tests := []struct {
		name     string
		auth     *model.AdminAuthorization
		wantSQL  string
		wantArgs []any
	}{
		{name: "all", auth: &model.AdminAuthorization{AllData: true}, wantSQL: ""},
		{name: "deny by default", auth: &model.AdminAuthorization{}, wantSQL: " AND 1=0"},
		{name: "business and self", auth: &model.AdminAuthorization{BusinessIDs: []int64{3, 7}, LinkedUserID: 9}, wantSQL: " AND (homestay_business_id IN (?,?) OR user_id=?)", wantArgs: []any{int64(3), int64(7), int64(9)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs := scopeCondition(tt.auth)
			if gotSQL != tt.wantSQL || !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Fatalf("scopeCondition()=(%q,%v), want (%q,%v)", gotSQL, gotArgs, tt.wantSQL, tt.wantArgs)
			}
		})
	}
}
