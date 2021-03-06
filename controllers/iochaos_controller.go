// Copyright 2019 Chaos Mesh Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	"github.com/chaos-mesh/chaos-mesh/controllers/iochaos"
	"github.com/chaos-mesh/chaos-mesh/pkg/utils"
)

// IoChaosReconciler reconciles an IoChaos object
type IoChaosReconciler struct {
	client.Client
	client.Reader
	record.EventRecorder
	Log logr.Logger
}

// +kubebuilder:rbac:groups=chaos-mesh.org,resources=iochaos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chaos-mesh.org,resources=iochaos/status,verbs=get;update;patch

// Reconcile reconciles an IOChaos resource
func (r *IoChaosReconciler) Reconcile(req ctrl.Request) (result ctrl.Result, err error) {

	chaos := &v1alpha1.IoChaos{}
	if err := r.Client.Get(context.Background(), req.NamespacedName, chaos); err != nil {
		if apierrors.IsNotFound(err) {
			r.Log.Info("io chaos not found")
		} else {
			r.Log.Error(err, "unable to get io chaos")
		}
		return ctrl.Result{}, nil
	}

	scheduler := chaos.GetScheduler()
	duration, err := chaos.GetDuration()
	if err != nil {
		r.Log.Error(err, fmt.Sprintf("unable to get iochaos[%s/%s]'s duration", chaos.Namespace, chaos.Name))
		return ctrl.Result{}, err
	}
	if scheduler == nil && duration == nil {
		return r.commonIoChaos(chaos, req)
	} else if scheduler != nil && duration != nil {
		return r.scheduleIoChaos(chaos, req)
	}

	if err != nil {
		if chaos.IsDeleted() || chaos.IsPaused() {
			r.Event(chaos, v1.EventTypeWarning, utils.EventChaosRecoverFailed, err.Error())
		} else {
			r.Event(chaos, v1.EventTypeWarning, utils.EventChaosInjectFailed, err.Error())
		}
	}
	return result, nil
}

func (r *IoChaosReconciler) commonIoChaos(chaos *v1alpha1.IoChaos, req ctrl.Request) (ctrl.Result, error) {
	cr := iochaos.NewCommonReconciler(r.Client, r.Reader, r.Log.WithValues("iochaos", req.NamespacedName), req, r.EventRecorder)
	return cr.Reconcile(req)
}

func (r *IoChaosReconciler) scheduleIoChaos(chaos *v1alpha1.IoChaos, req ctrl.Request) (ctrl.Result, error) {
	sr := iochaos.NewTwoPhaseReconciler(r.Client, r.Reader, r.Log.WithValues("iochaos", req.NamespacedName), req, r.EventRecorder)
	return sr.Reconcile(req)
}

func (r *IoChaosReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.IoChaos{}).
		Complete(r)
}
