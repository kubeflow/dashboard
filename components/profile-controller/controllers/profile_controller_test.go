package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	profilev1 "github.com/kubeflow/dashboard/components/profile-controller/api/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	istiosecurityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type namespaceLabelSuite struct {
	current  corev1.Namespace
	labels   map[string]string
	expected corev1.Namespace
}

func TestEnforceNamespaceLabelsFromConfig(t *testing.T) {
	name := "test-namespace"
	tests := []namespaceLabelSuite{
		{
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
			},
			map[string]string{
				"katib.kubeflow.org/metrics-collector-injection": "enabled",
				"serving.kubeflow.org/inferenceservice":          "enabled",
				"pipelines.kubeflow.org/enabled":                 "true",
				"app.kubernetes.io/part-of":                      "kubeflow-profile",
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"katib.kubeflow.org/metrics-collector-injection": "enabled",
						"serving.kubeflow.org/inferenceservice":          "enabled",
						"pipelines.kubeflow.org/enabled":                 "true",
						"app.kubernetes.io/part-of":                      "kubeflow-profile",
					},
					Name: name,
				},
			},
		},
		{
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"user-name":                             "Jim",
						"serving.kubeflow.org/inferenceservice": "disabled",
					},
					Name: name,
				},
			},
			map[string]string{
				"katib.kubeflow.org/metrics-collector-injection": "enabled",
				"serving.kubeflow.org/inferenceservice":          "enabled",
				"pipelines.kubeflow.org/enabled":                 "true",
				"app.kubernetes.io/part-of":                      "kubeflow-profile",
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"user-name": "Jim",
						"katib.kubeflow.org/metrics-collector-injection": "enabled",
						"serving.kubeflow.org/inferenceservice":          "disabled",
						"pipelines.kubeflow.org/enabled":                 "true",
						"app.kubernetes.io/part-of":                      "kubeflow-profile",
					},
					Name: name,
				},
			},
		},
		{
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"user-name":     "Jim",
						"removal-label": "enabled",
					},
					Name: name,
				},
			},
			map[string]string{
				"katib.kubeflow.org/metrics-collector-injection": "enabled",
				"serving.kubeflow.org/inferenceservice":          "enabled",
				"pipelines.kubeflow.org/enabled":                 "true",
				"app.kubernetes.io/part-of":                      "kubeflow-profile",
				"removal-label":                                  "",
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"user-name": "Jim",
						"katib.kubeflow.org/metrics-collector-injection": "enabled",
						"serving.kubeflow.org/inferenceservice":          "enabled",
						"pipelines.kubeflow.org/enabled":                 "true",
						"app.kubernetes.io/part-of":                      "kubeflow-profile",
					},
					Name: name,
				},
			},
		},
	}
	for _, test := range tests {
		setNamespaceLabels(&test.current, test.labels)
		if !reflect.DeepEqual(&test.expected, &test.current) {
			t.Errorf("Expect:\n%v; Output:\n%v", &test.expected, &test.current)
		}
	}
}

type getPluginSpecSuite struct {
	profile         *profilev1.Profile
	expectedPlugins []Plugin
}

func TestGetPluginSpec(t *testing.T) {
	role_arn := "arn:aws:iam::123456789012:role/test-iam-role"
	gcp_sa := "kubeflow2@project-id.iam.gserviceaccount.com"
	tests := []getPluginSpecSuite{
		{
			&profilev1.Profile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "aws-user-profile",
					Namespace: "k8snamespace",
				},
				Spec: profilev1.ProfileSpec{
					Plugins: []profilev1.Plugin{
						{
							TypeMeta: metav1.TypeMeta{
								Kind: KIND_AWS_IAM_FOR_SERVICE_ACCOUNT,
							},
							Spec: &runtime.RawExtension{
								Raw: []byte(fmt.Sprintf(`{"awsIamRole": "%v"}`, role_arn)),
							},
						},
					},
				},
			},
			[]Plugin{
				&AwsIAMForServiceAccount{
					AwsIAMRole: role_arn,
				},
			},
		},
		{
			&profilev1.Profile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "gcp-user-profile",
					Namespace: "k8snamespace",
				},
				Spec: profilev1.ProfileSpec{
					Plugins: []profilev1.Plugin{
						{
							TypeMeta: metav1.TypeMeta{
								Kind: KIND_WORKLOAD_IDENTITY,
							},
							Spec: &runtime.RawExtension{
								Raw: []byte(fmt.Sprintf(`{"gcpServiceAccount": "%v"}`, gcp_sa)),
							},
						},
					},
				},
			},
			[]Plugin{
				&GcpWorkloadIdentity{
					GcpServiceAccount: gcp_sa,
				},
			},
		},
	}
	for _, test := range tests {
		loadedPlugins, err := createMockReconciler().GetPluginSpec(test.profile)

		assert.Nil(t, err)
		if !reflect.DeepEqual(&test.expectedPlugins, &loadedPlugins) {
			expected, _ := json.Marshal(test.expectedPlugins)
			found, _ := json.Marshal(loadedPlugins)
			t.Errorf("Test: %v. Expected:\n%v\nFound:\n%v", test.profile.Name, string(expected), string(found))
		}
	}
}

func createMockReconciler() *ProfileReconciler {
	reconciler := &ProfileReconciler{
		Scheme:                     runtime.NewScheme(),
		Log:                        ctrl.Log,
		UserIdHeader:               "dummy",
		UserIdPrefix:               "dummy",
		WorkloadIdentity:           "dummy",
		DefaultNamespaceLabelsPath: "dummy",
	}
	return reconciler
}

// =========================================================================
// New Ginkgo tests — Namespace Adoption
//
// These run Reconcile() synchronously against a fake client (no envtest,
// no live manager). Two reasons:
//  1. This suite's BeforeSuite never starts a controller manager, so an
//     Eventually()-style test against a live reconcile loop would just
//     time out.
//  2. Reconcile() also touches Istio's AuthorizationPolicy CRD, which
//     isn't vendored for envtest in this repo — a real apiserver can't
//     satisfy that request, but a fake client only needs the Go type
//     registered in its scheme, not a real CRD installed.
// =========================================================================

// newFakeProfileReconciler builds a ProfileReconciler backed by a fresh
// in-memory fake client seeded with the given objects.
func newFakeProfileReconciler(objs ...client.Object) *ProfileReconciler {
	scheme := runtime.NewScheme()
	Expect(corev1.AddToScheme(scheme)).To(Succeed())
	Expect(rbacv1.AddToScheme(scheme)).To(Succeed())
	Expect(profilev1.AddToScheme(scheme)).To(Succeed())
	Expect(istiosecurityv1beta1.AddToScheme(scheme)).To(Succeed())

	return &ProfileReconciler{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(objs...).
			Build(),
		Scheme: scheme,
		Log:    ctrl.Log,
		// readDefaultLabelsFromFile calls os.Exit(1) if this path doesn't
		// resolve, so point it at the real default labels shipped with the
		// manifests rather than a placeholder.
		DefaultNamespaceLabelsPath: filepath.Join("..", "manifests", "kustomize", "base", "manager", "namespace-labels.yaml"),
	}
}

var _ = Describe("Profile Controller - Namespace Adoption", func() {

	ctx := context.Background()

	bareNamespace := func(name string, annotations, labels map[string]string) *corev1.Namespace {
		return &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Annotations: annotations,
				Labels:      labels,
			},
		}
	}

	profileFor := func(name, owner string) *profilev1.Profile {
		return &profilev1.Profile{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec: profilev1.ProfileSpec{
				Owner: rbacv1.Subject{Kind: "User", Name: owner},
			},
		}
	}

	reconcile := func(r *ProfileReconciler, name string) error {
		_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: name}})
		return err
	}

	// ---------------------------------------------------------------
	// Test 1: Namespace with NO owner annotation — should be adopted
	// ---------------------------------------------------------------
	Context("When a namespace exists with no owner annotation", func() {
		const nsName = "adopt-no-owner"
		const ownerName = "alice@example.com"

		It("should adopt the namespace, set the owner annotation, apply default labels, and enable istio injection", func() {
			ns := bareNamespace(nsName, nil, nil)
			profile := profileFor(nsName, ownerName)
			r := newFakeProfileReconciler(ns, profile)

			Expect(reconcile(r, nsName)).To(Succeed())

			var got corev1.Namespace
			Expect(r.Get(ctx, types.NamespacedName{Name: nsName}, &got)).To(Succeed())
			Expect(got.Annotations["owner"]).To(Equal(ownerName))
			Expect(got.Labels["istio-injection"]).To(Equal("enabled"))
			Expect(got.Labels).To(HaveKey("app.kubernetes.io/part-of"))

			// The adopted namespace should now be owned by the Profile, so
			// deleting the Profile will garbage-collect the namespace too.
			Expect(got.OwnerReferences).To(HaveLen(1))
			Expect(got.OwnerReferences[0].Kind).To(Equal("Profile"))
			Expect(got.OwnerReferences[0].Name).To(Equal(nsName))
			Expect(got.OwnerReferences[0].Controller).NotTo(BeNil())
			Expect(*got.OwnerReferences[0].Controller).To(BeTrue())

			var gotProfile profilev1.Profile
			Expect(r.Get(ctx, types.NamespacedName{Name: nsName}, &gotProfile)).To(Succeed())
			for _, c := range gotProfile.Status.Conditions {
				Expect(c.Type).NotTo(Equal(profilev1.ProfileFailed))
			}
		})
	})

	// ---------------------------------------------------------------
	// Test 2: Namespace already owned by the SAME user — update labels
	// ---------------------------------------------------------------
	Context("When a namespace exists already owned by the same user", func() {
		const nsName = "same-owner-ns"
		const ownerName = "bob@example.com"

		It("should update labels without changing the owner annotation", func() {
			ns := bareNamespace(nsName, map[string]string{"owner": ownerName}, nil)
			profile := profileFor(nsName, ownerName)
			r := newFakeProfileReconciler(ns, profile)

			Expect(reconcile(r, nsName)).To(Succeed())

			var got corev1.Namespace
			Expect(r.Get(ctx, types.NamespacedName{Name: nsName}, &got)).To(Succeed())
			Expect(got.Labels).To(HaveKey("app.kubernetes.io/part-of"))
			Expect(got.Annotations["owner"]).To(Equal(ownerName))

			// SetControllerReference is only called on first adoption (when
			// the namespace had no owner annotation yet). A namespace that
			// was already correctly owned before this fix existed should
			// not retroactively gain an ownerReference.
			Expect(got.OwnerReferences).To(BeEmpty())
		})
	})

	// ---------------------------------------------------------------
	// Test 3: Namespace owned by a DIFFERENT user — must be rejected
	// ---------------------------------------------------------------
	Context("When a namespace exists owned by a different user", func() {
		const nsName = "different-owner-ns"
		const existingOwner = "carol@example.com"
		const newOwner = "dave@example.com"

		It("should NOT overwrite the existing owner annotation, and should fail the Profile", func() {
			ns := bareNamespace(nsName, map[string]string{"owner": existingOwner}, nil)
			profile := profileFor(nsName, newOwner)
			r := newFakeProfileReconciler(ns, profile)

			// Reconcile itself returns nil here — it records a Failed
			// condition on the Profile rather than returning an error,
			// since this is an expected/handled outcome, not a transient
			// failure that should be requeued.
			Expect(reconcile(r, nsName)).To(Succeed())

			var got corev1.Namespace
			Expect(r.Get(ctx, types.NamespacedName{Name: nsName}, &got)).To(Succeed())
			Expect(got.Annotations["owner"]).To(Equal(existingOwner))

			var gotProfile profilev1.Profile
			Expect(r.Get(ctx, types.NamespacedName{Name: nsName}, &gotProfile)).To(Succeed())
			Expect(gotProfile.Status.Conditions).NotTo(BeEmpty())
			lastCondition := gotProfile.Status.Conditions[len(gotProfile.Status.Conditions)-1]
			Expect(lastCondition.Type).To(Equal(profilev1.ProfileFailed))
		})
	})

	// ---------------------------------------------------------------
	// Test 4: Namespace with existing Istio label — must NOT overwrite
	// ---------------------------------------------------------------
	Context("When a namespace already has a custom istio-injection label", func() {
		const nsName = "custom-istio-ns"
		const ownerName = "eve@example.com"

		It("should preserve the existing istio-injection label value", func() {
			ns := bareNamespace(nsName, nil, map[string]string{"istio-injection": "disabled"})
			profile := profileFor(nsName, ownerName)
			r := newFakeProfileReconciler(ns, profile)

			Expect(reconcile(r, nsName)).To(Succeed())

			var got corev1.Namespace
			Expect(r.Get(ctx, types.NamespacedName{Name: nsName}, &got)).To(Succeed())
			Expect(got.Labels["istio-injection"]).To(Equal("disabled"))
			Expect(got.Annotations["owner"]).To(Equal(ownerName))
		})
	})
})
