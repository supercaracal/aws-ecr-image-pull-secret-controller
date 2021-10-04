package worker

import (
	"encoding/base64"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"

	registries "github.com/supercaracal/aws-ecr-image-pull-secret-controller/internal/registry"
	secrets "github.com/supercaracal/aws-ecr-image-pull-secret-controller/internal/secret"
)

// Reconciler is
type Reconciler struct {
	client *secrets.ImgPullSecretClient
	lister corelisterv1.SecretLister
}

const (
	expirationPeriod = 8 * time.Hour
)

// NewReconciler is
func NewReconciler(cli kubernetes.Interface, list corelisterv1.SecretLister, rec record.EventRecorder) *Reconciler {
	return &Reconciler{client: secrets.NewImgPullSecretClient(cli, rec), lister: list}
}

// Run is
func (r *Reconciler) Run() {
	labelSet := labels.Set{
		"supercaracal.example.com/used-by": "aws-ecr-image-pull-secret-controller",
	}

	selector, err := labels.ValidatedSelectorFromSet(labelSet)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}

	targetSecrets, err := r.lister.List(selector)
	if err != nil {
		if !kubeerrors.IsNotFound(err) {
			utilruntime.HandleError(err)
		}
		return
	}

	baseTime := metav1.NewTime(time.Now().Add(-expirationPeriod))

	for _, secret := range targetSecrets {
		if baseTime.Before(&secret.CreationTimestamp) {
			continue
		}

		if err := r.renewFrom(secret); err != nil {
			utilruntime.HandleError(fmt.Errorf("%s/%s: %w", secret.Namespace, secret.Name, err))
			continue
		}

	}
}

func (r *Reconciler) renewFrom(secret *corev1.Secret) error {
	if secret.Data["AWS_ACCESS_KEY_ID"] == nil || secret.Data["AWS_SECRET_ACCESS_KEY"] == nil {
		return fmt.Errorf("AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY are required")
	}

	accessKeyID, err := decodeBase64(secret.Data["AWS_ACCESS_KEY_ID"])
	if err != nil {
		return err
	}

	secretAccessKey, err := decodeBase64(secret.Data["AWS_SECRET_ACCESS_KEY"])
	if err != nil {
		return err
	}

	a := secret.Annotations

	ecrCli, err := registries.NewECRClient(
		a["supercaracal.example.com/aws-ecr-image-pull-secret.aws_endpoint_url"],
		a["supercaracal.example.com/aws-ecr-image-pull-secret.aws_region"],
		accessKeyID,
		secretAccessKey,
	)
	if err != nil {
		return err
	}

	credential, err := ecrCli.Login(
		a["supercaracal.example.com/aws-ecr-image-pull-secret.aws_account_id"],
		a["supercaracal.example.com/aws-ecr-image-pull-secret.email"],
	)
	if err != nil {
		return err
	}

	if err = r.client.UpdateSecret(
		a["supercaracal.example.com/aws_ecr-image-pull-secret.name"],
		credential.Server,
		credential.UserName,
		credential.Password,
		credential.Email,
		secret.Namespace,
	); err != nil {
		return err
	}

	return nil
}

func decodeBase64(src []byte) (string, error) {
	size := base64.StdEncoding.DecodedLen(len(src))
	dst := make([]byte, size)
	if n, err := base64.StdEncoding.Decode(dst, src); err != nil {
		return "", err
	} else if n != size {
		return "", fmt.Errorf("base64 decoding size: want=%d, got=%d", size, n)
	}

	return string(dst), nil
}
