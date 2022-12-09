/*
Copyright 2022.

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
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv1beta1 "github.com/lianyz/k8s-operator/api/v1beta1"
)

// MyDaemonsetReconciler reconciles a MyDaemonset object
type MyDaemonsetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps.cncamp.io,resources=mydaemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.cncamp.io,resources=mydaemonsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.cncamp.io,resources=mydaemonsets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MyDaemonset object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.1/pkg/reconcile
func (r *MyDaemonsetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	myLogger := log.FromContext(ctx)
	myLogger.Info("Reconcile MyDaemonset", "NamespacedName", req.NamespacedName)

	myDaemonSet := appsv1beta1.MyDaemonset{}
	if err := r.Client.Get(ctx, req.NamespacedName, &myDaemonSet); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get %s", req.NamespacedName)
	}
	replicas := myDaemonSet.Spec.Replicas
	if replicas == 0 {
		replicas = 2
	}
	image := myDaemonSet.Spec.Image
	if image == "" {
		image = "nginx"
	}

	for i := 0; i < replicas; i++ {
		name := fmt.Sprintf("%s-%d", myDaemonSet.Name, i)
		err := r.Client.Get(ctx, types.NamespacedName{Namespace: myDaemonSet.Namespace, Name: name}, &v1.Pod{})
		if err == nil {
			myLogger.Info("Pod exists", "name", name)
			continue
		}

		if !errors.IsNotFound(err) {
			myLogger.Info("Failed to get pod", "err", err)
			return ctrl.Result{}, fmt.Errorf("failed to get pod")
		}

		pod := newPod(name, image, &myDaemonSet)

		if err := r.Client.Create(ctx, &pod); err != nil {
			myLogger.Info("Failed to create pod", "err", err)
			return ctrl.Result{}, fmt.Errorf("failed to create pod")
		}

		toBeUpdate := myDaemonSet.DeepCopy()
		toBeUpdate.Status.AvailableReplicas += 1
		if err := r.Client.Status().Update(ctx, toBeUpdate); err != nil {
			myLogger.Info("Failed to update myDaemonSet status", "err", err)
			return ctrl.Result{}, fmt.Errorf("failed to update myDaemonSet status")
		}

	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyDaemonsetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1beta1.MyDaemonset{}).
		Complete(r)
}

func newPod(podName, imageName string, myDaemonSet *appsv1beta1.MyDaemonset) v1.Pod {
	pod := v1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: myDaemonSet.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps.crane.io/v1alpha1",
					Kind:       "MyDaemonset",
					Name:       myDaemonSet.Name,
					UID:        myDaemonSet.UID,
				},
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Image: imageName,
					Name:  podName,
				},
			},
		},
	}

	return pod
}
