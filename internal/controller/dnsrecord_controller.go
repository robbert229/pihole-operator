/*
Copyright 2024.

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

package controller

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"github.com/robbert229/pihole-operator/api/v1alpha1"
	piholev1alpha1 "github.com/robbert229/pihole-operator/api/v1alpha1"
	"github.com/robbert229/pihole-operator/internal/pihole"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var defaultRequeueDuration = time.Second * 4

// DnsRecordReconciler reconciles a DnsRecord object
type DnsRecordReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=pihole.lab.johnrowley.co,resources=dnsrecords,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pihole.lab.johnrowley.co,resources=dnsrecords/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=pihole.lab.johnrowley.co,resources=dnsrecords/finalizers,verbs=update
//+kubebuilder:rbac:groups=pihole.lab.johnrowley.co,resources=piholes,verbs=get;list;watch;
//+kubebuilder:rbac:groups=apps,resources=replicasets,verbs=get;list;watch;
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;
//+kubebuilder:rbac:groups=core,resources=endpoints,verbs=get;list;watch;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DnsRecord object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *DnsRecordReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("handling event", "req", req.String())

	var record piholev1alpha1.DnsRecord
	if err := r.Get(ctx, req.NamespacedName, &record); err != nil {
		// if we couldn't get the dns record then we probably don't want to retry.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var namespace corev1.Namespace
	if err := r.Get(ctx, client.ObjectKey{Name: req.Namespace}, &namespace); err != nil {
		logger.Error(err, "failed to get current namespace")
		return ctrl.Result{
			RequeueAfter: defaultRequeueDuration,
		}, err
	}

	var instanceList piholev1alpha1.PiHoleList
	if err := r.List(ctx, &instanceList); err != nil {
		logger.Error(err, "failed to enumerate all pihole instances")
		return ctrl.Result{
			RequeueAfter: defaultRequeueDuration,
		}, err
	}

	for _, instance := range instanceList.Items {
		// if the namespace selector exists, use it to verify that the instance
		// applies to the current namespace.
		if nsSelector := instance.Spec.DNSRecordNamespaceSelector; nsSelector != nil {
			selector, err := metav1.LabelSelectorAsSelector(nsSelector)
			if err != nil {
				return ctrl.Result{
					RequeueAfter: defaultRequeueDuration,
				}, err
			}

			if !selector.Matches(labels.Set(namespace.GetLabels())) {
				continue
			}
		}

		// if the dnsrecord selector exists use it to verify that the instance
		// applies to the current dnsrecord.
		if recordSelector := instance.Spec.DNSRecordSelector; recordSelector != nil {
			selector, err := metav1.LabelSelectorAsSelector(recordSelector)
			if err != nil {
				return ctrl.Result{
					RequeueAfter: defaultRequeueDuration,
				}, err
			}

			if !selector.Matches(labels.Set(record.GetLabels())) {
				continue
			}
		}

		var secret corev1.Secret
		if err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: getSecretName(&instance)}, &secret); err != nil {
			logger.Error(err, "failed to get secret")
			return ctrl.Result{
				RequeueAfter: defaultRequeueDuration,
			}, err
		}

		password, ok := secret.Data["password"]
		if !ok {
			const errMessage = "password does not exist on credentials secret"
			err := fmt.Errorf(errMessage)
			logger.Error(err, errMessage)

			return ctrl.Result{
				RequeueAfter: defaultRequeueDuration,
			}, err
		}

		var endpoints corev1.Endpoints
		if err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: getHeadlessServiceName(&instance)}, &endpoints); err != nil {
			logger.Error(err, "failed to get headless service")
			return ctrl.Result{
				RequeueAfter: defaultRequeueDuration,
			}, err
		}

		for _, subset := range endpoints.Subsets {
			for _, address := range subset.Addresses {
				client := pihole.New(pihole.Config{
					Password: string(password),
					URL:      fmt.Sprintf("http://%s", address.IP+":80"),
				})

				err := client.Login(ctx)
				if err != nil {
					return ctrl.Result{
						RequeueAfter: defaultRequeueDuration,
					}, err
				}

				result, err := r.reconcileDNSRecord(ctx, logger, record, instance, client)
				if err != nil {
					return result, err
				}
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *DnsRecordReconciler) reconcileDNSRecord(
	ctx context.Context,
	logger logr.Logger,
	record piholev1alpha1.DnsRecord,
	instance piholev1alpha1.PiHole,
	client *pihole.Client,
) (ctrl.Result, error) {
	cnameRecords, err := client.ListCNAMERecords(ctx)
	if err != nil {
		return ctrl.Result{
			RequeueAfter: defaultRequeueDuration,
		}, err
	}

	aRecords, err := client.ListDNSRecords(ctx)
	if err != nil {
		return ctrl.Result{
			RequeueAfter: defaultRequeueDuration,
		}, err
	}

	cnameMap := map[string]pihole.CNAMERecord{}
	for _, record := range cnameRecords {
		cnameMap[record.Domain] = record
	}

	aMap := map[string]pihole.DNSRecord{}
	for _, record := range aRecords {
		aMap[record.Domain] = record
	}

	logger.Info("creating record", "record", record)

	if specRecord := record.Spec.A; specRecord != nil {
		if _, ok := aMap[specRecord.Domain]; !ok {
			_, err = client.CreateDNSRecord(ctx, &pihole.DNSRecord{
				IP:     specRecord.IP,
				Domain: specRecord.Domain,
			})
			if err != nil {
				return ctrl.Result{
					RequeueAfter: defaultRequeueDuration,
				}, err
			}
		}
	} else if specRecord := record.Spec.CNAME; specRecord != nil {
		if _, ok := cnameMap[specRecord.Domain]; !ok {
			_, err = client.CreateCNAMERecord(ctx, &pihole.CNAMERecord{
				Domain: specRecord.Domain,
				Target: specRecord.Target,
			})
			if err != nil {
				return ctrl.Result{
					RequeueAfter: defaultRequeueDuration,
				}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DnsRecordReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&piholev1alpha1.DnsRecord{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(ue event.UpdateEvent) bool {
				oldObj, oldOk := ue.ObjectOld.(*v1alpha1.DnsRecord)
				newObj, newOk := ue.ObjectNew.(*v1alpha1.DnsRecord)
				if !oldOk || !newOk {
					return false
				}

				return !reflect.DeepEqual(oldObj.Spec, newObj.Spec)
			},
		}).
		Complete(r)
}
