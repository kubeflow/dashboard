package controllers

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	profilev1 "github.com/kubeflow/dashboard/components/profile-controller/api/v1"
)

var _ = Describe("Profile status", func() {
	It("records a Failed condition when the namespace belongs to another owner", func(ctx SpecContext) {
		const name = "owned-by-someone-else"

		labelsPath := filepath.Join(GinkgoT().TempDir(), "namespace-labels.yaml")
		Expect(os.WriteFile(labelsPath, []byte("{}\n"), 0o600)).To(Succeed())

		Expect(k8sClient.Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Annotations: map[string]string{"owner": "alice@example.com"},
			},
		})).To(Succeed())
		Expect(k8sClient.Create(ctx, &profilev1.Profile{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec: profilev1.ProfileSpec{
				Owner: rbacv1.Subject{Kind: rbacv1.UserKind, Name: "bob@example.com"},
			},
		})).To(Succeed())

		r := &ProfileReconciler{
			Client:                     k8sClient,
			Scheme:                     scheme.Scheme,
			Log:                        ctrl.Log,
			DefaultNamespaceLabelsPath: labelsPath,
		}
		want := []profilev1.ProfileCondition{{
			Type:    profilev1.ProfileFailed,
			Message: "namespace already exist, but not owned by profile creator bob@example.com",
		}}

		// Reconcile twice: the condition must be stored once, not appended again.
		for range 2 {
			_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: name}})
			Expect(err).NotTo(HaveOccurred())

			got := &profilev1.Profile{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name}, got)).To(Succeed())
			Expect(got.Status.Conditions).To(Equal(want))
		}
	})
})
