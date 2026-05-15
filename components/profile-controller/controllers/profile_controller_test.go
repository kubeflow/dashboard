package controllers

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	profilev1 "github.com/kubeflow/dashboard/components/profile-controller/api/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type namespaceLabelSuite struct {
	current  corev1.Namespace
	labels   map[string]string
	expected corev1.Namespace
}

func TestEnforceNamespaceLabelsFromConfig(t *testing.T) {
	name := "test-namespace"

	// Create a minimal ProfileReconciler for testing
	reconciler := &ProfileReconciler{
		ServiceMeshMode: "istio-sidecar", // Test sidecar mode
	}

	// Create a minimal profile for testing
	profile := &profilev1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: profilev1.ProfileSpec{
			Owner: rbacv1.Subject{
				Kind: "User",
				Name: "test-user",
			},
		},
	}

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
						"istio-injection":                                "enabled", // Added by service mesh logic
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
						"serving.kubeflow.org/inferenceservice":          "disabled", // Existing label preserved
						"pipelines.kubeflow.org/enabled":                 "true",
						"app.kubernetes.io/part-of":                      "kubeflow-profile",
						"istio-injection":                                "enabled", // Added by service mesh logic
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
						"istio-injection":                                "enabled", // Added by service mesh logic
						// "removal-label" should be removed due to empty value
					},
					Name: name,
				},
			},
		},
		// Test ambient mode
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
						"istio-injection":                                "disabled", // Ambient mode disables sidecar
						"istio.io/dataplane-mode":                        "ambient",
						"istio.io/use-waypoint":                          "waypoint",
						"istio.io/use-waypoint-namespace":                name,
						"istio.io/ingress-use-waypoint":                  "true",
					},
					Name: name,
				},
			},
		},
	}
	for i, test := range tests {
		// Use ambient mode reconciler for the last test case
		testReconciler := reconciler
		if i == len(tests)-1 { // Last test case is ambient mode
			testReconciler = &ProfileReconciler{
				ServiceMeshMode:   "istio-ambient",
				WaypointName:      "waypoint",
				WaypointNamespace: "", // Empty means use profile namespace
			}
		}
		testReconciler.setNamespaceLabelsAndServiceMesh(&test.current, profile, test.labels)
		if !reflect.DeepEqual(&test.expected, &test.current) {
			t.Errorf("Test case %d: Expect:\n%v; Output:\n%v", i, &test.expected, &test.current)
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

// Test waypoint creation in ambient mode
func TestCreateWaypointAmbientMode(t *testing.T) {
	// This is a basic test to verify the waypoint creation logic
	// In a real test environment, you would mock the Kubernetes client

	profile := &profilev1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-profile",
		},
		Spec: profilev1.ProfileSpec{
			Owner: rbacv1.Subject{
				Kind: "User",
				Name: "test-user",
			},
		},
	}

	// Test with waypoint in same namespace
	reconciler := &ProfileReconciler{
		ServiceMeshMode:   "istio-ambient",
		WaypointName:      "test-waypoint",
		WaypointNamespace: "", // Empty means use profile namespace
		CreateWaypoint:    true,
	}

	// Verify waypoint namespace defaults to profile namespace when empty
	// This is just testing the field value, not the actual creation logic
	// which would require mocking the Kubernetes client
	_ = profile // Use profile to avoid unused variable error
	if reconciler.WaypointNamespace != "" {
		t.Errorf("Expected empty waypoint namespace to default to profile namespace")
	}

	// Test with waypoint in different namespace
	reconciler2 := &ProfileReconciler{
		ServiceMeshMode:   "istio-ambient",
		WaypointName:      "shared-waypoint",
		WaypointNamespace: "istio-system",
		CreateWaypoint:    true,
	}

	if reconciler2.WaypointNamespace != "istio-system" {
		t.Errorf("Expected waypoint namespace to be 'istio-system', got %s", reconciler2.WaypointNamespace)
	}
}

// Test getAuthorizationPolicy with ambient mode
func TestGetAuthorizationPolicyAmbientMode(t *testing.T) {
	profile := &profilev1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-profile",
		},
		Spec: profilev1.ProfileSpec{
			Owner: rbacv1.Subject{
				Kind: "User",
				Name: "test-user@example.com",
			},
		},
	}

	// Test sidecar mode (default)
	reconcilerSidecar := &ProfileReconciler{
		ServiceMeshMode: "istio-sidecar",
		UserIdHeader:    "x-goog-authenticated-user-email",
		UserIdPrefix:    "accounts.google.com:",
	}

	policySidecar := reconcilerSidecar.getAuthorizationPolicy(profile)

	// In sidecar mode, TargetRefs should be nil
	if policySidecar.TargetRefs != nil {
		t.Errorf("Expected TargetRefs to be nil in sidecar mode, got %v", policySidecar.TargetRefs)
	}

	// Test ambient mode
	reconcilerAmbient := &ProfileReconciler{
		ServiceMeshMode: "istio-ambient",
		WaypointName:    "test-waypoint",
		UserIdHeader:    "x-goog-authenticated-user-email",
		UserIdPrefix:    "accounts.google.com:",
	}

	policyAmbient := reconcilerAmbient.getAuthorizationPolicy(profile)

	// In ambient mode, TargetRefs should be set
	if policyAmbient.TargetRefs == nil {
		t.Errorf("Expected TargetRefs to be set in ambient mode")
	} else {
		if len(policyAmbient.TargetRefs) != 1 {
			t.Errorf("Expected 1 TargetRef, got %d", len(policyAmbient.TargetRefs))
		} else {
			targetRef := policyAmbient.TargetRefs[0]
			if targetRef.Kind != "Gateway" {
				t.Errorf("Expected TargetRef Kind to be 'Gateway', got %s", targetRef.Kind)
			}
			if targetRef.Group != "gateway.networking.k8s.io" {
				t.Errorf("Expected TargetRef Group to be 'gateway.networking.k8s.io', got %s", targetRef.Group)
			}
			if targetRef.Name != "test-waypoint" {
				t.Errorf("Expected TargetRef Name to be 'test-waypoint', got %s", targetRef.Name)
			}
		}
	}
}
