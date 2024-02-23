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
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	piholev1alpha1 "github.com/robbert229/pihole-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PiHoleReconciler reconciles a PiHole object
type PiHoleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func getPiHoleVersionFromSpec(instance *piholev1alpha1.PiHole) string {
	if instance.Spec.Version == nil {
		return "2024.02.0"
	}

	return *instance.Spec.Version
}

func getPiHoleImageFromSpec(instance *piholev1alpha1.PiHole) string {
	return "pihole/pihole" + ":" + getPiHoleVersionFromSpec(instance)
}

func getSecretName(instance *piholev1alpha1.PiHole) string {
	return fmt.Sprintf("%s-credentials", instance.Name)
}

func getReplicaSetName(instance *piholev1alpha1.PiHole) string {
	return instance.Name
}

func getHeadlessServiceName(instance *piholev1alpha1.PiHole) string {
	return fmt.Sprintf("%s-headless", instance.Name)
}

func getServiceName(instance *piholev1alpha1.PiHole) string {
	return instance.Name
}

func generatePassword() (string, error) {
	const passwordLength = 32
	randomBytes := make([]byte, passwordLength)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("unable to generate random bytes for password: %w", err)
	}

	password := base64.URLEncoding.EncodeToString(randomBytes)
	password = password[:passwordLength]

	return password, nil
}

func constructSecretForPiHole(instance *piholev1alpha1.PiHole, scheme *runtime.Scheme) (*corev1.Secret, error) {
	name := getSecretName(instance)

	password, err := generatePassword()
	if err != nil {
		return nil, fmt.Errorf("failed to generate password for secret: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			Name:        name,
			Namespace:   instance.Namespace,
		},
		Data: map[string][]byte{
			"password": []byte(password),
		},
	}

	if err := ctrl.SetControllerReference(instance, secret, scheme); err != nil {
		return nil, err
	}

	return secret, nil
}

var servicePorts = []corev1.ServicePort{
	{
		Name:     "dns-tcp",
		Port:     53,
		Protocol: corev1.ProtocolTCP,
	},
	{
		Name:     "dns-udp",
		Port:     53,
		Protocol: corev1.ProtocolUDP,
	},
	{
		Name:     "dhcp",
		Port:     67,
		Protocol: corev1.ProtocolUDP,
	},
	{
		Name:     "http",
		Port:     80,
		Protocol: corev1.ProtocolTCP,
	},
}

func getResourceLabels(instance *piholev1alpha1.PiHole) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "pihole",
		"app.kubernetes.io/instance":   instance.Name,
		"app.kubernetes.io/managed-by": "pihole-operator",
	}
}

func getServiceSelector(instance *piholev1alpha1.PiHole) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":     "pihole",
		"app.kubernetes.io/instance": instance.Name,
	}
}

func constructHeadlessServiceForPiHole(instance *piholev1alpha1.PiHole, scheme *runtime.Scheme) (*corev1.Service, error) {
	name := getHeadlessServiceName(instance)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      getResourceLabels(instance),
			Annotations: make(map[string]string),
			Name:        name,
			Namespace:   instance.Namespace,
		},

		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "None",
			Ports:     servicePorts,
			Selector:  getServiceSelector(instance),
		},
	}

	return svc, nil
}

func constructServiceForPiHole(instance *piholev1alpha1.PiHole, scheme *runtime.Scheme) (*corev1.Service, error) {
	name := getServiceName(instance)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      getResourceLabels(instance),
			Annotations: make(map[string]string),
			Name:        name,
			Namespace:   instance.Namespace,
		},

		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "",
			Ports:     servicePorts,
			Selector:  getServiceSelector(instance),
		},
	}

	return svc, nil
}

func constructReplicaSetForPiHole(instance *piholev1alpha1.PiHole, scheme *runtime.Scheme) (*appsv1.ReplicaSet, error) {
	name := instance.Name

	dnsUpstreams := strings.Join(instance.Spec.DNS.UpstreamServers, ";")

	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"app.kubernetes.io/name":       "pihole",
				"app.kubernetes.io/instance":   instance.Name,
				"app.kubernetes.io/managed-by": "pihole-operator",
			},
			Annotations: make(map[string]string),
			Name:        name,
			Namespace:   instance.Namespace,
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: instance.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: getServiceSelector(instance),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: getServiceSelector(instance),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "pihole",
							Image: getPiHoleImageFromSpec(instance),
							Env: []corev1.EnvVar{
								{
									Name: "WEBPASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: getSecretName(instance),
											},
											Key: "password",
										},
									},
								},
								{
									Name:  "PIHOLE_DNS_",
									Value: dnsUpstreams,
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "dns-tcp",
									ContainerPort: 53,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "dns-udp",
									ContainerPort: 53,
									Protocol:      corev1.ProtocolUDP,
								},
								{
									Name:          "dhcp",
									ContainerPort: 67,
									Protocol:      corev1.ProtocolUDP,
								},
								{
									Name:          "http",
									ContainerPort: 80,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"NET_ADMIN"},
								},
							},

							Resources: instance.Spec.Resources,
						},
					},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(instance, replicaSet, scheme); err != nil {
		return nil, err
	}

	return replicaSet, nil
}

//+kubebuilder:rbac:groups=apps,resources=replicasets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pihole.lab.johnrowley.co,resources=piholes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pihole.lab.johnrowley.co,resources=piholes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=pihole.lab.johnrowley.co,resources=piholes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PiHole object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *PiHoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("handling event", "req", req.String())

	var instance piholev1alpha1.PiHole
	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// if the instance is paused then do nothing.
	if instance.Spec.Paused {
		return ctrl.Result{}, nil
	}

	// reconcile secrets. This generates the credentials for the pihole
	// instance.
	if err := r.reconcileSecret(ctx, req, logger, instance); err != nil {
		return ctrl.Result{}, nil
	}

	// reconcile replica sets. This deploys the replicaset that runs the pihole
	// image.
	if err := r.reconcileReplicaSet(ctx, req, logger, instance); err != nil {
		return ctrl.Result{}, nil
	}

	// reconcile headless service. The headless service is responsible for
	// ensuring that we can access each of the pihole instances directly.
	if err := r.reconcileHeadlessService(ctx, req, logger, instance); err != nil {
		return ctrl.Result{}, nil
	}

	// reconcile headed service.
	if err := r.reconcileService(ctx, req, logger, instance); err != nil {
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *PiHoleReconciler) reconcileSecret(
	ctx context.Context,
	req ctrl.Request,
	logger logr.Logger,
	instance piholev1alpha1.PiHole,
) error {
	var secret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: getSecretName(&instance)}, &secret); err != nil {
		logger.Info("creating Secret for PiHole")
		if !errors.IsNotFound(err) {
			logger.Error(err, "unable to retrieve Secret for PiHole")
			return err
		}

		secret, err := constructSecretForPiHole(&instance, r.Scheme)
		if err != nil {
			logger.Error(err, "unable to construct Secret for PiHole")
			return err
		}

		if err := r.Create(ctx, secret); err != nil {
			logger.Error(err, "unable to create Secret for PiHole")
			return err
		}

		return nil
	}

	return nil
}

func (r *PiHoleReconciler) reconcileReplicaSet(
	ctx context.Context,
	req ctrl.Request,
	logger logr.Logger,
	instance piholev1alpha1.PiHole,
) error {
	var rs appsv1.ReplicaSet

	if err := r.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: getReplicaSetName(&instance)}, &rs); err != nil {
		logger.Info("creating ReplicaSet for PiHole")

		if !errors.IsNotFound(err) {
			logger.Error(err, "unable to retrieve ReplicaSet for PiHole")
			return err
		}

		rs, err := constructReplicaSetForPiHole(&instance, r.Scheme)
		if err != nil {
			logger.Error(err, "unable to construct ReplicaSet for PiHole")
			return err
		}

		if err := r.Create(ctx, rs); err != nil {
			logger.Error(err, "unable to create ReplicaSet for PiHole")
			return err
		}
	}

	logger.Info("patching ReplicaSet for PiHole")

	newRS, err := constructReplicaSetForPiHole(&instance, r.Scheme)
	if err != nil {
		logger.Error(err, "unable to construct ReplicaSet for PiHole")
		return err
	}

	err = r.Update(ctx, newRS)
	if err != nil {
		logger.Error(err, "unable to update ReplicaSet for PiHole")
		return err
	}

	return nil
}

func (r *PiHoleReconciler) reconcileHeadlessService(
	ctx context.Context,
	req ctrl.Request,
	logger logr.Logger,
	instance piholev1alpha1.PiHole,
) error {
	var svc corev1.Service

	if err := r.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: getHeadlessServiceName(&instance)}, &svc); err != nil {
		logger.Info("creating headless Service for PiHole")

		if !errors.IsNotFound(err) {
			logger.Error(err, "unable to retrieve headless Service for PiHole")
			return err
		}

		svc, err := constructHeadlessServiceForPiHole(&instance, r.Scheme)
		if err != nil {
			logger.Error(err, "unable to construct headless Service for PiHole")
			return err
		}

		if err := r.Create(ctx, svc); err != nil {
			logger.Error(err, "unable to create headless Service for PiHole")
			return err
		}
	}

	logger.Info("patching headless Service for PiHole")

	newSvc, err := constructHeadlessServiceForPiHole(&instance, r.Scheme)
	if err != nil {
		logger.Error(err, "unable to construct headless Service for PiHole")
		return err
	}

	err = r.Update(ctx, newSvc)
	if err != nil {
		logger.Error(err, "unable to update headless Service for PiHole")
		return err
	}

	return nil
}

func (r *PiHoleReconciler) reconcileService(
	ctx context.Context,
	req ctrl.Request,
	logger logr.Logger,
	instance piholev1alpha1.PiHole,
) error {
	var svc corev1.Service

	if err := r.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: getServiceName(&instance)}, &svc); err != nil {
		logger.Info("creating Service for PiHole")

		if !errors.IsNotFound(err) {
			logger.Error(err, "unable to retrieve Service for PiHole")
			return err
		}

		svc, err := constructServiceForPiHole(&instance, r.Scheme)
		if err != nil {
			logger.Error(err, "unable to construct Service for PiHole")
			return err
		}

		if err := r.Create(ctx, svc); err != nil {
			logger.Error(err, "unable to create Service for PiHole")
			return err
		}
	}

	logger.Info("patching Service for PiHole")

	newSvc, err := constructServiceForPiHole(&instance, r.Scheme)
	if err != nil {
		logger.Error(err, "unable to construct Service for PiHole")
		return err
	}

	err = r.Update(ctx, newSvc)
	if err != nil {
		logger.Error(err, "unable to update Service for PiHole")
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PiHoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&piholev1alpha1.PiHole{}).
		Owns(&appsv1.ReplicaSet{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
