package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

type memoryCache struct {
	*discovery.DiscoveryClient
}

func (c *memoryCache) Fresh() bool {
	//always return true since we're not caching
	return true
}

func (c *memoryCache) Invalidate() {
	// No-op for in-memory cache
}


type Client struct {
	Config	 *rest.Config
	Clientset *kubernetes.Clientset
	Dynamic dynamic.Interface
	RESTMapper *restmapper.DeferredDiscoveryRESTMapper
	DiscoveryCli *discovery.DiscoveryClient
}

func NewK8sClient() (*Client, error) {
	var cfg *rest.Config
	var err error
	kubeconfig := os.Getenv("KUBECONFIG")

	if kubeconfig != "" {
		cfg,err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}else{
		cfg, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes config: %v", err)
	}

	clientset,err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %v", err)
	}

	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %v", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %v", err)
	}

	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(&memoryCache{discoveryClient})


	return &Client{
		Config:        cfg,
		Clientset:     clientset,
		Dynamic:       dynamicClient,
		RESTMapper:    restMapper,
		DiscoveryCli:  discoveryClient,
	},nil
		
}

func RenderTemplate(projectType string, replace map[string]string) ([]byte, error) {
	fileName := "./manifests/" + projectType + ".yaml"
	yaml,err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %v", err)
	}
	for key, value := range replace {
		yaml = []byte(strings.ReplaceAll(string(yaml), "{{"+key+"}}", value))
	}

	return yaml, nil
}

func (c * Client) ApplyManifest(ctx context.Context, manifest []byte, namespace string) error {
	//decode yaml manifest:
	dec := yaml.NewYAMLOrJSONDecoder(strings.NewReader(string(manifest)), 4096)
	var rawObj map[string]interface{}
	for {

		if err := dec.Decode(&rawObj); err != nil {
			if err.Error() == "EOF" {
				break
			}

			return fmt.Errorf("Failed to decode manifest: %v", err)
		}
	
		u := &unstructured.Unstructured{Object: rawObj}
	
		//create mapping:
		apiVer := u.GetAPIVersion()
		kind := u.GetKind()
		gv,err := schema.ParseGroupVersion(apiVer)
		if err != nil{
			return fmt.Errorf("Error while parsing GV: %v", err)
		}
		gk:= schema.GroupKind{Group: gv.Group, Kind: kind}
		
		mapping,err := c.RESTMapper.RESTMapping(gk,gv.Version)
		if err != nil{
			c.RESTMapper.Reset()
			mapping,err = c.RESTMapper.RESTMapping(gk,gv.Version)
			if err != nil{
				return fmt.Errorf("Failed RESTMapping for %s %s: %v", apiVer,kind,err);
			}
		}

		//create or update resource
		name := u.GetName()
		var resourceInterface dynamic.ResourceInterface

		resourceInterface = c.Dynamic.Resource(mapping.Resource).Namespace(namespace)

		resourceInterface.Create(ctx,u,metav1.CreateOptions{})
		if err == nil{
			continue
		}
		
		if errors.IsAlreadyExists(err){
			existing,getErr := resourceInterface.Get(ctx,name,metav1.GetOptions{})
			if getErr != nil{
				return fmt.Errorf("Error Getting existing resource %s %s: %v",kind,name,getErr);
			}

			u.SetResourceVersion(existing.GetResourceVersion())
			_,updateErr := resourceInterface.Update(ctx,u,metav1.UpdateOptions{})
			if updateErr != nil{
				return fmt.Errorf("Error Update existing resource %s %s: %v", kind,name, updateErr)
			}
			continue
		}
		return fmt.Errorf("Error Creating Resource %s %s: %v", kind,name,err)
	}

	return nil
}
