package kfam

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	profilev1beta1 "github.com/kubeflow/dashboard/components/profile-controller/api/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// mocks
type mockProfileClient struct {
	listResult *profilev1beta1.ProfileList
	listErr    error
	getResult  *profilev1beta1.Profile
	getErr     error
}

func (m *mockProfileClient) Create(profile *profilev1beta1.Profile) (*profilev1beta1.Profile, error) {
	return nil, nil
}

func (m *mockProfileClient) Delete(name string, opts *metav1.DeleteOptions) error { return nil }

func (m *mockProfileClient) Get(name string, opts metav1.GetOptions) (*profilev1beta1.Profile, error) {
	return m.getResult, m.getErr
}

func (m *mockProfileClient) List(opts metav1.ListOptions) (*profilev1beta1.ProfileList, error) {
	return m.listResult, m.listErr
}

func (m *mockProfileClient) Update(profile *profilev1beta1.Profile) (*profilev1beta1.Profile, error) {
	return nil, nil
}

type mockBindingClient struct {
	listResult *BindingEntries
	listErr    error
}

func (m *mockBindingClient) Create(binding *Binding, userIdHeader string, userIdPrefix string, groupsHeader string, groupsClaim string) error {
	return nil
}

func (m *mockBindingClient) Delete(binding *Binding) error { return nil }

func (m *mockBindingClient) List(user string, groups []string, namespaces []string, role string) (*BindingEntries, error) {
	return m.listResult, m.listErr
}

func newTestClient(profileClient ProfileInterface, bindingClient BindingInterface) *KfamV1Alpha1Client {
	return &KfamV1Alpha1Client{
		profileClient: profileClient,
		bindingClient: bindingClient,
	}
}

func buildReadBindingRequest(params map[string]string) *http.Request {
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	req := httptest.NewRequest(http.MethodGet, "/bindings?"+q.Encode(), nil)
	return req
}

func TestReadBinding(t *testing.T) {
	groupBindings := BindingEntries{
		Bindings: []Binding{
			{
				Subject:           &rbacv1.Subject{Kind: rbacv1.GroupKind, Name: "foo"},
				ReferredNamespace: "team-a",
				RoleRef:           &rbacv1.RoleRef{Kind: "ClusterRole", Name: "edit"},
			},
			{
				Subject:           &rbacv1.Subject{Kind: rbacv1.GroupKind, Name: "bar"},
				ReferredNamespace: "team-a",
				RoleRef:           &rbacv1.RoleRef{Kind: "ClusterRole", Name: "edit"},
			},
		},
	}
	emptyBindingsList := BindingEntries{Bindings: nil} // nil because of json:omitempty in BindingEntries

	tests := []struct {
		name          string
		params        map[string]string
		profileClient ProfileInterface
		bindingClient BindingInterface
		wantStatus    int
		wantBindings  *BindingEntries
	}{
		{
			name:          "invalid groups JSON returns 401",
			params:        map[string]string{"namespace": "team-a", "groups": "not-json"},
			profileClient: &mockProfileClient{},
			bindingClient: &mockBindingClient{},
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:          "matching groups returns list BindingEntries with two Bindings",
			params:        map[string]string{"namespace": "team-a", "groups": "[\"foo\", \"bar\"]"},
			profileClient: &mockProfileClient{},
			bindingClient: &mockBindingClient{
				listResult: &groupBindings,
			},
			wantStatus:   http.StatusOK,
			wantBindings: &groupBindings,
		},
		{
			name:          "no matching groups returns empty BindingEntries",
			params:        map[string]string{"namespace": "team-a", "groups": "[\"foo\", \"bar\"]"},
			profileClient: &mockProfileClient{},
			bindingClient: &mockBindingClient{
				listResult: &emptyBindingsList,
			},
			wantStatus:   http.StatusOK,
			wantBindings: &emptyBindingsList,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			client := newTestClient(tt.profileClient, tt.bindingClient)
			req := buildReadBindingRequest(tt.params)
			rr := httptest.NewRecorder()

			// act
			client.ReadBinding(rr, req)

			// assert
			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}

			if tt.wantBindings != nil {
				var got BindingEntries
				if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}
				if !reflect.DeepEqual(got, *tt.wantBindings) {
					t.Errorf("bindings = %+v, want %+v", got, *tt.wantBindings)
				}
			}
		})
	}
}

func TestSanitizeClusterAdmins(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		out  []string
	}{
		{
			name: "empty slice",
			in:   []string{},
			out:  nil,
		},
		{
			name: "nil slice",
			in:   nil,
			out:  nil,
		},
		{
			name: "empty strings",
			in:   []string{"", ""},
			out:  nil,
		},
		{
			name: "whitespace only strings",
			in:   []string{"  ", " "},
			out:  nil,
		},
		{
			name: "valid admins",
			in:   []string{"user1@example.com", "user2@example.com"},
			out:  []string{"user1@example.com", "user2@example.com"},
		},
		{
			name: "admins with leading spaces",
			in:   []string{"  user1@example.com", " user2@example.com"},
			out:  []string{"user1@example.com", "user2@example.com"},
		},
		{
			name: "admins with trailing spaces",
			in:   []string{"user1@example.com  ", "user2@example.com "},
			out:  []string{"user1@example.com", "user2@example.com"},
		},
		{
			name: "admins with leading and trailing spaces",
			in:   []string{"  user1@example.com  ", " user2@example.com "},
			out:  []string{"user1@example.com", "user2@example.com"},
		},
		{
			name: "mixed empty, whitespace, and admins",
			in:   []string{"", "user1@example.com", "   ", "user2@example.com  "},
			out:  []string{"user1@example.com", "user2@example.com"},
		},
		{
			name: "duplicate admins",
			in:   []string{"user1@example.com", "user1@example.com"},
			out:  []string{"user1@example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitized := sanitizeClusterAdmins(tt.in)
			if !reflect.DeepEqual(sanitized, tt.out) {
				t.Errorf("output: %v, expected: %v", sanitized, tt.out)
			}
		})
	}
}
