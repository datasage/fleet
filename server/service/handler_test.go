package service

import (
	"bytes"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/gorilla/mux"
	"github.com/kolide/fleet/server/config"
	"github.com/kolide/fleet/server/datastore/inmem"
	"github.com/kolide/fleet/server/kolide"
	"github.com/kolide/fleet/server/mock"
	"github.com/stretchr/testify/assert"
)

func TestAPIRoutes(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)

	r := mux.NewRouter()
	ke := MakeKolideServerEndpoints(svc, "CHANGEME")
	kh := makeKolideKitHandlers(ke, nil)
	attachKolideAPIRoutes(r, kh)
	handler := mux.NewRouter()
	handler.PathPrefix("/").Handler(r)

	var routes = []struct {
		verb string
		uri  string
	}{
		{
			verb: "POST",
			uri:  "/api/v1/kolide/users",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/users",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/users/1",
		},
		{
			verb: "PATCH",
			uri:  "/api/v1/kolide/users/1",
		},
		{
			verb: "POST",
			uri:  "/api/v1/kolide/login",
		},
		{
			verb: "POST",
			uri:  "/api/v1/kolide/forgot_password",
		},
		{
			verb: "POST",
			uri:  "/api/v1/kolide/reset_password",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/me",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/config",
		},
		{
			verb: "PATCH",
			uri:  "/api/v1/kolide/config",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/invites",
		},
		{
			verb: "POST",
			uri:  "/api/v1/kolide/invites",
		},
		{
			verb: "DELETE",
			uri:  "/api/v1/kolide/invites/1",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/queries/1",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/queries",
		},
		{
			verb: "POST",
			uri:  "/api/v1/kolide/queries",
		},
		{
			verb: "PATCH",
			uri:  "/api/v1/kolide/queries/1",
		},
		{
			verb: "DELETE",
			uri:  "/api/v1/kolide/queries/1",
		},
		{
			verb: "POST",
			uri:  "/api/v1/kolide/queries/delete",
		},
		{
			verb: "POST",
			uri:  "/api/v1/kolide/queries/run",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/packs/1",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/packs",
		},
		{
			verb: "POST",
			uri:  "/api/v1/kolide/packs",
		},
		{
			verb: "PATCH",
			uri:  "/api/v1/kolide/packs/1",
		},
		{
			verb: "DELETE",
			uri:  "/api/v1/kolide/packs/1",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/packs/1/scheduled",
		},
		{
			verb: "POST",
			uri:  "/api/v1/kolide/schedule",
		},
		{
			verb: "DELETE",
			uri:  "/api/v1/kolide/schedule/1",
		},
		{
			verb: "PATCH",
			uri:  "/api/v1/kolide/schedule/1",
		}, {
			verb: "POST",
			uri:  "/api/v1/osquery/enroll",
		},
		{
			verb: "POST",
			uri:  "/api/v1/osquery/config",
		},
		{
			verb: "POST",
			uri:  "/api/v1/osquery/distributed/read",
		},
		{
			verb: "POST",
			uri:  "/api/v1/osquery/distributed/write",
		},
		{
			verb: "POST",
			uri:  "/api/v1/osquery/log",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/labels/1",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/labels",
		},
		{
			verb: "POST",
			uri:  "/api/v1/kolide/labels",
		},
		{
			verb: "DELETE",
			uri:  "/api/v1/kolide/labels/1",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/hosts/1",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/hosts",
		},
		{
			verb: "DELETE",
			uri:  "/api/v1/kolide/hosts/1",
		},
		{
			verb: "GET",
			uri:  "/api/v1/kolide/host_summary",
		},
	}

	for _, route := range routes {
		t.Run(fmt.Sprintf(": %v", route.uri), func(st *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(
				recorder,
				httptest.NewRequest(route.verb, route.uri, nil),
			)
			assert.NotEqual(st, 404, recorder.Code)
		})
	}
}

func TestModifyUserPermissions(t *testing.T) {
	var (
		admin, enabled bool
		uid            uint
	)
	ms := new(mock.Store)
	ms.SessionByKeyFunc = func(key string) (*kolide.Session, error) {
		return &kolide.Session{AccessedAt: time.Now(), UserID: uid, ID: 1}, nil
	}
	ms.DestroySessionFunc = func(session *kolide.Session) error {
		return nil
	}
	ms.MarkSessionAccessedFunc = func(session *kolide.Session) error {
		return nil
	}
	ms.UserByIDFunc = func(id uint) (*kolide.User, error) {
		return &kolide.User{ID: id, Enabled: enabled, Admin: admin}, nil
	}
	ms.SaveUserFunc = func(u *kolide.User) error {
		// Return an error so that the endpoint returns
		return errors.New("foo")
	}

	svc, err := newTestService(ms, nil)
	assert.Nil(t, err)

	handler := MakeHandler(svc, "CHANGEME", log.NewNopLogger())

	testCases := []struct {
		ActingUserID      uint
		ActingUserAdmin   bool
		ActingUserEnabled bool
		TargetUserID      uint
		Authorized        bool
	}{
		// Disabled regular user
		{
			ActingUserID:      1,
			ActingUserAdmin:   false,
			ActingUserEnabled: false,
			TargetUserID:      1,
			Authorized:        false,
		},
		// Enabled regular user acting on self
		{
			ActingUserID:      1,
			ActingUserAdmin:   false,
			ActingUserEnabled: true,
			TargetUserID:      1,
			Authorized:        true,
		},
		// Enabled regular user acting on other
		{
			ActingUserID:      2,
			ActingUserAdmin:   false,
			ActingUserEnabled: true,
			TargetUserID:      1,
			Authorized:        false,
		},
		// Disabled admin user
		{
			ActingUserID:      1,
			ActingUserAdmin:   true,
			ActingUserEnabled: false,
			TargetUserID:      1,
			Authorized:        false,
		},
		// Enabled admin user acting on self
		{
			ActingUserID:      1,
			ActingUserAdmin:   true,
			ActingUserEnabled: true,
			TargetUserID:      1,
			Authorized:        true,
		},
		// Enabled admin user acting on other
		{
			ActingUserID:      2,
			ActingUserAdmin:   true,
			ActingUserEnabled: true,
			TargetUserID:      1,
			Authorized:        true,
		},
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			// Set user params
			uid = tt.ActingUserID
			admin, enabled = tt.ActingUserAdmin, tt.ActingUserEnabled

			recorder := httptest.NewRecorder()
			path := fmt.Sprintf("/api/v1/kolide/users/%d", tt.TargetUserID)
			request := httptest.NewRequest("PATCH", path, bytes.NewBufferString("{}"))
			// Bearer token generated with session key CHANGEME on jwt.io
			request.Header.Add("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzZXNzaW9uX2tleSI6ImZsb29wIn0.ukCPTFvgSJrXbHH2QeAMx3EKwoMh1OmhP3xXxy5I-Wk")

			handler.ServeHTTP(recorder, request)
			if tt.Authorized {
				assert.NotEqual(t, 403, recorder.Code)
			} else {
				assert.Equal(t, 403, recorder.Code)
			}

		})
	}

}
