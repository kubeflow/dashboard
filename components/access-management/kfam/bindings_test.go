package kfam

import (
	"sort"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	listers "k8s.io/client-go/listers/rbac/v1"
)

// Building Binding Object from k8s.io/api/rbac/v1
func getBindingObject(binding string) *Binding {
	return &Binding{
		Subject: &rbacv1.Subject{
			Kind: rbacv1.UserKind,
			Name: binding,
		},
		RoleRef: &rbacv1.RoleRef{
			Kind: "clusterrole",
			Name: "edit",
		},
	}
}

func TestGetBindingName(t *testing.T) {
	// Table driven tests
	tests := []struct {
		name     string
		in       *Binding
		out      string
		hasError bool
	}{
		{"letters", getBindingObject("lalith.vaka@zq.msds.kp.org"), "user-lalith-vaka-zq-msds-kp-org-clusterrole-edit", false},
		{"numbers", getBindingObject("397401@zq.msds.kp.org"), "user-397401-zq-msds-kp-org-clusterrole-edit", false},
		{"letters-numbers", getBindingObject("lalith.397401@zq.msds.kp.org"), "user-lalith-397401-zq-msds-kp-org-clusterrole-edit", false},
		{"numbers-letters", getBindingObject("397401.vaka@zq.msds.kp.org"), "user-397401-vaka-zq-msds-kp-org-clusterrole-edit", false},
		{"lettersnumbers", getBindingObject("i397401@zq.msds.kp.org"), "user-i397401-zq-msds-kp-org-clusterrole-edit", false},
	}

	// format := "--- %s: %s (%s)\n"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, errorReturned := getBindingName(tt.in)
			if tt.hasError {
				// expected an error
				if errorReturned == nil {
					t.Fatalf("Expected error but got none:  input: %q", tt.in)
				}
			} else {
				// expected a value
				if errorReturned != nil {
					t.Fatalf("unExpected occured:  input: %q, errorReturned: %q", tt.in, errorReturned)
				}
				if s != tt.out {
					t.Fatalf("Value different than expected: input: %q, output: %q", s, tt.out)
				}
			}
		})
	}
}

// newTestBindingClient builds a BindingClient backed by a real in-memory lister
// populated with the given RoleBindings.
func newTestBindingClient(rbs []*rbacv1.RoleBinding) *BindingClient {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{
		cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
	})
	for _, rb := range rbs {
		indexer.Add(rb)
	}
	return &BindingClient{roleBindingLister: listers.NewRoleBindingLister(indexer)}
}

func userRoleBinding(name, namespace, user, k8sRole string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				USER: user,
				ROLE: roleBindingNameMap[k8sRole], // store frontend role name
			},
		},
		Subjects: []rbacv1.Subject{{Kind: rbacv1.UserKind, Name: user}},
		RoleRef:  rbacv1.RoleRef{Kind: "ClusterRole", Name: k8sRole},
	}
}

func groupRoleBinding(name, namespace, group, k8sRole string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				GROUP: group,
				ROLE:  roleBindingNameMap[k8sRole], // store frontend role name
			},
		},
		Subjects: []rbacv1.Subject{{Kind: rbacv1.GroupKind, Name: group}},
		RoleRef:  rbacv1.RoleRef{Kind: "ClusterRole", Name: k8sRole},
	}
}

// subjectNames extracts subject names from BindingEntries for easy comparison.
func subjectNames(entries *BindingEntries) []string {
	names := make([]string, 0, len(entries.Bindings))
	for _, b := range entries.Bindings {
		names = append(names, b.Subject.Name)
	}
	sort.Strings(names)
	return names
}

func TestBindingClientList(t *testing.T) {
	const ns = "test-ns"

	allBindings := []*rbacv1.RoleBinding{
		// 2 user bindings
		userRoleBinding("user-alice-edit", ns, "alice", "edit"),
		userRoleBinding("user-bob-view", ns, "bob", "view"),
		// 3 group bindings
		groupRoleBinding("group-team-a-edit", ns, "team-a", "edit"),
		groupRoleBinding("group-team-b-view", ns, "team-b", "view"),
		groupRoleBinding("group-team-c-admin", ns, "team-c", "admin"),
	}

	tests := []struct {
		name         string
		user         string
		groups       []string
		role         string
		wantSubjects []string
	}{
		{
			name:         "filter by user alice",
			user:         "alice",
			wantSubjects: []string{"alice"},
		},
		{
			name:         "filter by user bob",
			user:         "bob",
			wantSubjects: []string{"bob"},
		},
		{
			name:         "filter by single group team-a",
			groups:       []string{"team-a"},
			wantSubjects: []string{"team-a"},
		},
		{
			name:         "filter by all three groups",
			groups:       []string{"team-a", "team-b", "team-c"},
			wantSubjects: []string{"team-a", "team-b", "team-c"},
		},
		{
			name:         "no filter returns all bindings",
			wantSubjects: []string{"alice", "bob", "team-a", "team-b", "team-c"},
		},
		{
			name:         "filter by unknown user returns empty",
			user:         "unknown",
			wantSubjects: []string{},
		},
		{
			name:         "filter by unknown group returns empty",
			groups:       []string{"unknown-group"},
			wantSubjects: []string{},
		},
		{
			name:         "filter by user alice and group team-a returns both",
			user:         "alice",
			groups:       []string{"team-a"},
			wantSubjects: []string{"alice", "team-a"},
		},
		{
			name:         "combination of user and groups filter returns all matching",
			user:         "alice",
			groups:       []string{"team-b"},
			wantSubjects: []string{"alice", "team-b"},
		},
	}

	client := newTestBindingClient(allBindings)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := client.List(tt.user, tt.groups, []string{ns}, tt.role)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := subjectNames(entries)
			want := tt.wantSubjects
			sort.Strings(want)
			if len(got) != len(want) {
				t.Fatalf("got %d bindings %v, want %d %v", len(got), got, len(want), want)
			}
			for i := range got {
				if got[i] != want[i] {
					t.Errorf("subject[%d]: got %q, want %q", i, got[i], want[i])
				}
			}
		})
	}
}
