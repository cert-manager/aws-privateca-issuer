package helm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRBAC(t *testing.T) {
	helper := setupTest(t)
	defer helper.cleanup()

	tests := []struct {
		name     string
		values   map[string]interface{}
		validate func(t *testing.T, h *testHelper, deploymentName string)
	}{
		{
			name: "rbac enabled creates ClusterRole and ClusterRoleBinding",
			values: map[string]interface{}{
				"rbac": map[string]interface{}{
					"create": true,
				},
				"serviceAccount": map[string]interface{}{
					"create": true,
				},
			},
			validate: func(t *testing.T, h *testHelper, deploymentName string) {
				// Verify ClusterRole exists
				clusterRole, err := h.clientset.RbacV1().ClusterRoles().Get(context.TODO(), deploymentName, metav1.GetOptions{})
				require.NoError(t, err)
				assert.NotEmpty(t, clusterRole.Rules)

				// Verify ClusterRoleBinding exists
				clusterRoleBinding, err := h.clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), deploymentName, metav1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, deploymentName, clusterRoleBinding.RoleRef.Name)
				assert.Len(t, clusterRoleBinding.Subjects, 1)
				assert.Equal(t, deploymentName, clusterRoleBinding.Subjects[0].Name)
			},
		},
		{
			name: "rbac disabled does not create ClusterRole and ClusterRoleBinding",
			values: map[string]interface{}{
				"rbac": map[string]interface{}{
					"create": false,
				},
			},
			validate: func(t *testing.T, h *testHelper, deploymentName string) {
				// Verify ClusterRole does not exist
				_, err := h.clientset.RbacV1().ClusterRoles().Get(context.TODO(), deploymentName, metav1.GetOptions{})
				assert.Error(t, err, "ClusterRole should not exist when RBAC is disabled")

				// Verify ClusterRoleBinding does not exist
				_, err = h.clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), deploymentName, metav1.GetOptions{})
				assert.Error(t, err, "ClusterRoleBinding should not exist when RBAC is disabled")
			},
		},
		{
			name: "serviceAccount created when enabled",
			values: map[string]interface{}{
				"serviceAccount": map[string]interface{}{
					"create": true,
					"name":   "custom-sa",
				},
			},
			validate: func(t *testing.T, h *testHelper, deploymentName string) {
				// Verify ServiceAccount exists with custom name
				sa, err := h.clientset.CoreV1().ServiceAccounts(h.namespace).Get(context.TODO(), "custom-sa", metav1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, "custom-sa", sa.Name)
			},
		},
		{
			name: "approver role creates additional ClusterRole and ClusterRoleBinding",
			values: map[string]interface{}{
				"approverRole": map[string]interface{}{
					"enabled":            true,
					"serviceAccountName": "cert-manager",
					"namespace":          "cert-manager",
				},
			},
			validate: func(t *testing.T, h *testHelper, deploymentName string) {
				// Verify approver ClusterRole exists
				approverRoleName := "cert-manager-controller-approve:awspca-cert-manager-io"
				clusterRole, err := h.clientset.RbacV1().ClusterRoles().Get(context.TODO(), approverRoleName, metav1.GetOptions{})
				require.NoError(t, err)
				assert.NotEmpty(t, clusterRole.Rules)

				// Verify approver ClusterRoleBinding exists
				clusterRoleBinding, err := h.clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), approverRoleName, metav1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, approverRoleName, clusterRoleBinding.RoleRef.Name)
				assert.Len(t, clusterRoleBinding.Subjects, 1)
				assert.Equal(t, "cert-manager", clusterRoleBinding.Subjects[0].Name)
				assert.Equal(t, "cert-manager", clusterRoleBinding.Subjects[0].Namespace)
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
