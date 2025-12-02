package helm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAutoscaling(t *testing.T) {
	helper := setupTest(t)
	defer helper.cleanup()

	tests := []struct {
		name     string
		values   map[string]interface{}
		validate func(t *testing.T, h *testHelper, deploymentName string)
	}{
		{
			name: "autoscaling enabled creates HPA and removes replica count",
			values: map[string]interface{}{
				"autoscaling": map[string]interface{}{
					"enabled":                        true,
					"minReplicas":                    2,
					"maxReplicas":                    10,
					"targetCPUUtilizationPercentage": 70,
				},
			},
			validate: func(t *testing.T, h *testHelper, deploymentName string) {
				// Verify HPA exists and is configured correctly
				hpa, err := h.clientset.AutoscalingV2().HorizontalPodAutoscalers(h.namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, int32(2), *hpa.Spec.MinReplicas)
				assert.Equal(t, int32(10), hpa.Spec.MaxReplicas)

				// Verify Deployment doesn't have replica count set when autoscaling is enabled
				deployment, err := h.clientset.AppsV1().Deployments(h.namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
				require.NoError(t, err)
				assert.Nil(t, deployment.Spec.Replicas, "Deployment should not have replicas set when autoscaling is enabled")
			},
		},
		{
			name: "autoscaling disabled uses replica count from values",
			values: map[string]interface{}{
				"autoscaling": map[string]interface{}{
					"enabled": false,
				},
				"replicaCount": 3,
			},
			validate: func(t *testing.T, h *testHelper, deploymentName string) {
				// Verify HPA does not exist
				_, err := h.clientset.AutoscalingV2().HorizontalPodAutoscalers(h.namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
				assert.Error(t, err, "HPA should not exist when autoscaling is disabled")

				// Verify Deployment has correct replica count
				deployment, err := h.clientset.AppsV1().Deployments(h.namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
				require.NoError(t, err)
				require.NotNil(t, deployment.Spec.Replicas)
				assert.Equal(t, int32(3), *deployment.Spec.Replicas)
			},
		},
		{
			name: "autoscaling with memory target",
			values: map[string]interface{}{
				"autoscaling": map[string]interface{}{
					"enabled":                           true,
					"minReplicas":                       1,
					"maxReplicas":                       5,
					"targetMemoryUtilizationPercentage": 80,
				},
			},
			validate: func(t *testing.T, h *testHelper, deploymentName string) {
				hpa, err := h.clientset.AutoscalingV2().HorizontalPodAutoscalers(h.namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
				require.NoError(t, err)

				// Check for memory metric
				found := false
				for _, metric := range hpa.Spec.Metrics {
					if metric.Resource != nil && metric.Resource.Name == "memory" {
						found = true
						assert.Equal(t, int32(80), *metric.Resource.Target.AverageUtilization)
						break
					}
				}
				assert.True(t, found, "Memory metric should be configured")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			release := helper.installChart(tt.values)
			defer helper.uninstallChart(release.Name)

			deploymentName := release.Name + "-aws-privateca-issuer"
			helper.waitForDeployment(deploymentName)
			tt.validate(t, helper, deploymentName)

			t.Logf("Test %s completed successfully with release %s", tt.name, release.Name)
		})
	}
}
