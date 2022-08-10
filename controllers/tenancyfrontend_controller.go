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
	"fmt" // Basic functionalities

	"github.com/uiseong/sample-operator/api/v1alpha1"

	multitenancyv1alpha1 "github.com/uiseong/sample-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr" // Because of the cluster service target port definition
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// TenancyFrontendReconciler reconciles a TenancyFrontend object
type TenancyFrontendReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=multitenancy.example.com,resources=tenancyfrontends,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=multitenancy.example.com,resources=tenancyfrontends/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=multitenancy.example.com,resources=tenancyfrontends/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the TenancyFrontend object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *TenancyFrontendReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("")
	tenancyfrontend := &multitenancyv1alpha1.TenancyFrontend{}
	err := r.Get(ctx, req.NamespacedName, tenancyfrontend)

	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("TenancyFrontend resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get TenancyFrontend")
		return ctrl.Result{}, err
	}

	logger.Info("")

	found := &appsv1.Deployment{}

	err = r.Get(ctx, types.NamespacedName{Name: tenancyfrontend.Name, Namespace: tenancyfrontend.Namespace}, found)

	if err != nil && errors.IsNotFound(err) {
		dep := r.deploymentForTenancyFronted(tenancyfrontend, ctx)
		logger.Info("")
		err = r.Create(ctx, dep)
		if err != nil {
			logger.Error(err, "")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		logger.Error(err, "")
		return ctrl.Result{}, err
	}
	logger.Info("")
	return ctrl.Result{}, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *TenancyFrontendReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multitenancyv1alpha1.TenancyFrontend{}).
		Complete(r)
}

// deploymentForTenancyFronted returns a tenancyfrontend Deployment object
func (r *TenancyFrontendReconciler) deploymentForTenancyFronted(frontend *v1alpha1.TenancyFrontend, ctx context.Context) *appsv1.Deployment {
	logger := log.FromContext(ctx)
	ls := labelsForTenancyFrontend(frontend.Name, frontend.Name)
	replicas := frontend.Spec.Size

	// Just reflect the command in the deployment.yaml
	// for the ReadinessProbe and LivenessProbe
	// command: ["sh", "-c", "curl -s http://localhost:8080"]
	mycommand := make([]string, 3)
	mycommand[0] = "/bin/sh"
	mycommand[1] = "-c"
	mycommand[2] = "curl -s http://localhost:8080"

	// Using the context to log information
	logger.Info("Logging: Creating a new Deployment", "Replicas", replicas)
	message := "Logging: (Name: " + frontend.Name + ") \n"
	logger.Info(message)
	message = "Logging: (Namespace: " + frontend.Namespace + ") \n"
	logger.Info(message)

	for key, value := range ls {
		message = "Logging: (Key: [" + key + "] Value: [" + value + "]) \n"
		logger.Info(message)
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      frontend.Name,
			Namespace: frontend.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "quay.io/tsuedbroecker/service-frontend:latest",
						Name:  "service-frontend",
						Ports: []corev1.ContainerPort{{
							ContainerPort: 8080,
							Name:          "nginx-port",
						}},
						Env: []corev1.EnvVar{{
							Name: "VUE_APPID_DISCOVERYENDPOINT",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &v1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "appid.discovery-endpoint",
									},
									Key: "VUE_APPID_DISCOVERYENDPOINT",
								},
							}},
							{Name: "VUE_APPID_CLIENT_ID",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "appid.client-id-frontend",
										},
										Key: "VUE_APPID_CLIENT_ID",
									},
								}},
							{Name: "VUE_APP_API_URL_CATEGORIES",
								Value: "VUE_APP_API_URL_CATEGORIES_VALUE",
							},
							{Name: "VUE_APP_API_URL_PRODUCTS",
								Value: "VUE_APP_API_URL_PRODUCTS_VALUE",
							},
							{Name: "VUE_APP_API_URL_ORDERS",
								Value: "VUE_APP_API_URL_ORDERS_VALUE",
							},
							{Name: "VUE_APP_CATEGORY_NAME",
								Value: "VUE_APP_CATEGORY_NAME_VALUE",
							},
							{Name: "VUE_APP_HEADLINE",
								Value: frontend.Spec.DisplayName,
							},
							{Name: "VUE_APP_ROOT",
								Value: "/",
							}}, // End of Env listed values and Env definition
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								Exec: &corev1.ExecAction{Command: mycommand},
							},
							InitialDelaySeconds: 20,
						},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								Exec: &corev1.ExecAction{Command: mycommand},
							},
							InitialDelaySeconds: 20,
						},
					}}, // Container
				}, // PodSec
			}, // PodTemplateSpec
		}, // Spec
	} // Deployment

	// Set TenancyFrontend instance as the owner and controller
	ctrl.SetControllerReference(frontend, dep, r.Scheme)
	return dep
}

// labelsForTenancyFrontend returns the labels for selecting the resources
// belonging to the given tenancyfrontend CR name.
func labelsForTenancyFrontend(name_app string, name_cr string) map[string]string {
	return map[string]string{"app": name_app, "tenancyfrontend_cr": name_cr}
}

// ********************************************************
// additional functions
// Create Secret definition
func (r *TenancyFrontendReconciler) defineSecret(name string, namespace string, key string, value string, frontend *v1alpha1.TenancyFrontend) (*corev1.Secret, error) {
	secret := make(map[string]string)
	secret[key] = value

	sec := &corev1.Secret{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Immutable:  new(bool),
		Data:       map[string][]byte{},
		StringData: secret,
		Type:       "Opaque",
	}

	// Used to ensure that the secret will be deleted when the custom resource object is removed
	ctrl.SetControllerReference(frontend, sec, r.Scheme)

	return sec, nil
}

// Create Service NodePort definition

func (r *TenancyFrontendReconciler) defineServiceNodePort(name string, namespace string, frontend *v1alpha1.TenancyFrontend) (*corev1.Service, error) {
	// Define map for the selector
	mselector := make(map[string]string)
	key := "app"
	value := name
	mselector[key] = value

	// Define map for the labels
	mlabel := make(map[string]string)
	key = "app"
	value = "service-frontend"
	mlabel[key] = value

	var port int32 = 8080

	serv := &corev1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: mlabel},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{{
				Port: port,
				Name: "http",
			}},
			Selector: mselector,
		},
	}

	// Used to ensure that the service will be deleted when the custom resource object is removed
	ctrl.SetControllerReference(frontend, serv, r.Scheme)

	return serv, nil
}

// Create Service ClusterIP definition

func (r *TenancyFrontendReconciler) defineServiceClust(name string, namespace string, frontend *v1alpha1.TenancyFrontend) (*corev1.Service, error) {
	// Define map for the selector
	mselector := make(map[string]string)
	key := "app"
	value := name
	mselector[key] = value

	// Define map for the labels
	mlabel := make(map[string]string)
	key = "app"
	value = "service-frontend"
	mlabel[key] = value

	var port int32 = 80
	var targetPort int32 = 8080
	var clustserv = name + "clusterip"

	serv := &corev1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: clustserv, Namespace: namespace, Labels: mlabel},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{{
				Port:       port,
				TargetPort: intstr.IntOrString{IntVal: targetPort},
			}},
			Selector: mselector,
		},
	}

	// Used to ensure that the service will be deleted when the custom resource object is removed
	ctrl.SetControllerReference(frontend, serv, r.Scheme)

	return serv, nil
}

// Do all the tests for the status
//+kubebuilder:rbac:groups=multitenancy.example.net,resources=tenancyfrontends,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=multitenancy.example.net,resources=tenancyfrontends/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=multitenancy.example.net,resources=tenancyfrontends/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

func verifySecrectStatus(ctx context.Context, r *TenancyFrontendReconciler, targetSecretName string, targetSecret *v1.Secret, err error) error {
	logger := log.FromContext(ctx)

	if err != nil && errors.IsNotFound(err) {
		logger.Info(fmt.Sprintf("Target secret %s doesn't exist, creating it", targetSecretName))
		err = r.Create(context.TODO(), targetSecret)
		if err != nil {
			return err
		}
	} else {
		logger.Info(fmt.Sprintf("Target secret %s exists, updating it now", targetSecretName))
		err = r.Update(context.TODO(), targetSecret)
		if err != nil {
			return err
		}
	}

	return err
}
