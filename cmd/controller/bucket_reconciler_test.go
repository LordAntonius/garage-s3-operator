package main

import (
	"context"
	"testing"
	"time"

	v1 "abucquet.com/garage-s3-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestBucketReconciler_RequeuesAfterTransientError(t *testing.T) {
	// A bucket referencing a GarageS3Instance that does not exist simulates the
	// situation where the Garage API is unreachable (the reconciler cannot look
	// up the instance or reach the API). The reconciler must return a
	// RequeueAfter so the controller work queue retries instead of silently
	// dropping the item.

	bucket := &v1.GarageS3Bucket{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-bucket",
			Namespace: "default",
		},
		Spec: v1.GarageS3BucketSpec{
			InstanceRef: v1.GarageS3InstanceRef{
				Name:      "missing-instance",
				Namespace: "default",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(bucket).
		WithStatusSubresource(bucket).
		Build()

	r := &bucket_reconciler{
		Client: fakeClient,
		scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-bucket",
			Namespace: "default",
		},
	}

	result, err := r.Reconcile(context.Background(), req)
	if err == nil {
		t.Fatal("expected error from reconcile when instance is missing, got nil")
	}
	if result.RequeueAfter == 0 {
		t.Fatal("expected RequeueAfter > 0 so reconciliation retries after transient failure, got 0")
	}
	if result.RequeueAfter != bucketErrorRequeueInterval {
		t.Errorf("expected RequeueAfter=%v, got %v", bucketErrorRequeueInterval, result.RequeueAfter)
	}
}

func TestBucketReconciler_RequeuesOnSuccess(t *testing.T) {
	// Even after a fully successful reconciliation the bucket reconciler should
	// schedule a periodic requeue so it can recover from transient failures and
	// detect drift. We cannot easily simulate a full successful reconciliation
	// without a real Garage API, but we can verify the constant is defined with
	// a sensible value.

	if bucketRequeueInterval <= 0 {
		t.Fatal("bucketRequeueInterval must be positive for periodic requeue")
	}
	if bucketRequeueInterval > 30*time.Minute {
		t.Errorf("bucketRequeueInterval=%v seems too long for timely recovery", bucketRequeueInterval)
	}
}

func TestBucketReconciler_NotFoundDoesNotRequeue(t *testing.T) {
	// When the bucket CR itself is deleted (not found), the reconciler should
	// return a zero result with no error and no requeue.

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	r := &bucket_reconciler{
		Client: fakeClient,
		scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "nonexistent-bucket",
			Namespace: "default",
		},
	}

	result, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error for not-found bucket, got: %v", err)
	}
	if result.RequeueAfter != 0 || result.Requeue {
		t.Errorf("expected zero result for not-found bucket, got: %+v", result)
	}
}
