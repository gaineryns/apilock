package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/gorilla/mux"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type param_struct struct {
	Host        string `json:"host"`
	Namespace   string `json:"namespace" `
	CreatedAt   string `json:-`
	ProjectName string `json:"projectname"`
	LockName    string `json:"lockname"`
}

var createdAt = time.Now().String()

var clientset = configCluster()

func InitializeRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	router.Methods("POST").Path("/aquirelock").Name("aquirelock").HandlerFunc(aquireLock)
	router.Methods("DELETE").Path("/releaselock").Name("releaselock").HandlerFunc(releaseLock)
	router.Methods("GET").Path("/healthz").Name("healthz").HandlerFunc(healthz)
	return router
}
func healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("ok"))

}
func aquireLock(w http.ResponseWriter, r *http.Request) {

	var t param_struct
	w.Header().Set("Content-type", "application/json;charset=UTF-8")

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&t)
	if err != nil {
		panic(err)
	}

	if t.Host == "" || t.ProjectName == "" || t.LockName == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("one or many variable is not set")
		return
	}
	if t.Namespace == "" {
		t.Namespace = "default"
	}
	configMapLockName := t.LockName + "-" + t.ProjectName
	value := getlock(configMapLockName, t.Namespace, t.Host, t.ProjectName)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(value)
	return

}

func releaseLock(w http.ResponseWriter, r *http.Request) {

	var t param_struct
	w.Header().Set("Content-type", "application/json;charset=UTF-8")

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&t)
	if err != nil {
		panic(err)
	}

	if t.Host == "" || t.ProjectName == "" || t.LockName == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("one or many variable is not set")
		return
	}
	if t.Namespace == "" {
		t.Namespace = "default"
	}
	configMapLockName := t.LockName + "-" + t.ProjectName
	value := deletelock(configMapLockName, t.Namespace, t.Host, t.ProjectName)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(value)
	return

}

func getlock(configMapLockName string, namespace string, host string, projectName string) bool {

	// set a lock config map
	confLock := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapLockName, // input
			Namespace: namespace,         // input
		},
		Data: map[string]string{
			"CreatedAt":   createdAt,
			"LockHolder":  host,
			"ProjectName": projectName,
		},
	}

	// create lock config map
	_, err := clientset.CoreV1().ConfigMaps(namespace).Create(&confLock)
	if errors.IsAlreadyExists(err) {

		// keep lock if user if the lock holder
		// false = here we make a update not a delete
		return updatDeleteByMe(configMapLockName, host, namespace, false, projectName)

	}
	fmt.Println("Lock by you")
	return true
}

func deletelock(configMapLockName string, namespace string, host string, projectname string) bool {

	return updatDeleteByMe(configMapLockName, host, namespace, true, projectname)

}

func configCluster() *kubernetes.Clientset {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
		return nil
	}
	// create the clienset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
		return nil
	}
	return clientset
}

func updatDeleteByMe(configMapLockName string, applicant string, namespace string, deleteConfigmap bool, projectname string) bool {

	// set a lock config map
	confLock := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapLockName, // input
			Namespace: namespace,         // input
		},
		Data: map[string]string{
			"CreatedAt":   createdAt,
			"LockHolder":  applicant,
			"ProjectName": projectname,
		},
	}

	configmap, err := clientset.CoreV1().ConfigMaps(namespace).Get(configMapLockName, metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
		return false
	}
	lockHolder := configmap.Data["LockHolder"]
	if lockHolder == "" {
		configmap, err = clientset.CoreV1().ConfigMaps(namespace).Update(&confLock)
		if err != nil {
			panic(err.Error())
			return false
		}
		fmt.Printf("Lock is yours 1")
		return true
	}

	if lockHolder != applicant {
		// the pod which have the lock doesn't exists
		isHolderAlive, isHolderReady := true, true
		pod, err := clientset.CoreV1().Pods(namespace).Get(lockHolder, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			isHolderAlive = false
		} else if err != nil {
			panic(err.Error())
			return false
		}

		if pod != nil {
			for _, condition := range pod.Status.Conditions {
				if condition.Reason != "" {
					isHolderReady = false
				}
			}
		}

		if !isHolderAlive || !isHolderReady {
			if deleteConfigmap {
				err := clientset.CoreV1().ConfigMaps(namespace).Delete(configMapLockName, nil)
				if err != nil {
					panic(err.Error())
					return false
				}
				fmt.Println("release lock")
			} else {
				_, err := clientset.CoreV1().ConfigMaps(namespace).Update(&confLock)
				if err != nil {
					panic(err.Error())
					return false
				}
				fmt.Println("Lock by you")
			}
			return true
		}
		fmt.Printf("Lock  by %s", applicant)
		return false
	} else {
		if deleteConfigmap {
			err := clientset.CoreV1().ConfigMaps(namespace).Delete(configMapLockName, nil)
			if err != nil {
				panic(err.Error())
				return false
			}
		} else {
			fmt.Println("Lock by you")
		}
		return true
	}

}

func main() {
	router := InitializeRouter()
	log.Fatal(http.ListenAndServe(":8080", router))
}
