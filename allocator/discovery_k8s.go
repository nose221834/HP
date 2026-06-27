package main

import (
	"context"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	terminalNamespace = "terminal"
	terminalLabel     = "app=terminal"
	terminalPort      = 8080
)

// K8sDiscovery は k8s API から terminal Pod の一覧を取得する
// 本番時（k8s）での使用を想定

type K8sDiscovery struct {
	client *kubernetes.Clientset
}

func NewK8sDiscovery() (*K8sDiscovery, error) {
	client, err := newK8sClient()
	if err != nil {
		return nil, err
	}
	return &K8sDiscovery{client: client}, nil
}

func (k *K8sDiscovery) ListPodURLs(ctx context.Context) ([]string, error) {
	pods, err := k.client.CoreV1().Pods(terminalNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: terminalLabel,
	})
	if err != nil {
		return nil, fmt.Errorf("list pods error: %w", err)
	}

	var urls []string
	for _, pod := range pods.Items {
		// Running 状態のPodのみ対象
		if pod.Status.Phase != "Running" {
			continue
		}
		// HeadlessServiceを使ってPod名でDNS解決する
		// 例: http://terminal-0.terminal-svc.terminal.svc.cluster.local:8080
		url := fmt.Sprintf("http://%s.terminal-svc.%s.svc.cluster.local:%d",
			pod.Name,
			terminalNamespace,
			terminalPort,
		)
		urls = append(urls, url)
	}
	return urls, nil
}

// ─── k8s クライアントの初期化 ─────────────────────────────────────
// Pod内（本番）: InClusterConfig を使う
// ローカル（開発）: KUBECONFIG を使う

func newK8sClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = os.Getenv("HOME") + "/.kube/config"
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("k8s config error: %w", err)
		}
	}
	return kubernetes.NewForConfig(config)
}
