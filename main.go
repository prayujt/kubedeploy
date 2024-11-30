package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func handler(clientset *kubernetes.Clientset) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			deployToken := os.Getenv("DEPLOY_TOKEN")
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer "+deployToken {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Unable to read body", http.StatusBadRequest)
				return
			}

			var requestData struct {
				Namespace      string `json:"namespace"`
				DeploymentName string `json:"deploymentName"`
			}

			err = json.Unmarshal(body, &requestData)
			if err != nil {
				http.Error(w, "Invalid JSON body", http.StatusBadRequest)
				return
			}
			err = restartDeployment(clientset, requestData.Namespace, requestData.DeploymentName)
			if err != nil {
				log.Printf("Failed to restart deployment: %v", err)
				http.Error(w, "Failed to restart deployment", http.StatusInternalServerError)
				return
			}

			fmt.Fprintf(w, "Received namespace: %s, deployment name: %s. Deployment restarted", requestData.Namespace, requestData.DeploymentName)
		} else {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		}
	}
}

func restartDeployment(clientset *kubernetes.Clientset, namespace, deploymentName string) error {
	deploymentsClient := clientset.AppsV1().Deployments(namespace)

	deployment, err := deploymentsClient.Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		log.Printf("Failed to get deployment: %v", err)
		return err
	}

	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	_, err = deploymentsClient.Update(context.TODO(), deployment, metav1.UpdateOptions{})
	return err
}

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	http.HandleFunc("/", handler(clientset))
	log.Println("Starting server on :8080")
	http.ListenAndServe(":8080", nil)
}
