package k8s

import (
	"context"
	"fmt"
	"io"
	"os"
//	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
//	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
//	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
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


var workspaceId string
var namespace string = "cloud-ide"


type Client struct {
	Config	 *rest.Config
	Clientset *kubernetes.Clientset
	Dynamic dynamic.Interface
	RESTMapper *restmapper.DeferredDiscoveryRESTMapper
	DiscoveryCli *discovery.DiscoveryClient
}

func NewK8sClient(workId string) (*Client, error) {
	workspaceId = workId

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

/* todo: Replace function
func (c *Client) ApplyManifest(ctx context.Context, manifest []byte) error {
	//decode yaml manifest:
	dec := yaml.NewYAMLOrJSONDecoder(strings.NewReader(string(manifest)), 4096)
	var rawObj map[string]interface{}
	for {

		if err := dec.Decode(&rawObj); err != nil {
			if err.Error() == "EOF" {
				break
			}

			return fmt.Errorf("failed to decode manifest: %v", err)
		}
	
		u := &unstructured.Unstructured{Object: rawObj}
	
		//create mapping:
		apiVer := u.GetAPIVersion()
		kind := u.GetKind()
		gv,err := schema.ParseGroupVersion(apiVer)
		if err != nil{
			return fmt.Errorf("error while parsing GV: %v", err)
		}
		gk:= schema.GroupKind{Group: gv.Group, Kind: kind}
		
		mapping,err := c.RESTMapper.RESTMapping(gk,gv.Version)
		if err != nil{
			c.RESTMapper.Reset()
			mapping,err = c.RESTMapper.RESTMapping(gk,gv.Version)
			if err != nil{
				return fmt.Errorf("failed RESTMapping for %s %s: %v", apiVer,kind,err)
			}
		}

		//create or update resource:
		name := u.GetName()

		resourceInterface := c.Dynamic.Resource(mapping.Resource).Namespace(namespace)

		_,err = resourceInterface.Create(ctx,u,metav1.CreateOptions{})
		if err == nil{
			continue
		}
		
		if errors.IsAlreadyExists(err){
			existing,getErr := resourceInterface.Get(ctx,name,metav1.GetOptions{})
			if getErr != nil{
				return fmt.Errorf("error getting existing resource %s %s: %v",kind,name,getErr)
			}

			u.SetResourceVersion(existing.GetResourceVersion())
			_,updateErr := resourceInterface.Update(ctx,u,metav1.UpdateOptions{})
			if updateErr != nil{
				return fmt.Errorf("error updating existing resource %s %s: %v", kind,name, updateErr)
			}
			continue
		}
		return fmt.Errorf("error creating resource %s %s: %v", kind,name,err)
	}

	return nil
}
*/

func (c *Client) DeleteResource(ctx context.Context, kind string, name string, namespace string) error {
	var gvrMap = map[string]schema.GroupVersionResource{
		"Job": {Group: "batch", Version: "v1", Resource: "jobs"},
		"Pod": {Group: "", Version: "v1", Resource:"pods"},
		"Deployment": {Group: "apps", Version: "v1", Resource: "deployments"},
		"Service": {Group: "", Version: "v1", Resource: "services"},
		"Ingress": {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
		"Role": {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles"},
		"RoleBinding": {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"},
		"ServiceAccount": {Group: "", Version: "v1", Resource: "serviceaccounts"},
	}

	gvr, ok := gvrMap[kind]
	if !ok{
		return fmt.Errorf("error finding GVR of kind %s",kind)
	}

	if namespace == ""{
		namespace = "default"
	}

	return c.Dynamic.Resource(gvr).Namespace(namespace).Delete(ctx,name,metav1.DeleteOptions{})
}

func (c *Client) WaitForPodByLabel(ctx context.Context, namespace string, labelSelector string, timeout time.Duration) (string,error){
	tctx,cancel := context.WithTimeout(ctx,timeout)
	defer cancel()

	watcher,err := c.Clientset.CoreV1().Pods(namespace).Watch(ctx,metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err!=nil{
		return "", fmt.Errorf("error watching pod with label %s: %v", labelSelector,err)
	}

	ch := watcher.ResultChan()
	defer watcher.Stop()

	list,err := c.Clientset.CoreV1().Pods(namespace).List(ctx,metav1.ListOptions{LabelSelector: labelSelector})

	if err==nil{
		for _,p := range list.Items{
			if isPodReady(&p){
				return p.Name,nil
			}
		}
	}

	for{
		select{
		case ev,ok := <-ch:
			if !ok{
				return "",fmt.Errorf("pod watch channel closed")
			}
			if ev.Object == nil{
				continue
			}
			pod := ev.Object.(*corev1.Pod)
			if isPodReady(pod){
				return pod.Name,nil
			}
		case <-tctx.Done():
			return "",fmt.Errorf("timeout waiting for pod with label %s", labelSelector)
		}
	}
}

func isPodReady(p *corev1.Pod) bool{
	if p.Status.Phase != corev1.PodRunning{
		return false
	}

	for _,c := range p.Status.Conditions{
		if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue{
			return true
		}
	}

	return false
}

func (c *Client) StreamPodLogs(ctx context.Context, namespace string, podName string, follow bool, writer io.Writer) error {
	req := c.Clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{Follow: follow})
	stream, err := req.Stream(ctx)
	if err != nil{
		return fmt.Errorf("error getting log stream: %v",err)
	}
	defer stream.Close()

	_,err = io.Copy(writer, stream)
	return err
}

func (c *Client) ExecToPod(ctx context.Context, namespace string, podName string, container string, command []string, stdin io.Reader, stdout, stderr io.Writer,tty bool) error{
	req := c.Clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command: command,
			Container: container,
			Stdin: stdin!=nil,
			Stdout: stdout!=nil,
			Stderr: stderr!=nil,
			TTY: tty,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(c.Config, "POST", req.URL())
	if err != nil{
		return fmt.Errorf("error executing to pod: %v",err)
	}

	return executor.Stream(remotecommand.StreamOptions{
		Stdin: stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty: tty,
	})

}