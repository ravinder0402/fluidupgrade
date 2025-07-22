package secret

import (
	"context"
	"log"
	"os"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	Namespace    string
	clientset    *kubernetes.Clientset
	secretClient clientCoreV1.SecretInterface
)

func IsRegistrySecretCreated() bool {
	_, err := secretClient.Get(context.TODO(), RegistrySecretName, metav1.GetOptions{})
	return (err == nil)
}

func DeleteRegistrySecret() error {
	return secretClient.Delete(context.TODO(), RegistrySecretName, metav1.DeleteOptions{})
}

func EnsureRegistrySecret(config []byte) error {
	_, err := secretClient.Get(context.TODO(), RegistrySecretName, metav1.GetOptions{})
	if err == nil {
		log.Println("Registry Secret already exists")
		return nil
	}
	secretData := make(map[string][]byte)
	secretData["config.json"] = config
	secret := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: RegistrySecretName,
		},
		Data: secretData,
	}
	_, err = secretClient.Create(context.TODO(), secret, metav1.CreateOptions{})
	if err == nil {
		log.Println("Successfully created Registry Secret")
	}
	return err
}

func init() {
	// get namespace of the pod in which it is running
	bytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		log.Fatalln("failed to get pod namespace ", err)
	}
	Namespace = string(bytes)

	// load kube client by config for compass controller
	clientset, err = kubernetes.NewForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		log.Fatalln("failed to load k8s config for controller ", err)
	}

	secretClient = clientset.CoreV1().Secrets(Namespace)
}
