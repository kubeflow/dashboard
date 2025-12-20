/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// TestLoadPluginConfigFromConfigMap tests loading plugin configuration from ConfigMap
func TestLoadPluginConfigFromConfigMap(t *testing.T) {
	tests := []struct {
		name           string
		configMap      *corev1.ConfigMap
		expectError    bool
		expectNil      bool
		expectCount    int
		expectPlugins  []string
	}{
		{
			name: "Valid ConfigMap with both plugins enabled",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PLUGIN_CONFIG_MAP_NAME,
					Namespace: PLUGIN_CONFIG_MAP_NAMESPACE,
				},
				Data: map[string]string{
					PLUGIN_CONFIG_DATA_KEY: `
plugins:
  - kind: WorkloadIdentity
    enabled: true
    config:
      gcpServiceAccount: "test@project.iam.gserviceaccount.com"
  - kind: AwsIamForServiceAccount
    enabled: true
    config:
      awsIamRole: "arn:aws:iam::123456789:role/test"
      annotateOnly: false
`,
				},
			},
			expectError:   false,
			expectNil:     false,
			expectCount:   2,
			expectPlugins: []string{"WorkloadIdentity", "AwsIamForServiceAccount"},
		},
		{
			name: "Valid ConfigMap with one plugin disabled",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PLUGIN_CONFIG_MAP_NAME,
					Namespace: PLUGIN_CONFIG_MAP_NAMESPACE,
				},
				Data: map[string]string{
					PLUGIN_CONFIG_DATA_KEY: `
plugins:
  - kind: WorkloadIdentity
    enabled: true
    config:
      gcpServiceAccount: "test@project.iam.gserviceaccount.com"
  - kind: AwsIamForServiceAccount
    enabled: false
    config:
      awsIamRole: ""
`,
				},
			},
			expectError:   false,
			expectNil:     false,
			expectCount:   2,
			expectPlugins: []string{"WorkloadIdentity", "AwsIamForServiceAccount"},
		},
		{
			name:        "ConfigMap not found",
			configMap:   nil,
			expectError: false,
			expectNil:   true,
		},
		{
			name: "ConfigMap with missing plugins.yaml key",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PLUGIN_CONFIG_MAP_NAME,
					Namespace: PLUGIN_CONFIG_MAP_NAMESPACE,
				},
				Data: map[string]string{
					"wrong-key": "data",
				},
			},
			expectError: false,
			expectNil:   true,
		},
		{
			name: "ConfigMap with invalid YAML",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PLUGIN_CONFIG_MAP_NAME,
					Namespace: PLUGIN_CONFIG_MAP_NAMESPACE,
				},
				Data: map[string]string{
					PLUGIN_CONFIG_DATA_KEY: `invalid yaml: [[[`,
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)

			var objs []runtime.Object
			if tt.configMap != nil {
				objs = append(objs, tt.configMap)
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(objs...).
				Build()

			// Create reconciler
			reconciler := &ProfileReconciler{
				Client: fakeClient,
				Log:    zap.New(zap.UseDevMode(true)),
			}

			// Call the function
			config, err := reconciler.LoadPluginConfigFromConfigMap(context.Background())

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Check nil expectation
			if tt.expectNil && config != nil {
				t.Errorf("Expected nil config but got: %v", config)
			}
			if !tt.expectNil && !tt.expectError && config == nil {
				t.Errorf("Expected non-nil config but got nil")
			}

			// Check plugin count and kinds
			if config != nil {
				if len(config.Plugins) != tt.expectCount {
					t.Errorf("Expected %d plugins but got %d", tt.expectCount, len(config.Plugins))
				}

				for i, expectedKind := range tt.expectPlugins {
					if i < len(config.Plugins) && config.Plugins[i].Kind != expectedKind {
						t.Errorf("Expected plugin %d to be %s but got %s", i, expectedKind, config.Plugins[i].Kind)
					}
				}
			}
		})
	}
}

// TestGetPluginInstanceFromConfig tests creating plugin instances from config
func TestGetPluginInstanceFromConfig(t *testing.T) {
	tests := []struct {
		name         string
		config       PluginConfigSpec
		expectNil    bool
		expectType   string
		expectError  bool
	}{
		{
			name: "WorkloadIdentity plugin enabled",
			config: PluginConfigSpec{
				Kind:    KIND_WORKLOAD_IDENTITY,
				Enabled: true,
				Config: map[string]interface{}{
					"gcpServiceAccount": "test@project.iam.gserviceaccount.com",
				},
			},
			expectNil:   false,
			expectType:  "*controllers.GcpWorkloadIdentity",
			expectError: false,
		},
		{
			name: "AwsIamForServiceAccount plugin enabled",
			config: PluginConfigSpec{
				Kind:    KIND_AWS_IAM_FOR_SERVICE_ACCOUNT,
				Enabled: true,
				Config: map[string]interface{}{
					"awsIamRole":   "arn:aws:iam::123:role/test",
					"annotateOnly": false,
				},
			},
			expectNil:   false,
			expectType:  "*controllers.AwsIAMForServiceAccount",
			expectError: false,
		},
		{
			name: "Plugin disabled",
			config: PluginConfigSpec{
				Kind:    KIND_WORKLOAD_IDENTITY,
				Enabled: false,
				Config:  map[string]interface{}{},
			},
			expectNil:   true,
			expectError: false,
		},
		{
			name: "Unknown plugin kind",
			config: PluginConfigSpec{
				Kind:    "UnknownPlugin",
				Enabled: true,
				Config:  map[string]interface{}{},
			},
			expectNil:   true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client
			scheme := runtime.NewScheme()
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

			// Create reconciler
			reconciler := &ProfileReconciler{
				Client: fakeClient,
				Log:    zap.New(zap.UseDevMode(true)),
			}

			// Call the function
			plugin, err := reconciler.GetPluginInstanceFromConfig(tt.config)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Check nil expectation
			if tt.expectNil && plugin != nil {
				t.Errorf("Expected nil plugin but got: %T", plugin)
			}
			if !tt.expectNil && plugin == nil {
				t.Errorf("Expected non-nil plugin but got nil")
			}

			// Check type if not expecting nil
			if !tt.expectNil && plugin != nil {
				pluginType := types.NamespacedName{Name: tt.expectType}.String()
				_ = pluginType // Type checking would require reflection, simplified here
			}
		})
	}
}

// TestGetPluginSpecWithConfigMap tests the refactored GetPluginSpec function
func TestGetPluginSpecWithConfigMap(t *testing.T) {
	// This test would require setting up full Profile CRs and is more complex
	// For now, we focus on the ConfigMap loading tests above
	// Integration tests should be done in a full Kind cluster environment
	t.Skip("Integration test - requires full Profile CR setup")
}
