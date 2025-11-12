package gke

import (
	"context"
	"fmt"
	"time"

	container "cloud.google.com/go/container/apiv1"
	"cloud.google.com/go/container/apiv1/containerpb"
	"google.golang.org/api/option"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Client struct {
	containerClient *container.ClusterManagerClient
	k8sClient       kubernetes.Interface
	projectID       string
}

type ClusterConfig struct {
	Name      string
	Zone      string
	ProjectID string
}

type N8NInstance struct {
	Name                string
	Namespace           string
	Domain              string
	WorkerReplicas      int32
	MainCPURequest      string
	MainMemoryRequest   string
	WorkerCPURequest    string
	WorkerMemoryRequest string
	PostgresStorageSize string
	N8NStorageSize      string
	EncryptionKey       string
	DatabasePassword    string
}

func NewClient(projectID string, credentialsJSON []byte) (*Client, error) {
	ctx := context.Background()

	containerClient, err := container.NewClusterManagerClient(ctx, option.WithCredentialsJSON(credentialsJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create container client: %w", err)
	}

	return &Client{
		containerClient: containerClient,
		projectID:       projectID,
	}, nil
}

func (c *Client) CreateAutopilotCluster(ctx context.Context, config ClusterConfig) error {
	req := &containerpb.CreateClusterRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", config.ProjectID, config.Zone),
		Cluster: &containerpb.Cluster{
			Name:        config.Name,
			Description: "GKE Autopilot cluster for n8n instances",
			Autopilot: &containerpb.Autopilot{
				Enabled: true,
			},
			ReleaseChannel: &containerpb.ReleaseChannel{
				Channel: containerpb.ReleaseChannel_RAPID,
			},
			IpAllocationPolicy: &containerpb.IPAllocationPolicy{
				UseIpAliases: true,
			},
			NetworkConfig: &containerpb.NetworkConfig{
				EnableIntraNodeVisibility: true,
			},
		},
	}

	op, err := c.containerClient.CreateCluster(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	// Wait for operation to complete
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(30 * time.Second):
			getOp, err := c.containerClient.GetOperation(ctx, &containerpb.GetOperationRequest{
				Name: op.Name,
			})
			if err != nil {
				return fmt.Errorf("failed to get operation status: %w", err)
			}

			if getOp.Status == containerpb.Operation_DONE {
				if getOp.Error != nil {
					return fmt.Errorf("cluster creation failed: %s", getOp.Error.Message)
				}
				return nil
			}
		}
	}
}

func (c *Client) ConnectToCluster(ctx context.Context, clusterName, zone string) error {
	cluster, err := c.containerClient.GetCluster(ctx, &containerpb.GetClusterRequest{
		Name: fmt.Sprintf("projects/%s/locations/%s/clusters/%s", c.projectID, zone, clusterName),
	})
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	config := &rest.Config{
		Host: fmt.Sprintf("https://%s", cluster.Endpoint),
		TLSClientConfig: rest.TLSClientConfig{
			CAData: []byte(cluster.MasterAuth.ClusterCaCertificate),
		},
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	c.k8sClient = k8sClient
	return nil
}

func (c *Client) DeployN8NInstance(ctx context.Context, instance N8NInstance) error {
	if c.k8sClient == nil {
		return fmt.Errorf("kubernetes client not connected")
	}

	// Create namespace
	if err := c.createNamespace(ctx, instance.Namespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// Create secrets
	if err := c.createSecrets(ctx, instance); err != nil {
		return fmt.Errorf("failed to create secrets: %w", err)
	}

	// Create configmap
	if err := c.createConfigMap(ctx, instance); err != nil {
		return fmt.Errorf("failed to create configmap: %w", err)
	}

	// Create PVCs
	if err := c.createPVCs(ctx, instance); err != nil {
		return fmt.Errorf("failed to create PVCs: %w", err)
	}

	// Deploy PostgreSQL
	if err := c.deployPostgreSQL(ctx, instance); err != nil {
		return fmt.Errorf("failed to deploy PostgreSQL: %w", err)
	}

	// Deploy Redis
	if err := c.deployRedis(ctx, instance); err != nil {
		return fmt.Errorf("failed to deploy Redis: %w", err)
	}

	// Deploy n8n main
	if err := c.deployN8NMain(ctx, instance); err != nil {
		return fmt.Errorf("failed to deploy n8n main: %w", err)
	}

	// Deploy n8n workers
	if err := c.deployN8NWorkers(ctx, instance); err != nil {
		return fmt.Errorf("failed to deploy n8n workers: %w", err)
	}

	// Note: External access is handled by a shared Cloudflare Tunnel
	// DNS records should be created manually at Cloudflare panel:
	// - CNAME: {customer-domain} -> tunnel.example.com
	// - Service will be accessible via ClusterIP within the cluster

	return nil
}

func (c *Client) createNamespace(ctx context.Context, namespace string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"name": namespace,
			},
		},
	}

	_, err := c.k8sClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) createSecrets(ctx context.Context, instance N8NInstance) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "n8n-secrets",
			Namespace: instance.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"DB_POSTGRESDB_PASSWORD": []byte(instance.DatabasePassword),
			"N8N_ENCRYPTION_KEY":     []byte(instance.EncryptionKey),
		},
	}

	_, err := c.k8sClient.CoreV1().Secrets(instance.Namespace).Create(ctx, secret, metav1.CreateOptions{})
	return err
}

func (c *Client) createConfigMap(ctx context.Context, instance N8NInstance) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "n8n-config",
			Namespace: instance.Namespace,
		},
		Data: map[string]string{
			"N8N_HOST":               instance.Domain,
			"N8N_PORT":               "5678",
			"N8N_PROTOCOL":           "http",
			"WEBHOOK_URL":            fmt.Sprintf("http://%s", instance.Domain),
			"GENERIC_TIMEZONE":       "UTC",
			"N8N_LOG_LEVEL":          "info",
			"N8N_LOG_OUTPUT":         "console",
			"EXECUTIONS_MODE":        "queue",
			"QUEUE_BULL_REDIS_HOST":  "redis-service",
			"QUEUE_BULL_REDIS_PORT":  "6379",
			"QUEUE_BULL_REDIS_DB":    "0",
			"DB_TYPE":                "postgresdb",
			"DB_POSTGRESDB_HOST":     "postgres-service",
			"DB_POSTGRESDB_PORT":     "5432",
			"DB_POSTGRESDB_DATABASE": "n8n",
			"DB_POSTGRESDB_USER":     "n8n",
			"N8N_METRICS":            "true",
			"N8N_METRICS_PREFIX":     "n8n_",
			"N8N_SECURE_COOKIE":      "false",
			"N8N_PERSIST_SESSIONS":   "true",
		},
	}

	_, err := c.k8sClient.CoreV1().ConfigMaps(instance.Namespace).Create(ctx, cm, metav1.CreateOptions{})
	return err
}

func (c *Client) createPVCs(ctx context.Context, instance N8NInstance) error {
	// PostgreSQL PVC
	postgresPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "postgres-pvc",
			Namespace: instance.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(instance.PostgresStorageSize),
				},
			},
		},
	}

	// n8n data PVC
	n8nPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "n8n-data-pvc",
			Namespace: instance.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(instance.N8NStorageSize),
				},
			},
		},
	}

	if _, err := c.k8sClient.CoreV1().PersistentVolumeClaims(instance.Namespace).Create(ctx, postgresPVC, metav1.CreateOptions{}); err != nil {
		return err
	}

	if _, err := c.k8sClient.CoreV1().PersistentVolumeClaims(instance.Namespace).Create(ctx, n8nPVC, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

func (c *Client) deployPostgreSQL(ctx context.Context, instance N8NInstance) error {
	replicas := int32(1)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "postgres",
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"app": "postgres",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "postgres",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "postgres",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "postgres",
							Image: "postgres:15-alpine",
							Ports: []corev1.ContainerPort{
								{ContainerPort: 5432},
							},
							Env: []corev1.EnvVar{
								{Name: "POSTGRES_DB", Value: "n8n"},
								{Name: "POSTGRES_USER", Value: "n8n"},
								{
									Name: "POSTGRES_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: "n8n-secrets"},
											Key:                  "DB_POSTGRESDB_PASSWORD",
										},
									},
								},
								{Name: "PGDATA", Value: "/var/lib/postgresql/data/pgdata"},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "postgres-storage",
									MountPath: "/var/lib/postgresql/data",
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("256Mi"),
									corev1.ResourceCPU:    resource.MustParse("250m"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("512Mi"),
									corev1.ResourceCPU:    resource.MustParse("500m"),
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "postgres-storage",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "postgres-pvc",
								},
							},
						},
					},
				},
			},
		},
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "postgres-service",
			Namespace: instance.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "postgres",
			},
			Ports: []corev1.ServicePort{
				{
					Port:       5432,
					TargetPort: intstr.FromInt(5432),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	if _, err := c.k8sClient.AppsV1().Deployments(instance.Namespace).Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
		return err
	}

	if _, err := c.k8sClient.CoreV1().Services(instance.Namespace).Create(ctx, service, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

func (c *Client) deployRedis(ctx context.Context, instance N8NInstance) error {
	replicas := int32(1)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis",
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"app": "redis",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "redis",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "redis",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "redis",
							Image:   "redis:7-alpine",
							Command: []string{"redis-server"},
							Args:    []string{"--appendonly", "yes"},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 6379},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("128Mi"),
									corev1.ResourceCPU:    resource.MustParse("100m"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("256Mi"),
									corev1.ResourceCPU:    resource.MustParse("200m"),
								},
							},
						},
					},
				},
			},
		},
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-service",
			Namespace: instance.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "redis",
			},
			Ports: []corev1.ServicePort{
				{
					Port:       6379,
					TargetPort: intstr.FromInt(6379),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	if _, err := c.k8sClient.AppsV1().Deployments(instance.Namespace).Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
		return err
	}

	if _, err := c.k8sClient.CoreV1().Services(instance.Namespace).Create(ctx, service, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

func (c *Client) deployN8NMain(ctx context.Context, instance N8NInstance) error {
	replicas := int32(1)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "n8n-main",
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"app":       "n8n-main",
				"component": "main",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":       "n8n-main",
					"component": "main",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":       "n8n-main",
						"component": "main",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "n8n",
							Image: "n8nio/n8n:latest",
							Ports: []corev1.ContainerPort{
								{ContainerPort: 5678},
							},
							Env: []corev1.EnvVar{
								{
									Name: "N8N_ENCRYPTION_KEY",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: "n8n-secrets"},
											Key:                  "N8N_ENCRYPTION_KEY",
										},
									},
								},
								{
									Name: "DB_POSTGRESDB_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: "n8n-secrets"},
											Key:                  "DB_POSTGRESDB_PASSWORD",
										},
									},
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{Name: "n8n-config"},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "n8n-data",
									MountPath: "/home/node/.n8n",
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse(instance.MainMemoryRequest),
									corev1.ResourceCPU:    resource.MustParse(instance.MainCPURequest),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("1Gi"),
									corev1.ResourceCPU:    resource.MustParse("1000m"),
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "n8n-data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "n8n-data-pvc",
								},
							},
						},
					},
				},
			},
		},
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "n8n-service",
			Namespace: instance.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":       "n8n-main",
				"component": "main",
			},
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(5678),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	if _, err := c.k8sClient.AppsV1().Deployments(instance.Namespace).Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
		return err
	}

	if _, err := c.k8sClient.CoreV1().Services(instance.Namespace).Create(ctx, service, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

func (c *Client) deployN8NWorkers(ctx context.Context, instance N8NInstance) error {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "n8n-worker",
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"app":       "n8n-worker",
				"component": "worker",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &instance.WorkerReplicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":       "n8n-worker",
					"component": "worker",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":       "n8n-worker",
						"component": "worker",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "n8n-worker",
							Image:   "n8nio/n8n:latest",
							Command: []string{"n8n", "worker"},
							Env: []corev1.EnvVar{
								{
									Name: "N8N_ENCRYPTION_KEY",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: "n8n-secrets"},
											Key:                  "N8N_ENCRYPTION_KEY",
										},
									},
								},
								{
									Name: "DB_POSTGRESDB_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: "n8n-secrets"},
											Key:                  "DB_POSTGRESDB_PASSWORD",
										},
									},
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{Name: "n8n-config"},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse(instance.WorkerMemoryRequest),
									corev1.ResourceCPU:    resource.MustParse(instance.WorkerCPURequest),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("512Mi"),
									corev1.ResourceCPU:    resource.MustParse("500m"),
								},
							},
						},
					},
				},
			},
		},
	}

	if _, err := c.k8sClient.AppsV1().Deployments(instance.Namespace).Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

func (c *Client) ScaleWorkers(ctx context.Context, namespace string, replicas int32) error {
	deployment, err := c.k8sClient.AppsV1().Deployments(namespace).Get(ctx, "n8n-worker", metav1.GetOptions{})
	if err != nil {
		return err
	}

	deployment.Spec.Replicas = &replicas
	_, err = c.k8sClient.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	return err
}

func (c *Client) DeleteN8NInstance(ctx context.Context, namespace string) error {
	if err := c.k8sClient.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func (c *Client) GetInstanceStatus(ctx context.Context, namespace string) (map[string]interface{}, error) {
	status := make(map[string]interface{})

	// Get deployments status
	deployments, err := c.k8sClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	depStatus := make(map[string]map[string]interface{})
	for _, dep := range deployments.Items {
		depStatus[dep.Name] = map[string]interface{}{
			"replicas":          dep.Status.Replicas,
			"readyReplicas":     dep.Status.ReadyReplicas,
			"availableReplicas": dep.Status.AvailableReplicas,
		}
	}
	status["deployments"] = depStatus

	// Get services status
	services, err := c.k8sClient.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	svcStatus := make([]string, 0)
	for _, svc := range services.Items {
		svcStatus = append(svcStatus, svc.Name)
	}
	status["services"] = svcStatus

	return status, nil
}

func (c *Client) Close() error {
	if c.containerClient != nil {
		return c.containerClient.Close()
	}
	return nil
}
