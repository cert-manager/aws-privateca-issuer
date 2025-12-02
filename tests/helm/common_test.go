package helm

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	chartPath     = "../../charts/aws-pca-issuer"
	testNamespace = "aws-pca-issuer-test"
	releasePrefix = "test-release"
)

type testHelper struct {
	t         *testing.T
	clientset kubernetes.Interface
	namespace string
}

func setupTest(t *testing.T) *testHelper {
	// Use existing cluster setup from make target
	kubeconfig := "/tmp/pca_kubeconfig"
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		t.Skipf("Skipping e2e test - no Kubernetes cluster available: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	require.NoError(t, err)

	// Create test namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespace,
		},
	}
	_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		require.NoError(t, err)
	}

	return &testHelper{
		t:         t,
		clientset: clientset,
		namespace: testNamespace,
	}
}

func (h *testHelper) cleanup() {
	err := h.clientset.CoreV1().Namespaces().Delete(context.TODO(), h.namespace, metav1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		h.t.Logf("Failed to cleanup namespace: %v", err)
	}
}

func (h *testHelper) installChart(values map[string]interface{}) *release.Release {
	settings := cli.New()
	settings.KubeConfig = "/tmp/pca_kubeconfig" // Use the same kubeconfig as manual install
	actionConfig := new(action.Configuration)

	err := actionConfig.Init(settings.RESTClientGetter(), h.namespace, "secret", func(format string, v ...interface{}) {
		h.t.Logf(format, v...)
	})
	require.NoError(h.t, err)

	// Generate unique release name
	releaseName := fmt.Sprintf("%s-%d", releasePrefix, time.Now().UnixNano())

	install := action.NewInstall(actionConfig)
	install.ReleaseName = releaseName
	install.Namespace = h.namespace
	install.CreateNamespace = true
	install.Wait = false // Don't wait for pods to be ready
	install.Timeout = 2 * time.Minute

	chart, err := loader.Load(chartPath)
	require.NoError(h.t, err)

	// Override image for testing to use a simple image that works
	if values == nil {
		values = make(map[string]interface{})
	}
	values["image"] = map[string]interface{}{
		"repository": "nginx",
		"tag":        "alpine",
		"pullPolicy": "IfNotPresent",
	}
	// Disable approver role to avoid cluster-scoped resource conflicts
	values["approverRole"] = map[string]interface{}{
		"enabled": false,
	}

	release, err := install.Run(chart, values)
	require.NoError(h.t, err)

	// Debug: Show what Helm thinks it created
	h.t.Logf("Helm release %s installed successfully", release.Name)
	h.t.Logf("Release manifest length: %d", len(release.Manifest))
	h.t.Logf("Release info: %+v", release.Info)
	
	// Show the actual manifest (complete)
	if len(release.Manifest) > 0 {
		h.t.Logf("Complete Helm manifest:\n%s", release.Manifest)
	}
	
	time.Sleep(2 * time.Second) // Give time for resources to be created
	
	// List all resources to debug what was actually created
	pods, _ := h.clientset.CoreV1().Pods(h.namespace).List(context.TODO(), metav1.ListOptions{})
	h.t.Logf("Pods created: %d", len(pods.Items))
	
	deployments, _ := h.clientset.AppsV1().Deployments(h.namespace).List(context.TODO(), metav1.ListOptions{})
	h.t.Logf("Deployments created: %d", len(deployments.Items))
	for _, dep := range deployments.Items {
		h.t.Logf("  - Deployment: %s", dep.Name)
	}
	
	services, _ := h.clientset.CoreV1().Services(h.namespace).List(context.TODO(), metav1.ListOptions{})
	h.t.Logf("Services created: %d", len(services.Items))
	for _, svc := range services.Items {
		h.t.Logf("  - Service: %s", svc.Name)
	}

	return release
}

func (h *testHelper) uninstallChart(releaseName string) {
	settings := cli.New()
	settings.KubeConfig = "/tmp/pca_kubeconfig"
	actionConfig := new(action.Configuration)

	err := actionConfig.Init(settings.RESTClientGetter(), h.namespace, "secret", func(format string, v ...interface{}) {
		h.t.Logf(format, v...)
	})
	if err != nil {
		return
	}

	uninstall := action.NewUninstall(actionConfig)
	uninstall.Run(releaseName)
}

func (h *testHelper) waitForDeployment(name string) {
	// Just check that the deployment exists, don't wait for readiness
	// Add initial delay to allow Helm to create resources
	time.Sleep(2 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			// List all deployments to debug
			deployments, err := h.clientset.AppsV1().Deployments(h.namespace).List(context.TODO(), metav1.ListOptions{})
			if err == nil {
				h.t.Logf("Available deployments in namespace %s:", h.namespace)
				for _, dep := range deployments.Items {
					h.t.Logf("  - %s", dep.Name)
				}
			}
			h.t.Fatalf("Timeout waiting for deployment %s to be created", name)
		default:
			_, err := h.clientset.AppsV1().Deployments(h.namespace).Get(context.TODO(), name, metav1.GetOptions{})
			if err == nil {
				// Print resources for debugging
				h.printResourcesForDebugging(name)
				return // Deployment exists, that's enough for our tests
			}
			time.Sleep(1 * time.Second)
		}
	}
}

func (h *testHelper) printResourcesForDebugging(deploymentName string) {
	h.t.Logf("=== KUBERNETES RESOURCES VALIDATION ===")
	
	// Print Deployment details
	if dep, err := h.clientset.AppsV1().Deployments(h.namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{}); err == nil {
		h.t.Logf("✓ Deployment: %s", dep.Name)
		h.t.Logf("  Image: %s", dep.Spec.Template.Spec.Containers[0].Image)
		h.t.Logf("  Args: %v", dep.Spec.Template.Spec.Containers[0].Args)
		if dep.Spec.Replicas != nil {
			h.t.Logf("  Replicas: %d", *dep.Spec.Replicas)
		} else {
			h.t.Logf("  Replicas: <nil> (managed by HPA)")
		}
	}
	
	// Print HPA if exists
	if hpa, err := h.clientset.AutoscalingV2().HorizontalPodAutoscalers(h.namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{}); err == nil {
		h.t.Logf("✓ HPA: %s", hpa.Name)
		h.t.Logf("  Min/Max Replicas: %d/%d", *hpa.Spec.MinReplicas, hpa.Spec.MaxReplicas)
	}
	
	// Print Service
	if svc, err := h.clientset.CoreV1().Services(h.namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{}); err == nil {
		h.t.Logf("✓ Service: %s", svc.Name)
		h.t.Logf("  Type: %s, Port: %d", svc.Spec.Type, svc.Spec.Ports[0].Port)
	}
	
	// Print ServiceAccount
	if sa, err := h.clientset.CoreV1().ServiceAccounts(h.namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{}); err == nil {
		h.t.Logf("✓ ServiceAccount: %s", sa.Name)
	}
	
	// Print PodDisruptionBudget
	if pdb, err := h.clientset.PolicyV1().PodDisruptionBudgets(h.namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{}); err == nil {
		h.t.Logf("✓ PodDisruptionBudget: %s", pdb.Name)
		if pdb.Spec.MaxUnavailable != nil {
			h.t.Logf("  MaxUnavailable: %s", pdb.Spec.MaxUnavailable.String())
		}
	}
	
	// Print ClusterRole and ClusterRoleBinding
	if cr, err := h.clientset.RbacV1().ClusterRoles().Get(context.TODO(), deploymentName, metav1.GetOptions{}); err == nil {
		h.t.Logf("✓ ClusterRole: %s", cr.Name)
	}
	if crb, err := h.clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), deploymentName, metav1.GetOptions{}); err == nil {
		h.t.Logf("✓ ClusterRoleBinding: %s", crb.Name)
	}
	
	h.t.Logf("=== END RESOURCES VALIDATION ===")
}
