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
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

const (
	// Default ConfigMap name and namespace for plugin configuration
	PLUGIN_CONFIG_MAP_NAME      = "profile-controller-plugins-config"
	PLUGIN_CONFIG_MAP_NAMESPACE = "kubeflow"
	PLUGIN_CONFIG_DATA_KEY      = "plugins.yaml"
)

// PluginConfigSpec represents a single plugin's configuration from ConfigMap
// This is what gets parsed from the YAML in the ConfigMap
type PluginConfigSpec struct {
	// Kind is the plugin type (e.g., "WorkloadIdentity", "AwsIamForServiceAccount")
	Kind string `yaml:"kind" json:"kind"`

	// Enabled indicates if this plugin should be applied to profiles
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Config contains the plugin-specific configuration as a map
	// This will be marshaled into the specific plugin struct (GcpWorkloadIdentity, AwsIAMForServiceAccount, etc.)
	Config map[string]interface{} `yaml:"config" json:"config"`
}

// PluginsConfiguration represents the entire plugins configuration from ConfigMap
type PluginsConfiguration struct {
	// Plugins is a list of plugin configurations
	Plugins []PluginConfigSpec `yaml:"plugins" json:"plugins"`
}

// LoadPluginConfigFromConfigMap reads the plugin configuration from a Kubernetes ConfigMap
// It looks for a ConfigMap named "profile-controller-plugins-config" in the "kubeflow" namespace
// Returns PluginsConfiguration with all plugin configs, or nil if ConfigMap doesn't exist
func (r *ProfileReconciler) LoadPluginConfigFromConfigMap(ctx context.Context) (*PluginsConfiguration, error) {
	logger := r.Log.WithValues("configmap", PLUGIN_CONFIG_MAP_NAME, "namespace", PLUGIN_CONFIG_MAP_NAMESPACE)

	// If Client is not initialized (e.g., in unit tests), return nil to fallback to Profile CR
	if r.Client == nil {
		logger.Info("Client not initialized, skipping ConfigMap loading")
		return nil, nil
	}

	// Fetch the ConfigMap from Kubernetes
	configMap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      PLUGIN_CONFIG_MAP_NAME,
		Namespace: PLUGIN_CONFIG_MAP_NAMESPACE,
	}, configMap)

	if err != nil {
		// If ConfigMap doesn't exist, return nil (fallback to default behavior)
		if apierrors.IsNotFound(err) {
			logger.Info("Plugin ConfigMap not found, using default plugin configuration")
			return nil, nil
		}
		// For other errors, return the error
		logger.Error(err, "Failed to get plugin ConfigMap")
		return nil, err
	}

	// Extract the YAML data from the ConfigMap
	yamlData, ok := configMap.Data[PLUGIN_CONFIG_DATA_KEY]
	if !ok {
		logger.Info("ConfigMap found but missing plugins.yaml key, using default plugin configuration")
		return nil, nil
	}

	// Parse the YAML into our struct
	pluginConfig := &PluginsConfiguration{}
	err = yaml.Unmarshal([]byte(yamlData), pluginConfig)
	if err != nil {
		logger.Error(err, "Failed to unmarshal plugin configuration from ConfigMap")
		return nil, err
	}

	logger.Info("Successfully loaded plugin configuration from ConfigMap", "pluginCount", len(pluginConfig.Plugins))
	return pluginConfig, nil
}

// GetPluginInstanceFromConfig creates a Plugin instance from a PluginConfigSpec
// This function takes the generic config from ConfigMap and creates the specific plugin type
func (r *ProfileReconciler) GetPluginInstanceFromConfig(config PluginConfigSpec) (Plugin, error) {
	logger := r.Log.WithValues("pluginKind", config.Kind)

	// Skip disabled plugins
	if !config.Enabled {
		logger.Info("Plugin is disabled, skipping")
		return nil, nil
	}

	// Create the appropriate plugin instance based on Kind
	var pluginIns Plugin
	switch config.Kind {
	case KIND_WORKLOAD_IDENTITY:
		pluginIns = &GcpWorkloadIdentity{}
	case KIND_AWS_IAM_FOR_SERVICE_ACCOUNT:
		pluginIns = &AwsIAMForServiceAccount{}
	default:
		logger.Info("Unknown plugin kind, skipping", "kind", config.Kind)
		return nil, nil
	}

	// Marshal the config map to JSON, then unmarshal into the specific plugin struct
	// This converts map[string]interface{} -> JSON bytes -> specific struct type
	configBytes, err := json.Marshal(config.Config)
	if err != nil {
		logger.Error(err, "Failed to marshal plugin config")
		return nil, err
	}

	err = json.Unmarshal(configBytes, pluginIns)
	if err != nil {
		logger.Error(err, "Failed to unmarshal plugin config into specific type")
		return nil, err
	}

	logger.Info("Successfully created plugin instance from ConfigMap config")
	return pluginIns, nil
}
