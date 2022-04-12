package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientSet := getClientSet()

	// create a namespace named "foo" and delete it when main exits
	nsFoo := createNamespace(ctx, clientSet, "foo")
	defer func() {
		deleteNamespace(ctx, clientSet, nsFoo)
	}()

	// create an nginx deployment named "hello-world" in the nsFoo namespace
	deployNginx(ctx, clientSet, nsFoo, "hello-world")
	fmt.Printf("You can now see your running service: http://localhost:8080/hello\n\n")

	// listen to pod logs from namespace foo
	listenToPodLogs(ctx, clientSet, nsFoo, "hello-world")

	// wait for ctrl-c to exit the program
	waitForExitSignal()
}

func createNamespace(ctx context.Context, clientSet *kubernetes.Clientset, name string) *corev1.Namespace {
	fmt.Printf("Creating namespace %q.\n\n", name)
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	ns, err := clientSet.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	panicIfError(err)
	return ns
}

func deleteNamespace(ctx context.Context, clientSet *kubernetes.Clientset, ns *corev1.Namespace) {
	fmt.Printf("\n\nDeleting namespace %q.\n", ns.Name)
	panicIfError(clientSet.CoreV1().Namespaces().Delete(ctx, ns.Name, metav1.DeleteOptions{}))
}

func deployNginx(ctx context.Context, clientSet *kubernetes.Clientset, ns *corev1.Namespace, name string) {
	deployment := createNginxDeployment(ctx, clientSet, ns, name)
	waitForReadyReplicas(ctx, clientSet, deployment)
	createNginxService(ctx, clientSet, ns, name)
	createNginxIngress(ctx, clientSet, ns, name)
}

func createNginxDeployment(ctx context.Context, clientSet *kubernetes.Clientset, ns *corev1.Namespace, name string) *appv1.Deployment {
	var (
		matchLabel = map[string]string{"app": "nginx"}
		objMeta    = metav1.ObjectMeta{
			Name:      name,
			Namespace: ns.Name,
			Labels:    matchLabel,
		}
	)

	deployment := &appv1.Deployment{
		ObjectMeta: objMeta,
		Spec: appv1.DeploymentSpec{
			Replicas: to.Int32Ptr(2),
			Selector: &metav1.LabelSelector{MatchLabels: matchLabel},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: matchLabel,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: "nginxdemos/hello:latest",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}
	deployment, err := clientSet.AppsV1().Deployments(ns.Name).Create(ctx, deployment, metav1.CreateOptions{})
	panicIfError(err)
	return deployment
}

func waitForReadyReplicas(ctx context.Context, clientSet *kubernetes.Clientset, deployment *appv1.Deployment) {
	fmt.Printf("Waiting for ready replicas in deployment %q\n", deployment.Name)
	for {
		expectedReplicas := *deployment.Spec.Replicas
		readyReplicas := getReadyReplicasForDeployment(ctx, clientSet, deployment)
		if readyReplicas == expectedReplicas {
			fmt.Printf("replicas are ready!\n\n")
			break
		}

		fmt.Printf("replicas are not ready yet. %d/%d\n", readyReplicas, expectedReplicas)
		time.Sleep(1 * time.Second)
	}
}

func getReadyReplicasForDeployment(ctx context.Context, clientSet *kubernetes.Clientset, deployment *appv1.Deployment) int32 {
	dep, err := clientSet.AppsV1().Deployments(deployment.Namespace).Get(ctx, deployment.Name, metav1.GetOptions{})
	panicIfError(err)

	return dep.Status.ReadyReplicas
}

func createNginxService(ctx context.Context, clientSet *kubernetes.Clientset, ns *corev1.Namespace, name string) {
	var (
		matchLabel = map[string]string{"app": "nginx"}
		objMeta    = metav1.ObjectMeta{
			Name:      name,
			Namespace: ns.Name,
			Labels:    matchLabel,
		}
	)

	service := &corev1.Service{
		ObjectMeta: objMeta,
		Spec: corev1.ServiceSpec{
			Selector: matchLabel,
			Ports: []corev1.ServicePort{
				{
					Port:     80,
					Protocol: corev1.ProtocolTCP,
					Name:     "http",
				},
			},
		},
	}
	service, err := clientSet.CoreV1().Services(ns.Name).Create(ctx, service, metav1.CreateOptions{})
	panicIfError(err)
}

func createNginxIngress(ctx context.Context, clientSet *kubernetes.Clientset, ns *corev1.Namespace, name string) {
	var (
		prefix  = netv1.PathTypePrefix
		objMeta = metav1.ObjectMeta{
			Name:      name,
			Namespace: ns.Name,
		}
		ingressPath = netv1.HTTPIngressPath{
			PathType: &prefix,
			Path:     "/hello",
			Backend: netv1.IngressBackend{
				Service: &netv1.IngressServiceBackend{
					Name: name,
					Port: netv1.ServiceBackendPort{
						Name: "http",
					},
				},
			},
		}
	)

	ingress := &netv1.Ingress{
		ObjectMeta: objMeta,
		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{
				{
					IngressRuleValue: netv1.IngressRuleValue{
						HTTP: &netv1.HTTPIngressRuleValue{
							Paths: []netv1.HTTPIngressPath{ingressPath},
						},
					},
				},
			},
		},
	}
	ingress, err := clientSet.NetworkingV1().Ingresses(ns.Name).Create(ctx, ingress, metav1.CreateOptions{})
	panicIfError(err)
}

func listenToPodLogs(ctx context.Context, clientSet *kubernetes.Clientset, ns *corev1.Namespace, containerName string) {
	// list all the pods in namespace foo
	podList := listPods(ctx, clientSet, ns)

	for _, pod := range podList.Items {
		podName := pod.Name
		go func() {
			opts := &corev1.PodLogOptions{
				Container: containerName,
				Follow:    true,
			}
			podLogs, err := clientSet.CoreV1().Pods(ns.Name).GetLogs(podName, opts).Stream(ctx)
			panicIfError(err)

			_, _ = os.Stdout.ReadFrom(podLogs)
		}()
	}
}

func listPods(ctx context.Context, clientSet *kubernetes.Clientset, ns *corev1.Namespace) *corev1.PodList {
	podList, err := clientSet.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{})
	panicIfError(err)

	fmt.Printf("Listing pods in %q namespace.\n", ns.Name)
	for _, pod := range podList.Items {
		fmt.Printf("# Pod: \n## namespace/name: %q\n## spec.containers[0].name: %q\n## spec.containers[0].image: %q\n", path.Join(pod.Namespace, pod.Name), pod.Spec.Containers[0].Name, pod.Spec.Containers[0].Image)
	}
	fmt.Printf("\n\n")
	return podList
}

func waitForExitSignal() {
	fmt.Printf("Type ctrl-c to exit\n\n")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
}

func getClientSet() *kubernetes.Clientset {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	panicIfError(err)

	// create the clientSet
	cs, err := kubernetes.NewForConfig(config)
	panicIfError(err)
	return cs
}

func panicIfError(err error) {
	if err != nil {
		panic(err.Error())
	}
}
