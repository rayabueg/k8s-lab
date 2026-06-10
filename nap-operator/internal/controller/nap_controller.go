/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package controller

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	restv1alpha1 "github.com/rayabueg/nap-operator/api/v1alpha1"
)

// NapReconciler reconciles a Nap object.
type NapReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=rest.example.com,resources=naps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rest.example.com,resources=naps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=rest.example.com,resources=naps/finalizers,verbs=update

// Reconcile drives a Nap toward its desired state.
//
// Lifecycle:
//  1. First sight (WakesAt unset): parse spec.duration, set status.wakesAt = now + d,
//     phase = Sleeping, then requeue at d.
//  2. Wake (WakesAt <= now): log spec.message (or default) and set phase = Awake.
//  3. Already Awake: no-op.
func (r *NapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var nap restv1alpha1.Nap
	if err := r.Get(ctx, req.NamespacedName, &nap); err != nil {
		// Object was deleted between event and reconcile — nothing to do.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Terminal state.
	if nap.Status.Phase == restv1alpha1.NapPhaseAwake {
		return ctrl.Result{}, nil
	}

	// 1. First sight — start the nap.
	if nap.Status.WakesAt == nil {
		d, err := time.ParseDuration(nap.Spec.Duration)
		if err != nil {
			// Spec is bad. Logging once is enough; don't error-requeue forever.
			// (When you add a validating webhook later, this branch becomes dead code.)
			logger.Error(err, "invalid spec.duration; not requeueing", "duration", nap.Spec.Duration)
			return ctrl.Result{}, nil
		}

		wakes := metav1.NewTime(time.Now().Add(d))
		nap.Status.WakesAt = &wakes
		nap.Status.Phase = restv1alpha1.NapPhaseSleeping

		if err := r.Status().Update(ctx, &nap); err != nil {
			return ctrl.Result{}, err
		}

		logger.Info("nap started", "duration", d, "wakesAt", wakes.Time.Format(time.RFC3339))
		return ctrl.Result{RequeueAfter: d}, nil
	}

	// 2. Wake (or close to it).
	remaining := time.Until(nap.Status.WakesAt.Time)
	if remaining > 0 {
		// Reconciled early — e.g. a spec edit fired a watch event before the timer.
		// Just wait for the real wake.
		return ctrl.Result{RequeueAfter: remaining}, nil
	}

	msg := nap.Spec.Message
	if msg == "" {
		msg = "yawn... that was a good nap"
	}
	logger.Info(msg, "name", nap.Name, "overslept", -remaining)

	nap.Status.Phase = restv1alpha1.NapPhaseAwake
	if err := r.Status().Update(ctx, &nap); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager wires this reconciler into the controller manager.
func (r *NapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&restv1alpha1.Nap{}).
		Complete(r)
}
