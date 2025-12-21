package checker

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseGVRFromOwnerRef(t *testing.T) {
	tests := []struct {
		name      string
		ownerRef  *metav1.OwnerReference
		wantGroup string
		wantVer   string
		wantRes   string
		wantErr   bool
	}{
		{
			name: "ReplicaSet owner",
			ownerRef: &metav1.OwnerReference{
				APIVersion: "apps/v1",
				Kind:       "ReplicaSet",
			},
			wantGroup: "apps",
			wantVer:   "v1",
			wantRes:   "replicasets",
			wantErr:   false,
		},
		{
			name: "Deployment owner",
			ownerRef: &metav1.OwnerReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
			},
			wantGroup: "apps",
			wantVer:   "v1",
			wantRes:   "deployments",
			wantErr:   false,
		},
		{
			name: "JobSet owner (custom resource)",
			ownerRef: &metav1.OwnerReference{
				APIVersion: "jobset.x-k8s.io/v1alpha2",
				Kind:       "JobSet",
			},
			wantGroup: "jobset.x-k8s.io",
			wantVer:   "v1alpha2",
			wantRes:   "jobsets",
			wantErr:   false,
		},
		{
			name: "Core resource (Pod)",
			ownerRef: &metav1.OwnerReference{
				APIVersion: "v1",
				Kind:       "Pod",
			},
			wantGroup: "",
			wantVer:   "v1",
			wantRes:   "pods",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gvr, err := parseGVRFromOwnerRef(tt.ownerRef)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseGVRFromOwnerRef() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if gvr.Group != tt.wantGroup {
				t.Errorf("Group = %v, want %v", gvr.Group, tt.wantGroup)
			}
			if gvr.Version != tt.wantVer {
				t.Errorf("Version = %v, want %v", gvr.Version, tt.wantVer)
			}
			if gvr.Resource != tt.wantRes {
				t.Errorf("Resource = %v, want %v", gvr.Resource, tt.wantRes)
			}
		})
	}
}

func TestCollectOwnerChain(t *testing.T) {
	// Note: This test requires full Clientset and RESTConfig which is not available in unit tests
	// collectOwnerChainWithStatus needs these to create ResourceStatus for each owner
	t.Skip("collectOwnerChainWithStatus requires Clientset and RESTConfig for integration testing")
}

func TestCollectOwnerChain_SkipWorkload(t *testing.T) {
	// Note: This test requires full Clientset and RESTConfig which is not available in unit tests
	t.Skip("collectOwnerChainWithStatus requires Clientset and RESTConfig for integration testing")
}

func TestParseGVRFromOwnerRef_ResourceNameGeneration(t *testing.T) {
	// Test that resource names are correctly pluralized
	tests := []struct {
		kind string
		want string
	}{
		{kind: "Pod", want: "pods"},
		{kind: "Deployment", want: "deployments"},
		{kind: "Job", want: "jobs"},
		{kind: "CronJob", want: "cronjobs"},
		{kind: "StatefulSet", want: "statefulsets"},
		{kind: "DaemonSet", want: "daemonsets"},
		{kind: "ReplicaSet", want: "replicasets"},
		{kind: "Workload", want: "workloads"},
		{kind: "JobSet", want: "jobsets"},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			ownerRef := &metav1.OwnerReference{
				APIVersion: "apps/v1",
				Kind:       tt.kind,
			}
			gvr, err := parseGVRFromOwnerRef(ownerRef)
			if err != nil {
				t.Errorf("parseGVRFromOwnerRef() error = %v", err)
				return
			}
			if gvr.Resource != tt.want {
				t.Errorf("Resource = %v, want %v", gvr.Resource, tt.want)
			}
		})
	}
}
