package helm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeployment(t *testing.T) {
	helper := setupTest(t)
	defer helper.cleanup()

	tests := []struct {
		name     string
		values   map[string]interface{}
		validate func(t *testing.T, h *testHelper, deploymentName string)
	}{
		{
			name: "disableApprovedCheck adds command line flag",
			values: map[string]interface{}{
				"disableApprovedCheck": true,
			},
			validate: func(t *testing.T, h *testHelper, deploymentName string) {
				deployment, err := h.clientset.AppsV1().Deployments(h.namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
				require.NoError(t, err)

				container := deployment.Spec.Template.Spec.Containers[0]
				assert.Contains(t, container.Args, "-disable-approved-check")
			},
		},
		{
			name: "disableClientSideRateLimiting adds command line flag",
			values: map[string]interface{}{
				"disableClientSideRateLimiting": true,
			},
			validate: func(t *testing.T, h *testHelper, deploymentName string) {
				deployment, err := h.clientset.AppsV1().Deployments(h.namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
				require.NoError(t, err)

				container := deployment.Spec.Template.Spec.Containers[0]
				assert.Contains(t, container.Args, "-disable-client-side-rate-limiting")
			},
		},
		{
			name: "priorityClassName sets priority class on pod",
			values: map[string]interface{}{
				"priorityClassName": "high-priority",
			},
			validate: func(t *testing.T, h *testHelper, deploymentName string) {
				deployment, err := h.clientset.AppsV1().Deployments(h.namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
				require.NoError(t, err)

				assert.Equal(t, "high-priority", deployment.Spec.Template.Spec.PriorityClassName)
			},
		},
		{
			name: "env variables are added to container",
			values: map[string]interface{}{
				"env": map[string]interface{}{
					"AWS_REGION":    "us-west-2",
					"LOG_LEVEL":     "debug",
					"CUSTOM_CONFIG": "test-value",
				},
			},
			validate: func(t *testing.T, h *testHelper, deploymentName string) {
				deployment, err := h.clientset.AppsV1().Deployments(h.namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
				require.NoError(t, err)

				container := deployment.Spec.Template.Spec.Containers[0]
				envMap := make(map[string]string)
				for _, env := range container.Env {
					envMap[env.Name] = env.Value
				}

				assert.Equal(t, "us-west-2", envMap["AWS_REGION"])
				assert.Equal(t, "debug", envMap["LOG_LEVEL"])
				assert.Equal(t, "test-value", envMap["CUSTOM_CONFIG"])
			},
		},
		{
			name: "volumeMounts and volumes are configured",
			values: map[string]interface{}{
				"volumes": []interface{}{
					map[string]interface{}{
						"name": "config-volume",
						"configMap": map[string]interface{}{
							"name": "app-config",
						},
					},
				},
				"volumeMounts": []interface{}{
					map[string]interface{}{
						"name":      "config-volume",
						"mountPath": "/etc/config",
						"readOnly":  true,
					},
				},
			},
			validate: func(t *testing.T, h *testHelper, deploymentName string) {
				deployment, err := h.clientset.AppsV1().Deployments(h.namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
				require.NoError(t, err)

				// Check volumes
				assert.Len(t, deployment.Spec.Template.Spec.Volumes, 1)
				assert.Equal(t, "config-volume", deployment.Spec.Template.Spec.Volumes[0].Name)

				// Check volume mounts
				container := deployment.Spec.Template.Spec.Containers[0]
				assert.Len(t, container.VolumeMounts, 1)
				assert.Equal(t, "config-volume", container.VolumeMounts[0].Name)
				assert.Equal(t, "/etc/config", container.VolumeMounts[0].MountPath)
				assert.True(t, container.VolumeMounts[0].ReadOnly)
			},
		},
		{
			name: "extraContainers adds sidecar containers",
			values: map[string]interface{}{
				"extraContainers": []interface{}{
					map[string]interface{}{
						"name":  "sidecar",
						"image": "nginx:latest",
						"ports": []interface{}{
							map[string]interface{}{
								"containerPort": 80,
							},
						},
					},
				},
			},
			validate: func(t *testing.T, h *testHelper, deploymentName string) {
				deployment, err := h.clientset.AppsV1().Deployments(h.namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
				require.NoError(t, err)

				assert.Len(t, deployment.Spec.Template.Spec.Containers, 2)

				// Find the sidecar container
				var sidecar *corev1.Container
				for _, container := range deployment.Spec.Template.Spec.Containers {
					if container.Name == "sidecar" {
						sidecar = &container
						break
					}
				}

				require.NotNil(t, sidecar, "Sidecar container should exist")
				assert.Equal(t, "nginx:latest", sidecar.Image)
				assert.Len(t, sidecar.Ports, 1)
				assert.Equal(t, int32(80), sidecar.Ports[0].ContainerPort)
			},
		},
		{
			name: "podLabels adds labels to pod template",
			values: map[string]interface{}{
				"podLabels": map[string]interface{}{
					"custom-label":         "custom-value",
					"environment":          "test",
					"monitoring.io/scrape": "true",
				},
			},
			validate: func(t *testing.T, h *testHelper, deploymentName string) {
				deployment, err := h.clientset.AppsV1().Deployments(h.namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
				require.NoError(t, err)

				labels := deployment.Spec.Template.Labels
				assert.Equal(t, "custom-value", labels["custom-label"])
				assert.Equal(t, "test", labels["environment"])
				assert.Equal(t, "true", labels["monitoring.io/scrape"])
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
