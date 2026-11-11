package kfam

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	profilev1beta1 "github.com/kubeflow/dashboard/components/profile-controller/api/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

// ---------------------------------------------------------------------------
// Test fakes
// ---------------------------------------------------------------------------

type fakeProfileClient struct {
	owner       string // name returned by Get as Spec.Owner.Name; empty -> Get errors
	listErr     error  // returned by List when set
	getCalls    []string
	deleteCalls []string
}

func (f *fakeProfileClient) Create(profile *profilev1beta1.Profile) (*profilev1beta1.Profile, error) {
	return profile, nil
}

func (f *fakeProfileClient) Delete(name string, opts *metav1.DeleteOptions) error {
	f.deleteCalls = append(f.deleteCalls, name)
	return nil
}

func (f *fakeProfileClient) Get(name string, opts metav1.GetOptions) (*profilev1beta1.Profile, error) {
	f.getCalls = append(f.getCalls, name)
	if f.owner == "" {
		return nil, errors.New("profile not found")
	}
	return &profilev1beta1.Profile{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       profilev1beta1.ProfileSpec{Owner: rbacv1.Subject{Kind: "User", Name: f.owner}},
	}, nil
}

func (f *fakeProfileClient) List(opts metav1.ListOptions) (*profilev1beta1.ProfileList, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return &profilev1beta1.ProfileList{}, nil
}

func (f *fakeProfileClient) Update(profile *profilev1beta1.Profile) (*profilev1beta1.Profile, error) {
	return profile, nil
}

type createBindingCall struct {
	referredNamespace string
	userName          string
	userIdHeader      string
	userIdPrefix      string
}

type fakeBindingClient struct {
	createCalls []createBindingCall
	deleteCalls []*Binding
}

func (f *fakeBindingClient) Create(binding *Binding, userIdHeader string, userIdPrefix string) error {
	f.createCalls = append(f.createCalls, createBindingCall{
		referredNamespace: binding.ReferredNamespace,
		userName:          binding.User.Name,
		userIdHeader:      userIdHeader,
		userIdPrefix:      userIdPrefix,
	})
	return nil
}

func (f *fakeBindingClient) Delete(binding *Binding) error {
	f.deleteCalls = append(f.deleteCalls, binding)
	return nil
}

func (f *fakeBindingClient) List(user string, namespaces []string, role string) (*BindingEntries, error) {
	return &BindingEntries{Bindings: []Binding{}}, nil
}

// ---------------------------------------------------------------------------
// getUserEmail unit tests (regression: unchecked slicing used to panic)
// ---------------------------------------------------------------------------

func TestGetUserEmail(t *testing.T) {
	tests := []struct {
		name        string
		headerName  string
		prefix      string
		headerValue string // "" means the header is not set at all
		want        string
		wantErr     error
	}{
		{
			name:        "strips matching prefix",
			headerName:  "x-goog-authenticated-user-email",
			prefix:      "accounts.google.com:",
			headerValue: "accounts.google.com:owner@example.com",
			want:        "owner@example.com",
		},
		{
			name:        "empty prefix returns raw value",
			headerName:  "kubeflow-userid",
			prefix:      "",
			headerValue: "owner@example.com",
			want:        "owner@example.com",
		},
		{
			name:        "missing header with non-empty prefix errors (regression: panicked)",
			headerName:  "x-goog-authenticated-user-email",
			prefix:      "accounts.google.com:",
			headerValue: "",
			wantErr:     errMissingIdentityHeader,
		},
		{
			name:        "missing header with empty prefix errors",
			headerName:  "kubeflow-userid",
			prefix:      "",
			headerValue: "",
			wantErr:     errMissingIdentityHeader,
		},
		{
			name:        "value shorter than prefix errors (regression: panicked)",
			headerName:  "x-goog-authenticated-user-email",
			prefix:      "accounts.google.com:",
			headerValue: "owner",
			wantErr:     errIdentityPrefixMismatch,
		},
		{
			name:        "value not starting with prefix errors (regression: truncated to a wrong identity)",
			headerName:  "x-goog-authenticated-user-email",
			prefix:      "accounts.google.com:",
			headerValue: "not-a-prefixed-value-longer-than-twenty@example.com",
			wantErr:     errIdentityPrefixMismatch,
		},
		{
			name:        "value equal to prefix errors (empty user after stripping)",
			headerName:  "x-goog-authenticated-user-email",
			prefix:      "accounts.google.com:",
			headerValue: "accounts.google.com:",
			wantErr:     errEmptyIdentityAfterStrip,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kc := &KfamV1Alpha1Client{
				userIdHeader: tt.headerName,
				userIdPrefix: tt.prefix,
			}
			header := http.Header{}
			if tt.headerValue != "" {
				header.Set(tt.headerName, tt.headerValue)
			}

			email, err := kc.getUserEmail(header)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got email %q and nil error", tt.wantErr, email)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if email != tt.want {
				t.Errorf("email: %q, expected: %q", email, tt.want)
			}
		})
	}
}

func TestIdentityErrorKind(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "missing header",
			err:  errMissingIdentityHeader,
			want: "missing identity header",
		},
		{
			name: "prefix mismatch",
			err:  errIdentityPrefixMismatch,
			want: "identity prefix mismatch",
		},
		{
			name: "empty after strip",
			err:  errEmptyIdentityAfterStrip,
			want: "empty identity after prefix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := identityErrorKind(tt.err); got != tt.want {
				t.Errorf("kind: %q, expected: %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Handler-level tests (regression: the mutating endpoints used to panic and
// drop the connection instead of answering with a status code)
// ---------------------------------------------------------------------------

// newIAPStyleClient returns a client configured like the GCP/IAP-style flag
// defaults: a non-empty userid prefix. This is the configuration that used to
// panic on a missing header.
func newIAPStyleClient(profile ProfileInterface, binding BindingInterface) *KfamV1Alpha1Client {
	return &KfamV1Alpha1Client{
		profileClient: profile,
		bindingClient: binding,
		clusterAdmins: []string{"admin@example.com"},
		userIdHeader:  "x-goog-authenticated-user-email",
		userIdPrefix:  "accounts.google.com:",
	}
}

func TestMutatingEndpointsReturn401WithoutIdentityHeader(t *testing.T) {
	kc := newIAPStyleClient(&fakeProfileClient{owner: "owner@example.com"}, &fakeBindingClient{})
	server := httptest.NewServer(NewRouter(kc))
	defer server.Close()

	const bindingBody = `{"user":{"kind":"User","name":"bob@example.com"},"referredNamespace":"teamx","RoleRef":{"kind":"ClusterRole","name":"edit"}}`
	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{
			name:   "create binding",
			method: http.MethodPost,
			path:   "/kfam/v1/bindings",
			body:   bindingBody,
		},
		{
			name:   "delete binding",
			method: http.MethodDelete,
			path:   "/kfam/v1/bindings",
			body:   bindingBody,
		},
		{
			name:   "delete profile",
			method: http.MethodDelete,
			path:   "/kfam/v1/profiles/teamx",
			body:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, server.URL+tt.path, strings.NewReader(tt.body))
			if err != nil {
				t.Fatal(err)
			}
			// A panic inside the handler tears down the connection before any
			// status is written; the request below must instead succeed at the
			// transport level and yield a 401.
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed (a handler panic drops the connection): %v", err)
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("status: %d, expected: 401, body: %s", resp.StatusCode, body)
			}
			if len(strings.TrimSpace(string(body))) == 0 {
				t.Error("expected a non-empty error body explaining the missing identity header")
			}
		})
	}
}

func TestMutatingEndpointsReturn401OnPrefixMismatch(t *testing.T) {
	kc := newIAPStyleClient(&fakeProfileClient{owner: "owner@example.com"}, &fakeBindingClient{})
	server := httptest.NewServer(NewRouter(kc))
	defer server.Close()

	const bindingBody = `{"user":{"kind":"User","name":"bob@example.com"},"referredNamespace":"teamx","RoleRef":{"kind":"ClusterRole","name":"edit"}}`
	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "create binding", method: http.MethodPost, path: "/kfam/v1/bindings", body: bindingBody},
		{name: "delete binding", method: http.MethodDelete, path: "/kfam/v1/bindings", body: bindingBody},
		{name: "delete profile", method: http.MethodDelete, path: "/kfam/v1/profiles/teamx", body: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, server.URL+tt.path, strings.NewReader(tt.body))
			if err != nil {
				t.Fatal(err)
			}
			// Header present but not carrying the configured prefix: must be
			// rejected, not truncated to a wrong identity.
			req.Header.Set("x-goog-authenticated-user-email", "someothergateway:owner@example.com")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("status: %d, expected: 401, body: %s", resp.StatusCode, body)
			}
			if len(strings.TrimSpace(string(body))) == 0 {
				t.Error("expected a non-empty error body explaining the prefix mismatch")
			}
		})
	}
}

func TestMutatingEndpointsWithValidIdentity(t *testing.T) {
	const bindingBody = `{"user":{"kind":"User","name":"bob@example.com"},"referredNamespace":"teamx","RoleRef":{"kind":"ClusterRole","name":"edit"}}`

	t.Run("owner can create a binding", func(t *testing.T) {
		profile := &fakeProfileClient{owner: "owner@example.com"}
		binding := &fakeBindingClient{}
		kc := newIAPStyleClient(profile, binding)
		server := httptest.NewServer(NewRouter(kc))
		defer server.Close()

		req, err := http.NewRequest(http.MethodPost, server.URL+"/kfam/v1/bindings", strings.NewReader(bindingBody))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("x-goog-authenticated-user-email", "accounts.google.com:owner@example.com")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("status: %d, expected: 200, body: %s", resp.StatusCode, body)
		}
		if len(binding.createCalls) != 1 {
			t.Fatalf("create calls: %d, expected: 1", len(binding.createCalls))
		}
		call := binding.createCalls[0]
		if call.referredNamespace != "teamx" || call.userName != "bob@example.com" {
			t.Errorf("recorded binding: %+v, expected namespace teamx user bob@example.com", call)
		}
		// The header/prefix configuration must be passed through so the Istio
		// AuthorizationPolicy matches the deployment's identity format.
		if call.userIdHeader != "x-goog-authenticated-user-email" || call.userIdPrefix != "accounts.google.com:" {
			t.Errorf("recorded header config: %q / %q, expected x-goog-authenticated-user-email / accounts.google.com:",
				call.userIdHeader, call.userIdPrefix)
		}
	})

	t.Run("owner can delete a profile", func(t *testing.T) {
		profile := &fakeProfileClient{owner: "owner@example.com"}
		kc := newIAPStyleClient(profile, &fakeBindingClient{})
		server := httptest.NewServer(NewRouter(kc))
		defer server.Close()

		req, err := http.NewRequest(http.MethodDelete, server.URL+"/kfam/v1/profiles/teamx", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("x-goog-authenticated-user-email", "accounts.google.com:owner@example.com")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("status: %d, expected: 200, body: %s", resp.StatusCode, body)
		}
		if len(profile.deleteCalls) != 1 || profile.deleteCalls[0] != "teamx" {
			t.Errorf("delete calls: %v, expected [teamx]", profile.deleteCalls)
		}
	})

	// The pre-existing permission-denied behavior must be preserved: bindings
	// answer 403 and profile deletion 401, both without a body.
	t.Run("non-owner is denied with the existing status codes", func(t *testing.T) {
		profile := &fakeProfileClient{owner: "someoneelse@example.com"}
		binding := &fakeBindingClient{}
		kc := newIAPStyleClient(profile, binding)
		server := httptest.NewServer(NewRouter(kc))
		defer server.Close()

		tests := []struct {
			name   string
			method string
			path   string
			body   string
			want   int
		}{
			{name: "create binding", method: http.MethodPost, path: "/kfam/v1/bindings", body: bindingBody, want: http.StatusForbidden},
			{name: "delete binding", method: http.MethodDelete, path: "/kfam/v1/bindings", body: bindingBody, want: http.StatusForbidden},
			{name: "delete profile", method: http.MethodDelete, path: "/kfam/v1/profiles/teamx", body: "", want: http.StatusUnauthorized},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req, err := http.NewRequest(tt.method, server.URL+tt.path, strings.NewReader(tt.body))
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("x-goog-authenticated-user-email", "accounts.google.com:owner@example.com")
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				if resp.StatusCode != tt.want {
					t.Fatalf("status: %d, expected: %d, body: %s", resp.StatusCode, tt.want, body)
				}
				if len(body) != 0 {
					t.Errorf("expected an empty body on the permission-denied path, got: %s", body)
				}
			})
		}
	})
}

// ---------------------------------------------------------------------------
// Latent panic regressions in ReadBinding and QueryClusterAdmin
// ---------------------------------------------------------------------------

func TestReadBindingReturnsOnProfileListError(t *testing.T) {
	// profileClient.List fails: before the fix, the missing return made the
	// handler dereference a nil profile list right after writing the response.
	profile := &fakeProfileClient{listErr: errors.New("list failed")}
	kc := newIAPStyleClient(profile, &fakeBindingClient{})
	server := httptest.NewServer(NewRouter(kc))
	defer server.Close()

	resp, err := http.Get(server.URL + "/kfam/v1/bindings")
	if err != nil {
		t.Fatalf("request failed (a handler panic drops the connection): %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status: %d, expected: 403, body: %s", resp.StatusCode, body)
	}
	if len(body) == 0 {
		t.Error("expected the error message in the response body")
	}
}

// failingResponseWriter simulates a client that went away mid-response.
type failingResponseWriter struct {
	header http.Header
	code   int
}

func (f *failingResponseWriter) Header() http.Header { return f.header }

func (f *failingResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("client disconnected")
}

func (f *failingResponseWriter) WriteHeader(code int) { f.code = code }

func TestQueryClusterAdminOnWriteFailure(t *testing.T) {
	// Before the fix, the write-failure branch called err.Error() on a nil err
	// and panicked; it must instead report a 500.
	kc := newIAPStyleClient(&fakeProfileClient{}, &fakeBindingClient{})
	req := httptest.NewRequest(http.MethodGet, "/kfam/v1/role/clusteradmin?user=admin@example.com", nil)
	w := &failingResponseWriter{header: http.Header{}}

	// A panic here would fail the test via the test goroutine.
	kc.QueryClusterAdmin(w, req)

	if w.code != http.StatusInternalServerError {
		t.Fatalf("status: %d, expected: 500", w.code)
	}
}
