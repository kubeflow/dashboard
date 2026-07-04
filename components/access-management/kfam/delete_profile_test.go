package kfam

import (
	"net/http"
	"net/http/httptest"
	"testing"

	profilev1beta1 "github.com/kubeflow/dashboard/components/profile-controller/api/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type stubProfileClient struct {
	getName    string
	deleteName string
}

func (s *stubProfileClient) Create(profile *profilev1beta1.Profile) (*profilev1beta1.Profile, error) {
	return profile, nil
}

func (s *stubProfileClient) Delete(name string, opts *metav1.DeleteOptions) error {
	s.deleteName = name
	return nil
}

func (s *stubProfileClient) Get(name string, opts metav1.GetOptions) (*profilev1beta1.Profile, error) {
	s.getName = name
	return &profilev1beta1.Profile{
		Spec: profilev1beta1.ProfileSpec{
			Owner: rbacv1.Subject{
				Name: "user@example.com",
			},
		},
	}, nil
}

func (s *stubProfileClient) List(opts metav1.ListOptions) (*profilev1beta1.ProfileList, error) {
	return &profilev1beta1.ProfileList{}, nil
}

func (s *stubProfileClient) Update(profile *profilev1beta1.Profile) (*profilev1beta1.Profile, error) {
	return profile, nil
}

type stubBindingClient struct{}

func (s *stubBindingClient) Create(binding *Binding, userIdHeader string, userIdPrefix string) error {
	return nil
}

func (s *stubBindingClient) Delete(binding *Binding) error {
	return nil
}

func (s *stubBindingClient) List(user string, namespaces []string, role string) (*BindingEntries, error) {
	return &BindingEntries{}, nil
}

func TestDeleteProfileUsesPathVariable(t *testing.T) {
	profileClient := &stubProfileClient{}
	client := &KfamV1Alpha1Client{
		profileClient: profileClient,
		bindingClient: &stubBindingClient{},
		userIdHeader:  "x-user",
		userIdPrefix:  "accounts.google.com:",
	}

	req := httptest.NewRequest(http.MethodDelete, "/kfam/v1/profiles/test-profile?dryRun=All", nil)
	req.Header.Set("x-user", "accounts.google.com:user@example.com")
	rec := httptest.NewRecorder()

	NewRouter(client).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", rec.Code, http.StatusOK)
	}
	if profileClient.getName != "test-profile" {
		t.Fatalf("profile lookup used %q, want %q", profileClient.getName, "test-profile")
	}
	if profileClient.deleteName != "test-profile" {
		t.Fatalf("profile delete used %q, want %q", profileClient.deleteName, "test-profile")
	}
}
