package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	profilev1 "github.com/kubeflow/dashboard/components/profile-controller/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
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
		namespaceLabelSuite{
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
		namespaceLabelSuite{
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
		namespaceLabelSuite{
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

func newFakeLimitRangeReconciler(t *testing.T, objs ...client.Object) *ProfileReconciler {
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, profilev1.AddToScheme(scheme))
	return &ProfileReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build(),
		Scheme: scheme,
		Log:    ctrl.Log,
	}
}

func TestReconcileLimitRange(t *testing.T) {
	ctx := context.Background()
	key := types.NamespacedName{Name: KFLIMITRANGE, Namespace: "team-a"}
	newProfile := func(spec *corev1.LimitRangeSpec) *profilev1.Profile {
		return &profilev1.Profile{
			ObjectMeta: metav1.ObjectMeta{Name: "team-a", UID: "team-a-uid"},
			Spec:       profilev1.ProfileSpec{LimitRangeSpec: spec},
		}
	}
	containerSpec := func(defaults corev1.ResourceList) *corev1.LimitRangeSpec {
		return &corev1.LimitRangeSpec{Limits: []corev1.LimitRangeItem{{
			Type:    corev1.LimitTypeContainer,
			Default: defaults,
		}}}
	}
	spec := containerSpec(corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("500m"),
		corev1.ResourceMemory: resource.MustParse("512Mi"),
	})
	unowned := func() *corev1.LimitRange {
		return &corev1.LimitRange{
			ObjectMeta: metav1.ObjectMeta{Name: KFLIMITRANGE, Namespace: "team-a"},
			Spec: *containerSpec(corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("2"),
				corev1.ResourceMemory: resource.MustParse("2Gi"),
			}),
		}
	}

	t.Run("unset creates nothing", func(t *testing.T) {
		r := newFakeLimitRangeReconciler(t)
		require.NoError(t, r.reconcileLimitRange(ctx, newProfile(nil)))
		err := r.Get(ctx, key, &corev1.LimitRange{})
		assert.True(t, apierrors.IsNotFound(err))
	})

	t.Run("empty limits are treated as unset", func(t *testing.T) {
		r := newFakeLimitRangeReconciler(t)
		require.NoError(t, r.reconcileLimitRange(ctx, newProfile(&corev1.LimitRangeSpec{Limits: []corev1.LimitRangeItem{}})))
		err := r.Get(ctx, key, &corev1.LimitRange{})
		assert.True(t, apierrors.IsNotFound(err))
	})

	t.Run("set creates an owned LimitRange", func(t *testing.T) {
		r := newFakeLimitRangeReconciler(t)
		profile := newProfile(spec)
		require.NoError(t, r.reconcileLimitRange(ctx, profile))
		found := &corev1.LimitRange{}
		require.NoError(t, r.Get(ctx, key, found))
		assert.True(t, apiequality.Semantic.DeepEqual(*spec, found.Spec))
		assert.True(t, metav1.IsControlledBy(found, profile))
	})

	t.Run("changed spec updates the LimitRange", func(t *testing.T) {
		r := newFakeLimitRangeReconciler(t)
		require.NoError(t, r.reconcileLimitRange(ctx, newProfile(spec)))
		changed := containerSpec(corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("250m")})
		require.NoError(t, r.reconcileLimitRange(ctx, newProfile(changed)))
		found := &corev1.LimitRange{}
		require.NoError(t, r.Get(ctx, key, found))
		assert.True(t, apiequality.Semantic.DeepEqual(*changed, found.Spec), "got %v", found.Spec)
	})

	t.Run("equivalent quantities do not update", func(t *testing.T) {
		r := newFakeLimitRangeReconciler(t)
		require.NoError(t, r.reconcileLimitRange(ctx, newProfile(containerSpec(corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")}))))
		before := &corev1.LimitRange{}
		require.NoError(t, r.Get(ctx, key, before))
		require.NoError(t, r.reconcileLimitRange(ctx, newProfile(containerSpec(corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1000m")}))))
		after := &corev1.LimitRange{}
		require.NoError(t, r.Get(ctx, key, after))
		assert.Equal(t, before.ResourceVersion, after.ResourceVersion)
	})

	t.Run("unset deletes an owned LimitRange", func(t *testing.T) {
		r := newFakeLimitRangeReconciler(t)
		require.NoError(t, r.reconcileLimitRange(ctx, newProfile(spec)))
		require.NoError(t, r.reconcileLimitRange(ctx, newProfile(nil)))
		err := r.Get(ctx, key, &corev1.LimitRange{})
		assert.True(t, apierrors.IsNotFound(err))
	})

	t.Run("unset leaves an unowned LimitRange alone", func(t *testing.T) {
		existing := unowned()
		r := newFakeLimitRangeReconciler(t, existing)
		require.NoError(t, r.reconcileLimitRange(ctx, newProfile(nil)))
		found := &corev1.LimitRange{}
		require.NoError(t, r.Get(ctx, key, found))
		assert.True(t, apiequality.Semantic.DeepEqual(existing.Spec, found.Spec))
		assert.Empty(t, found.OwnerReferences)
	})

	t.Run("set adopts and overwrites an unowned LimitRange", func(t *testing.T) {
		r := newFakeLimitRangeReconciler(t, unowned())
		profile := newProfile(spec)
		require.NoError(t, r.reconcileLimitRange(ctx, profile))
		found := &corev1.LimitRange{}
		require.NoError(t, r.Get(ctx, key, found))
		assert.True(t, apiequality.Semantic.DeepEqual(*spec, found.Spec))
		assert.True(t, metav1.IsControlledBy(found, profile))
	})
}
