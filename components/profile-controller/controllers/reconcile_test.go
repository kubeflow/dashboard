package controllers

// Tests for Profile controller Reconcile() failure paths.
//
// The existing tests (TestEnforceNamespaceLabelsFromConfig, TestGetPluginSpec) never
// call Reconcile().  These tests fill that gap using the fake client with
// WithStatusSubresource, which mirrors real API-server behaviour: r.Update() cannot
// write to the status subresource, only r.Status().Update() can.
//
// A secondary bug was confirmed and fixed alongside these tests:
// appendErrorConditionAndReturn() previously called r.Update() instead of
// r.Status().Update(), causing ProfileFailed conditions to be silently discarded on
// real clusters. The fix (r.Status().Update()) is verified by
// TestReconcile_NamespaceOwnedByAnother_ProfileFailedPersisted.

import (
	"context"
	"os"
	"testing"

	profilev1 "github.com/kubeflow/dashboard/components/profile-controller/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	istioSecurityClient "istio.io/client-go/pkg/apis/security/v1beta1"
)

// emptyLabelsFile creates a temp YAML file with an empty map and returns its path.
// readDefaultLabelsFromFile calls os.Exit(1) on any read/parse error, so a valid
// (but empty) file must exist at the path set in DefaultNamespaceLabelsPath.
func emptyLabelsFile(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "ns-labels-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp labels file: %v", err)
	}
	if _, err := f.WriteString("{}\n"); err != nil {
		t.Fatalf("failed to write temp labels file: %v", err)
	}
	f.Close()
	return f.Name()
}

func testProfileScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(s))
	require.NoError(t, profilev1.AddToScheme(s))
	require.NoError(t, istioSecurityClient.AddToScheme(s))
	return s
}

func newTestProfile(name, ownerName string) *profilev1.Profile {
	return &profilev1.Profile{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: profilev1.ProfileSpec{
			Owner: rbacv1.Subject{Kind: "User", Name: ownerName},
		},
	}
}

// TestReconcile_NamespaceOwnedByAnother_ProfileFailedPersisted verifies that when a
// namespace already exists but is owned by a different user, Reconcile:
//  1. returns Result{}, nil (no requeue — intentional design)
//  2. persists a ProfileFailed condition via r.Status().Update()
//
// Uses WithStatusSubresource to match real API-server semantics.
func TestReconcile_NamespaceOwnedByAnother_ProfileFailedPersisted(t *testing.T) {
	scheme := testProfileScheme(t)
	profile := newTestProfile("alice", "alice@example.com")
	existingNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "alice",
			Annotations: map[string]string{"owner": "bob@example.com"},
		},
	}

	// WithStatusSubresource: mirrors real API server — only r.Status().Update() writes status.
	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, existingNs).
		WithStatusSubresource(&profilev1.Profile{}).
		Build()

	r := &ProfileReconciler{
		Client:                     cl,
		Scheme:                     scheme,
		Log:                        ctrl.Log,
		UserIdHeader:               "x-goog-authenticated-user-email",
		UserIdPrefix:               "accounts.google.com:",
		WorkloadIdentity:           "",
		DefaultNamespaceLabelsPath: emptyLabelsFile(t),
	}

	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "alice"},
	})

	assert.NoError(t, err, "namespace ownership conflict should not propagate as an error")
	assert.Equal(t, ctrl.Result{}, result, "should return empty Result (no requeue)")

	var got profilev1.Profile
	require.NoError(t, cl.Get(context.Background(), types.NamespacedName{Name: "alice"}, &got))
	t.Logf("persisted conditions: %+v", got.Status.Conditions)

	found := false
	for _, c := range got.Status.Conditions {
		if c.Type == profilev1.ProfileFailed {
			found = true
			t.Logf("ProfileFailed message: %q", c.Message)
		}
	}
	assert.True(t, found,
		"ProfileFailed condition must be persisted by r.Status().Update() — "+
			"if this fails the fix regressed back to r.Update()")
}

// TestReconcile_NamespaceNotFound_CreatesNamespaceAndNoProfileFailed verifies that
// the happy namespace-creation path does not set a ProfileFailed condition.
func TestReconcile_NamespaceNotFound_CreatesNamespaceAndNoProfileFailed(t *testing.T) {
	scheme := testProfileScheme(t)
	profile := newTestProfile("carol", "carol@example.com")

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile).
		WithStatusSubresource(&profilev1.Profile{}).
		Build()

	r := &ProfileReconciler{
		Client:                     cl,
		Scheme:                     scheme,
		Log:                        ctrl.Log,
		UserIdHeader:               "x-goog-authenticated-user-email",
		UserIdPrefix:               "accounts.google.com:",
		WorkloadIdentity:           "",
		DefaultNamespaceLabelsPath: emptyLabelsFile(t),
	}

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "carol"},
	})
	// Sub-controllers (Istio, SA) may fail in a fake environment; that is acceptable.
	// The only invariant: ProfileFailed must NOT be set when namespace creation succeeds.
	t.Logf("Reconcile returned err=%v", err)

	var got profilev1.Profile
	require.NoError(t, cl.Get(context.Background(), types.NamespacedName{Name: "carol"}, &got))
	t.Logf("persisted conditions: %+v", got.Status.Conditions)

	for _, c := range got.Status.Conditions {
		assert.NotEqual(t, profilev1.ProfileFailed, c.Type,
			"ProfileFailed must not be set on a successful namespace creation path")
	}

	// Verify the namespace was actually created
	var ns corev1.Namespace
	assert.NoError(t, cl.Get(context.Background(), types.NamespacedName{Name: "carol"}, &ns),
		"Namespace carol should have been created by Reconcile")
}

// TestReconcile_ProfileNotFound_ReturnsNilError verifies that a missing Profile
// (already deleted) causes Reconcile to return nil without panicking.
func TestReconcile_ProfileNotFound_ReturnsNilError(t *testing.T) {
	scheme := testProfileScheme(t)

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	r := &ProfileReconciler{
		Client:                     cl,
		Scheme:                     scheme,
		Log:                        ctrl.Log,
		UserIdHeader:               "x-goog-authenticated-user-email",
		UserIdPrefix:               "accounts.google.com:",
		WorkloadIdentity:           "",
		DefaultNamespaceLabelsPath: emptyLabelsFile(t),
	}

	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "doesnotexist"},
	})
	assert.NoError(t, err, "missing Profile should be treated as already deleted, not an error")
	assert.Equal(t, ctrl.Result{}, result)
}
