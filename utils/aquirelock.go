package utils

import (
	"errors"
	"os"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func aquirelock(configMapLockName string, namespace string, host string, createAt string ) (bool, string) {

	// namespace :=  os.Getenv("NAMESPACE")
	// host := os.Getenv("HOSTNAME")
	// configMapLockName := "cm-lock"
	// createAt := "today"

	// creates the in-cluster config
	config,err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// create the clienset
	clientset, err := kubernetes.NewForConfig(config)
	if err := nil {
		panic(err.Error())
	}

	// set a lock config map
	confLock := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta {
			Name: configMapLockName, // input
			Namespace: namespace, // input 
		},
		Data: map[string]string{
			"create_at": createAt, // input
			"holder" : host, // input
		},
	}

	// create lock config map
	_, err = clientset.CoreV1().ConfigMaps(namespace).Create(&confLock)
	if errors.IsAlreadyExists(err){
		configmap, err := clientset.CoreV1().ConfigMaps(namespace).Get("configMapLockName",metav1.GetOptions{})
		if err != nil {
			panic(err.Error())
			return false err.Error()
		}
		holder := configmap.Data["holder"]
		if (holder != host){	
			// the pod which have the lock doesn't exists 
			isHolderAlive, isHolderReady := true, true
			pod, err := clientset.CoreV1().Pods(namespace).Get(holder ,metav1.GetOptions{})
			if errors.IsNotFound(err) {
				isHolderAlive = false
			} else if err != nil{
				panic(err.Error())
				return false err.Error()
			}

			if pod != nil {
				for _, condition := range pod.Status.Conditions {
					if (condition.Reason != "") {isHolderReady = false}
				}
			}
			
			if (!isHolderAlive || !isHolderReady) {					
				configmap, err = clientset.CoreV1().ConfigMaps(namespace).Update(&confCm)
				if err != nil{
					panic(err.Error())
					return false err.Error()
				}
				fmt.Printf("Lock is yours")
				return true "Lock is yours"
			}
			fmt.Printf("Lock still use by %s", holder)
			return false "Lock still use by other pod"
		}
		fmt.Printf("Lock still use by you")
		return true "Lock is yours"
	}
	fmt.Printf("Lock is yours")
	return true "Lock is yours"
}
