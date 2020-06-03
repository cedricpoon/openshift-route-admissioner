package main

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/golang/glog"
	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()

	// (https://github.com/kubernetes/kubernetes/issues/57982)
	defaulter = runtime.ObjectDefaulter(runtimeScheme)
)

var (
	ignoredNamespaces = []string{
		metav1.NamespaceSystem,
		metav1.NamespacePublic,
	}
	clientset *kubernetes.Clientset
)

const (
	routeLabelName    = "route-admissioner-label-map"
	keyName           = "key"
	mapName           = "map"
	allowedDomainName = "route-admissioner/allowed-domain"
)

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type labelMapKeyPair struct {
	Domain string `json:"domain"`
	Value  string `json:"value"`
}

// WebhookServer server object
type WebhookServer struct {
	server *http.Server
}

// WhSvrParameters Webhook Server parameters
type WhSvrParameters struct {
	port     int    // webhook server port
	certFile string // path to the x509 certificate for https
	keyFile  string // path to the x509 private key matching `CertFile`
}

func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionregistrationv1beta1.AddToScheme(runtimeScheme)
	// defaulting with webhooks:
	// https://github.com/kubernetes/kubernetes/issues/57982
	_ = v1.AddToScheme(runtimeScheme)

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
}

func admissionRequired(ignoredList []string, metadata *routev1.Route) bool {
	// skip special kubernetes system namespaces
	for _, namespace := range ignoredList {
		if metadata.Namespace == namespace {
			glog.Infof("Skip validation for %v for it's in special namespace:%v", metadata.Name, metadata.Namespace)
			return false
		}
	}
	return true
}

func validationRequired(ignoredList []string, metadata *routev1.Route) bool {
	required := admissionRequired(ignoredList, metadata)
	glog.Infof("Validation policy for %v/%v: required:%v", metadata.Namespace, metadata.Name, required)
	return required
}

func withListedSuffix(a string, list []string) bool {
	for _, b := range list {
		if strings.HasSuffix(a, b) {
			return true
		}
	}
	return false
}

func updateLabels(target map[string]string, added map[string]string) (patch []patchOperation) {
	values := make(map[string]string)
	for key, value := range added {
		if target == nil || target[key] == "" {
			values[key] = value
		}
	}
	patch = append(patch, patchOperation{
		Op:    "add",
		Path:  "/metadata/labels",
		Value: values,
	})
	return patch
}

func createPatch(availableLabels map[string]string, labels map[string]string) ([]byte, error) {
	var patch []patchOperation

	patch = append(patch, updateLabels(availableLabels, labels)...)

	return json.Marshal(patch)
}

// validate deployments and services
func (whsvr *WebhookServer) mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request
	var (
		route                           *routev1.Route
		resourceNamespace, resourceName string
		routeSpec                       *routev1.RouteSpec
		zone                            string
	)

	glog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, resourceName, req.UID, req.Operation, req.UserInfo)

	switch req.Kind.Kind {
	case "Route":
		if err := json.Unmarshal(req.Object.Raw, &route); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		resourceName, resourceNamespace, routeSpec = route.Name, route.Namespace, &route.Spec
	}

	if !validationRequired(ignoredNamespaces, route) {
		glog.Infof("Skipping validation for %s/%s due to policy check", resourceNamespace, resourceName)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	tenantNamespace, err := clientset.CoreV1().Namespaces().Get(route.GetNamespace(), metav1.GetOptions{})
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	if allowedDomainStr, ok := tenantNamespace.GetAnnotations()[allowedDomainName]; ok {
		allowedDomains := strings.Split(allowedDomainStr, ",")
		if !withListedSuffix(routeSpec.Host, allowedDomains) {
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: fmt.Sprintf(
						"Route not allowed. Allowed domains: %s", strings.Join(allowedDomains[:], ", ")),
				},
			}
		}
	}

	namespace, _ := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")

	configmap, err := clientset.CoreV1().ConfigMaps(string(namespace)).Get(routeLabelName, metav1.GetOptions{})
	if err != nil {
		return &v1beta1.AdmissionResponse{Allowed: true}
	}

	if labelMapStr, ok := configmap.Data[mapName]; ok {
		var labelMap []labelMapKeyPair
		if err := json.Unmarshal([]byte(labelMapStr), &labelMap); err != nil {
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		for _, obj := range labelMap {
			if withListedSuffix(routeSpec.Host, []string{obj.Domain}) {
				if obj.Value == "" {
					return &v1beta1.AdmissionResponse{Allowed: true}
				}
				zone = obj.Value
			}
		}
		if zone == "" {
			return &v1beta1.AdmissionResponse{Allowed: true}
		}
	} else {
		return &v1beta1.AdmissionResponse{Allowed: true}
	}

	patchBytes, err := createPatch(nil, map[string]string{configmap.Data[keyName]: zone})
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	glog.Infof("AdmissionResponse: patch=%v\n", string(patchBytes))
	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

// Serve method for webhook server
func (whsvr *WebhookServer) serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	ctx := r.Context()

	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		glog.Error("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		glog.Errorf("Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		glog.Errorf("Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		fmt.Println(r.URL.Path)
		if r.URL.Path == "/mutate" {
			admissionResponse = whsvr.mutate(&ar)
		}
	}

	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		glog.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	glog.Infof("Ready to write reponse ...")
	if _, err := w.Write(resp); err != nil {
		glog.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}

	// to validate server certificate expiry and log warning if expiring in a month
	validateCert(ctx.Value(CtxCert).(*x509.Certificate))
}
