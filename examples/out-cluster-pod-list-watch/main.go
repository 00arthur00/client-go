package main

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"os/signal"
	"path/filepath"
)

func Must(e interface{}) {
	if e != nil {
		panic(e)
	}
}

func InitClientSet() (*kubernetes.Clientset, error) {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(restConfig)
}

func InitListerWatcher(clientSet *kubernetes.Clientset, resource, namespace string, fieldSelector fields.Selector) cache.ListerWatcher {
	restClient := clientSet.CoreV1().RESTClient()
	return cache.NewListWatchFromClient(restClient, resource, namespace, fieldSelector)
}

func main() {
	clientSet, err := InitClientSet()
	if err != nil {
		panic(err)
	}

	// 什么常量
	resource := "pods"
	namespace := "default"

	podListerWatcher := InitListerWatcher(clientSet, resource, namespace, fields.Everything())

	// 1. list操作
	listObj, err := podListerWatcher.List(metav1.ListOptions{})

	// meta 包封装了一些处理 runtime.Object 对象的方法，屏蔽了反射和类型转换的过程，
	// 提取出的 items 类型为 []runtime.Object
	items, err := meta.ExtractList(listObj)
	if err != nil {
		Must(err)
	}
	fmt.Println("list result:")
	for _, item := range items {
		pod, ok := item.(*v1.Pod)
		if !ok {
			return
		}
		fmt.Printf("namespace: %s, resource name:%s\n", pod.Namespace, pod.Name)
	}

	// 2. watch 操作
	listMetaInterface, err := meta.ListAccessor(listObj)
	if err != nil {
		Must(err)
	}
	resourceVersion := listMetaInterface.GetResourceVersion()

	watchObj, err := podListerWatcher.Watch(metav1.ListOptions{
		ResourceVersion: resourceVersion,
	})

	// 接收信号
	stopCh := make(chan os.Signal)
	signal.Notify(stopCh, os.Interrupt)
	fmt.Println("Start watching...")
	for {
		select {
		case <-stopCh:
			fmt.Println("exit")
			return
		case event, ok := <-watchObj.ResultChan():
			if !ok {
				fmt.Println("Broken channel")
				break
			}
			podInfo, ok := event.Object.(*v1.Pod)
			if !ok {
				return
			}
			fmt.Printf("eventType: %s, watch obj:%+v\n", event.Type, podInfo.Name)
		}
	}
}
