package secret

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
)

// @see https://github.com/kubernetes/kubectl/blob/c1d804c03fefc4de986592e9db203a306467e65d/pkg/generate/versioned/secret_for_docker_registry.go
// @see https://github.com/kubernetes/kubernetes/blob/b2ecd1b3a3192fbbe2b9e348e095326f51dc43dd/pkg/apis/core/types.go#L4938-L4961
// @see https://github.com/kubernetes/client-go/blob/master/kubernetes/typed/core/v1/secret.go
// @see https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/cri-api/pkg/errors/errors.go

// DockerConfigJSON represents a local docker auth config file for pulling images.
type DockerConfigJSON struct {
	Auths DockerConfig `json:"auths" datapolicy:"token"`
}

// DockerConfig represents the config file used by the docker CLI.
// This config that represents the credentials that should be used
// when pulling images from specific image repositories.
type DockerConfig map[string]DockerConfigEntry

// DockerConfigEntry is a entry for a config of docker registry
type DockerConfigEntry struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty" datapolicy:"password"`
	Email    string `json:"email,omitempty"`
	Auth     string `json:"auth,omitempty" datapolicy:"token"`
}

// ImgPullSecretClient is a client for Kubernetes APIs
type ImgPullSecretClient struct {
	set      kubernetes.Interface
	recorder record.EventRecorder
}

var (
	creOpts = metav1.CreateOptions{}
	delOpts = metav1.DeleteOptions{}
)

// NewImgPullSecretClient is a constructor
func NewImgPullSecretClient(set kubernetes.Interface, rec record.EventRecorder) *ImgPullSecretClient {
	return &ImgPullSecretClient{set: set, recorder: rec}
}

// DeleteSecret is
func (c *ImgPullSecretClient) DeleteSecret(secret *corev1.Secret) error {
	err := c.set.CoreV1().Secrets(secret.Namespace).Delete(context.TODO(), secret.Name, delOpts)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	c.recorder.Eventf(secret, corev1.EventTypeNormal, "SuccessfulDelete", "Deleted secret %s/%s", secret.Namespace, secret.Name)
	klog.V(4).Infof("Deleted secret %s/%s successfully", secret.Namespace, secret.Name)
	return nil
}

// CreateSecret is
func (c *ImgPullSecretClient) CreateSecret(name, server, user, password, email, namespace string) error {
	secret := corev1.Secret{}
	secret.Name = name
	secret.Namespace = namespace
	secret.Type = corev1.SecretTypeDockerConfigJson
	secret.Data = map[string][]byte{}

	data, err := encodeSecretData(server, user, password, email)
	if err != nil {
		return err
	}

	secret.Data[corev1.DockerConfigJsonKey] = data

	if _, err = c.set.CoreV1().Secrets(namespace).Create(context.TODO(), &secret, creOpts); err != nil {
		return err
	}

	c.recorder.Eventf(&secret, corev1.EventTypeNormal, "SuccessfulCreate", "Created secret %s/%s", namespace, name)
	klog.V(4).Infof("Created secret %s/%s successfully", namespace, name)
	return nil
}

func encodeSecretData(server, user, password, email string) ([]byte, error) {
	field := fmt.Sprintf("%s:%s", user, password)
	auth := base64.StdEncoding.EncodeToString([]byte(field))

	entry := DockerConfigEntry{
		Username: user,
		Password: password,
		Email:    email,
		Auth:     auth,
	}

	body := DockerConfigJSON{
		Auths: DockerConfig{server: entry},
	}

	return json.Marshal(body)
}
