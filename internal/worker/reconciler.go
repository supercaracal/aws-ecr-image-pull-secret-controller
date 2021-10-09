package worker

import (
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

type imgPullSecretInfo struct {
	name               string
	email              string
	awsEndpointURL     string
	awsAccountID       string
	awsRegion          string
	awsAccessKeyID     string
	awsSecretAccessKey string
}

const (
	expirationPeriod         = 8 * time.Hour
	awsAccessKeyIDName       = "AWS_ACCESS_KEY_ID"
	awsSecretAccessKeyName   = "AWS_SECRET_ACCESS_KEY"
	domainPrefix             = "supercaracal.example.com"
	labelKey                 = domainPrefix + "/used-by"
	annotationPrefix         = domainPrefix + "/aws-ecr-image-pull-secret"
	annotationName           = annotationPrefix + ".name"
	annotationEmail          = annotationPrefix + ".email"
	annotationAWSEndpointURL = annotationPrefix + ".aws_endpoint_url"
	annotationAWSAccountID   = annotationPrefix + ".aws_account_id"
	annotationAWSRegion      = annotationPrefix + ".aws_region"
	managerName              = "aws-ecr-image-pull-secret-controller"
)

// NewReconciler is
func NewReconciler(cli kubernetes.Interface, list corelisterv1.SecretLister, rec record.EventRecorder) *Reconciler {
	return &Reconciler{client: secrets.NewImgPullSecretClient(cli, rec), lister: list}
}

// Run is
func (r *Reconciler) Run() {
	selector, err := labels.ValidatedSelectorFromSet(labels.Set{labelKey: managerName})
	if err != nil {
		utilruntime.HandleError(err)
		return
	}

	loginSecrets, err := r.lister.List(selector)
	if err != nil {
		if !kubeerrors.IsNotFound(err) {
			utilruntime.HandleError(err)
		}
		return
	}

	for _, loginSecret := range loginSecrets {
		if err := r.renewImgPullSecretIfNeeded(loginSecret); err != nil {
			utilruntime.HandleError(fmt.Errorf("%s/%s: %w", loginSecret.Namespace, loginSecret.Name, err))
			continue
		}
	}
}

func (r *Reconciler) renewImgPullSecretIfNeeded(loginSecret *corev1.Secret) error {
	info, err := extractImgPullSecretInfo(loginSecret)
	if err != nil {
		return err
	}

	imgPullSecret, err := r.lister.Secrets(loginSecret.Namespace).Get(info.name)
	if err != nil && !kubeerrors.IsNotFound(err) {
		return err
	}
	if err == nil && imgPullSecret != nil {
		baseTime := metav1.NewTime(time.Now().Add(-expirationPeriod))
		if baseTime.Before(&imgPullSecret.CreationTimestamp) {
			return nil
		}
	}

	ecrCli, err := registries.NewECRClient(info.awsEndpointURL, info.awsRegion, info.awsAccessKeyID, info.awsSecretAccessKey)
	if err != nil {
		return err
	}

	credential, err := ecrCli.Login(info.awsAccountID, info.email)
	if err != nil {
		return err
	}

	if imgPullSecret != nil {
		if err := r.client.DeleteSecret(imgPullSecret); err != nil {
			return err
		}
	}

	if err := r.client.CreateSecret(
		info.name,
		credential.Server,
		credential.UserName,
		credential.Password,
		credential.Email,
		loginSecret,
	); err != nil {

		return err
	}

	return nil
}

func extractImgPullSecretInfo(loginSecret *corev1.Secret) (*imgPullSecretInfo, error) {
	if loginSecret.Data[awsAccessKeyIDName] == nil || loginSecret.Data[awsSecretAccessKeyName] == nil {
		return nil, fmt.Errorf("%s and %s are required", awsAccessKeyIDName, awsSecretAccessKeyName)
	}

	return &imgPullSecretInfo{
		name:               loginSecret.Annotations[annotationName],
		email:              loginSecret.Annotations[annotationEmail],
		awsEndpointURL:     loginSecret.Annotations[annotationAWSEndpointURL],
		awsAccountID:       loginSecret.Annotations[annotationAWSAccountID],
		awsRegion:          loginSecret.Annotations[annotationAWSRegion],
		awsAccessKeyID:     string(loginSecret.Data[awsAccessKeyIDName]),
		awsSecretAccessKey: string(loginSecret.Data[awsSecretAccessKeyName]),
	}, nil
}
