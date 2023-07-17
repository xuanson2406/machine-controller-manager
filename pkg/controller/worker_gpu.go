// Handle request for worker group enable GPU
// Package controller is used to provide the core functionalities of machine-controller-manager
package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/flock"
	"github.com/pkg/errors"
	"github.com/xuanson2406/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/action"
	hchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/strvals"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

const (
	GPUChartAlreadyInstalled = 0
	GPUChartNotInstalled     = 1
	InstallPrometheusStack   = 2
	InstallPrometheusAdapter = 3
)

// ProviderSpec is the spec to be used while parsing the calls.
type ProviderSpec struct {
	ClusterName        string `json:"cluster,omitempty"`
	WorkerPoolName     string `json:"workerPoolName,omitempty"`
	NetworkName        string `json:"networkName,omitempty"`
	IPMode             string `json:"ipMode,omitempty"`
	NetworkAdapterType string `json:"networkAdapterType,omitempty"`
	IsPrimary          bool   `json:"isPrimary,omitempty"`
	VappTemplate       string `json:"vAppTemplateName,omitempty"`
	Catalog            string `json:"catalogName,omitempty"`
	VCPU               int    `json:"vCPU,omitempty"`
	VGPU               string `json:"vGPU,omitempty"`
	VGpuId             string `json:"vGpuId,omitempty"`
	RAM                int    `json:"ram,omitempty"`
	Disk               int    `json:"disk,omitempty"`
	PublicKey          string `json:"publicKey,omitempty"`
	Storagepolicy      string `json:"storagePolicy,omitempty"`
	ApiUrl             string `json:"apiUrl,omitempty"`
	VpcID              string `json:"vpcID,omitempty"`
	BackendPortalToken string `json:"backendPortalToken,omitempty"`
	VpcToken           string `json:"vpcToken,omitempty"`
	Zone               string `json:"zone,omitempty"` // this field will use in future version
}

func (c *controller) CheckGPUWorkerGroup(ctx context.Context, machinedeployment *v1alpha1.MachineDeployment) (bool, error) {

	machineClassInterface := c.controlMachineClient.MachineClasses(c.namespace)
	machineClass, err := machineClassInterface.Get(ctx, machinedeployment.Spec.Template.Spec.Class.Name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("MachineClass %s/%s not found. Skipping. %v", c.namespace, machineClass.Name, err)
		return false, err
	}
	providerSpec, err := DecodeProviderSpecFromMachineClass(machineClass)
	if err != nil {
		return false, err
	}
	if providerSpec.VGPU == "gpu" {
		klog.V(3).Infof("Worker group %s is enable GPU - waiting for install GPU chart to shoot", machinedeployment.Name)
		return true, nil
	}
	return false, nil
}

func (c *controller) InstallChartForShoot(ctx context.Context, machineDeployment *v1alpha1.MachineDeployment) error {
	var (
		url       = "https://registry.fke.fptcloud.com/chartrepo/xplat-fke"
		repoName  = "xplat-fke"
		shootName = machineDeployment.Namespace
		strategy  string
	)
	EnableGPU, err := c.CheckGPUWorkerGroup(ctx, machineDeployment)
	if err != nil {
		return err
	}
	if !EnableGPU {
		klog.V(3).Infof("Shoot cluster %s is disable GPU - skipping install helm chart to shoot\n", shootName)
		return nil
	}
	kubeconfigFile := os.Getenv("HOME") + "/kubeconfig/" + shootName
	err = os.MkdirAll(filepath.Dir(kubeconfigFile), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("Could not create dir to store chart repository: [%v]", err)
	}
	kubeconfigSecret, err := c.controlCoreClient.CoreV1().Secrets(c.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, secret := range kubeconfigSecret.Items {
		if strings.Contains(secret.Name, "user-kubeconfig") {
			err = ioutil.WriteFile(kubeconfigFile, []byte(string(secret.Data["kubeconfig"])), os.ModePerm)
			if err != nil {
				return fmt.Errorf("Could not create file to save kubeconfig of cluster %s: [%v]", shootName, err)
			}
		}
	}

	settings := CreateSetting("", kubeconfigFile)
	cleanUp := c.CleanReleaseFail(kubeconfigFile)
	if cleanUp != nil {
		return fmt.Errorf("Unable to clean Release fail in shoot cluster [%s]: %v", shootName, cleanUp)
	}
	checkInstalled, err := c.CheckChartInstalled(settings)
	if err != nil {
		return fmt.Errorf("Unable to check release GPU is installed in cluster %s: [%v]", shootName, err)
	}
	if checkInstalled == GPUChartAlreadyInstalled {
		klog.Infof("Shoot cluster %s have already installed charts GPU - skipping install helm chart to shoot\n", shootName)
		return nil
	}
	klog.V(3).Infof("Shoot cluster %s is enabled GPU - starting install helm chart to shoot\n", shootName)

	label := c.getLabelWorkerGroup(ctx, machineDeployment)
	if label == nil {
		strategy = "mig.strategy=none"
	} else {
		switch value := label["nvidia.com/mig.config"]; value {
		case "all-1g.6gb":
			strategy = "mig.strategy=single"
		case "all-2g.12gb":
			strategy = "mig.strategy=single"
		case "all-balanced":
			strategy = "mig.strategy=mixed"
		}
	}
	// Add helm repo
	repoAdd(repoName, url, settings)
	// Update charts from the helm repo
	repoUpdate(settings)
	if checkInstalled == GPUChartNotInstalled {
		// Install GPU Operator
		err = c.InstallGPUOperatorChart(repoName, strategy, kubeconfigFile)
		if err != nil {
			return fmt.Errorf("Unable to install chart gpu-operator to cluster %s: [%v]", shootName, err)
		}
		// Install kube-prometheus-stack
		err = c.InstallPrometheusStackChart(repoName, kubeconfigFile)
		if err != nil {
			return fmt.Errorf("Unable to install chart kube-prometheus-stack to cluster %s: [%v]", shootName, err)
		}
		// Install prometheus-adapter
		err = c.InstallPrometheusAdapterChart(repoName, kubeconfigFile)
		if err != nil {
			return fmt.Errorf("Unable to install chart prometheus-adapter to cluster %s: [%v]", shootName, err)
		}
	}
	if checkInstalled == InstallPrometheusStack {
		err = c.InstallPrometheusStackChart(repoName, kubeconfigFile)
		if err != nil {
			return fmt.Errorf("Unable to install chart kube-prometheus-stack to cluster %s: [%v]", shootName, err)
		}
		err = c.InstallPrometheusAdapterChart(repoName, kubeconfigFile)
		if err != nil {
			return fmt.Errorf("Unable to install chart prometheus-adapter to cluster %s: [%v]", shootName, err)
		}
	}
	if checkInstalled == InstallPrometheusAdapter {
		err = c.InstallPrometheusAdapterChart(repoName, kubeconfigFile)
		if err != nil {
			return fmt.Errorf("Unable to install chart prometheus-adapter to cluster %s: [%v]", shootName, err)
		}
	}
	return nil
}
func (c *controller) InstallGPUOperatorChart(repoName, value, kubeconfigFile string) error {
	klog.V(4).Infof("Strategy of GPU operator: %s", value)
	settings := CreateSetting("gpu-operator", kubeconfigFile)
	err := c.installChart(repoName, "gpu-operator", value, settings)
	time.Sleep(30 * time.Second)
	return err
}
func (c *controller) InstallPrometheusStackChart(repoName, kubeconfigFile string) error {
	settings := CreateSetting("prometheus", kubeconfigFile)
	err := c.installChart(repoName, "kube-prometheus-stack", "", settings)
	time.Sleep(45 * time.Second)
	return err
}
func (c *controller) InstallPrometheusAdapterChart(repoName, kubeconfigFile string) error {
	settings := CreateSetting("prometheus", kubeconfigFile)
	service, err := c.controlCoreClient.CoreV1().Services("prometheus").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app=kube-prometheus-stack-prometheus",
	})
	if err != nil {
		return fmt.Errorf("Unable to get svc with selector in cluster %s: [%v]", c.namespace, err)
	}
	prometheus_service := service.Items[0].Name
	value := "prometheus.url=http://" + prometheus_service + ".prometheus.svc.cluster.local"
	err = c.installChart(repoName, "prometheus-adapter", value, settings)
	time.Sleep(15 * time.Second)
	return err
}

func CreateSetting(namespace string, kubeconfig string) *cli.EnvSettings {
	os.Setenv("HELM_NAMESPACE", namespace)
	settings := cli.New()
	settings.KubeConfig = kubeconfig
	return settings
}

// RepoAdd adds repo with given name and url
func repoAdd(name, url string, settings *cli.EnvSettings) error {
	repoFile := settings.RepositoryConfig

	//Ensure the file directory exists as it is required for file locking
	err := os.MkdirAll(filepath.Dir(repoFile), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("Could not create dir to store chart repository: %v", err)
	}

	// Acquire a file lock for process synchronization
	fileLock := flock.New(strings.Replace(repoFile, filepath.Ext(repoFile), ".lock", 1))
	lockCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	locked, err := fileLock.TryLockContext(lockCtx, time.Second)
	if err == nil && locked {
		defer fileLock.Unlock()
	}
	if err != nil {
		return fmt.Errorf("Could not acquire a file lock for process synchronization: %v", err)
	}

	b, err := ioutil.ReadFile(repoFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Could not read chart repository file: %v", err)
	}

	var f repo.File
	if err := yaml.Unmarshal(b, &f); err != nil {
		return fmt.Errorf("Unable to unmarshal repository file: %v", err)
	}

	if f.Has(name) {
		// fmt.Printf("repository name (%s) already exists\n", name)
		klog.Infof("repository name (%s) already exists\n", name)
		return nil
	}

	c := repo.Entry{
		Name: name,
		URL:  url,
	}

	r, err := repo.NewChartRepository(&c, getter.All(settings))
	if err != nil {
		return fmt.Errorf("Could not construct chart repository: %v", err)
	}

	if _, err := r.DownloadIndexFile(); err != nil {
		err := errors.Wrapf(err, "looks like %q is not a valid chart repository or cannot be reached", url)
		return fmt.Errorf("Could not reach repository file: %v", err)
	}

	f.Update(&c)

	if err := f.WriteFile(repoFile, 0644); err != nil {
		return fmt.Errorf("Could not write content to chart repository file: %v", err)
	}
	// fmt.Printf("%q has been added to your repositories\n", name)
	return nil
}

// RepoUpdate updates charts for all helm repos
func repoUpdate(settings *cli.EnvSettings) error {
	repoFile := settings.RepositoryConfig

	f, err := repo.LoadFile(repoFile)
	if os.IsNotExist(errors.Cause(err)) || len(f.Repositories) == 0 {
		return fmt.Errorf("Chart not exist: %v", err)
	}
	var repos []*repo.ChartRepository
	for _, cfg := range f.Repositories {
		r, err := repo.NewChartRepository(cfg, getter.All(settings))
		if err != nil {
			return fmt.Errorf("Unable to construct chart repository: %s\n", err.Error())
		}
		repos = append(repos, r)
	}

	// fmt.Printf("Hang tight while we grab the latest from your chart repositories...\n")
	var wg sync.WaitGroup
	for _, re := range repos {
		wg.Add(1)
		go func(re *repo.ChartRepository) {
			defer wg.Done()
			if _, err := re.DownloadIndexFile(); err != nil {
				klog.Errorf("...Unable to get an update from the %q chart repository (%s):\n\t%v", re.Config.Name, re.Config.URL, err.Error())
			} else {
				klog.Infof("...Successfully got an update from the %q chart repository\n", re.Config.Name)
			}
		}(re)
	}
	wg.Wait()
	// fmt.Printf("Update Complete. ⎈ Happy Helming!⎈\n")
	klog.Infof("Update Complete. ⎈ Happy Helming!⎈\n")
	return nil
}

// InstallChart
func (c *controller) installChart(repo, chart, value string, settings *cli.EnvSettings) error {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), debug); err != nil {
		// log.Fatal(err)
		return fmt.Errorf("Unable to init the action configuration: %v", err)
	}
	client := action.NewInstall(actionConfig)

	if client.Version == "" && client.Devel {
		client.Version = ">0.0.0-0"
	}
	//name, chart, err := client.NameAndChart(args)
	rand.Seed(time.Now().UnixNano())
	client.ReleaseName = fmt.Sprintf("%s-%d", chart, rand.Intn(1000000000))
	client.CreateNamespace = true
	client.GenerateName = true

	cp, err := client.ChartPathOptions.LocateChart(fmt.Sprintf("%s/%s", repo, chart), settings)
	if err != nil {
		// log.Fatal(err)
		return fmt.Errorf("Unable to locate chart repository: %v", err)
	}

	debug("CHART PATH: %s\n", cp)

	p := getter.All(settings)
	valueOpts := &values.Options{}
	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		// log.Fatal(err)
		return fmt.Errorf("Unable to merge value from file values.yaml to chart: %v", err)
	}
	args := map[string]string{
		"set": value,
	}
	// Add args
	if value != "" {
		if err := strvals.ParseInto(args["set"], vals); err != nil {
			// log.Fatal(errors.Wrap(err, "failed parsing --set data"))
			return fmt.Errorf("Unable to parse argument to chart: %v", err)
		}
	}

	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := loader.Load(cp)
	if err != nil {
		return fmt.Errorf("Unable to load chart from chart path: %v", err)
	}

	validInstallableChart, err := isChartInstallable(chartRequested)
	if !validInstallableChart {
		// log.Fatal(err)
		return fmt.Errorf("Unable to install chart into shoot cluster: %v", err)
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		// If CheckDependencies returns an error, we have unfulfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		// https://github.com/helm/helm/issues/2209
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			if client.DependencyUpdate {
				man := &downloader.Manager{
					Out:              os.Stdout,
					ChartPath:        cp,
					Keyring:          client.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          p,
					RepositoryConfig: settings.RepositoryConfig,
					RepositoryCache:  settings.RepositoryCache,
				}
				if err := man.Update(); err != nil {
					return fmt.Errorf("Unable to update manager: %v", err)
				}
			} else {
				// log.Fatal(err)
				return fmt.Errorf("Unable to update dependency: %v", err)
			}
		}
	}

	client.Namespace = settings.Namespace()
	release, err := client.Run(chartRequested, vals)
	if err != nil {
		if err.Error() == "cannot re-use a name that is still in use" {
			klog.Infof("Chart %s already installed in shoot cluster", chart)
			return nil
		} else {
			return fmt.Errorf("Unable to install chart to shoot cluster %s: [%v]", c.namespace, err)
		}
	} else {
		klog.Infof("Successed install chart [%s] to shoot cluster [%s] by release [%s] !", chart, c.namespace, release.Name)
	}
	return nil
}

func UnInstallChart(name string, settings *cli.EnvSettings) error {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), debug); err != nil {
		return fmt.Errorf("Unable to initialize action config to uninstall release [%s]: %v", name, err)
	}
	client := action.NewUninstall(actionConfig)
	_, err := client.Run(name)
	if err != nil {
		return fmt.Errorf("Unable to uninstall release [%s]: %v", name, err)
	}
	klog.Infof("Release %s has been uninstalled\n", name)
	return nil
}

func isChartInstallable(ch *hchart.Chart) (bool, error) {
	switch ch.Metadata.Type {
	case "", "application":
		return true, nil
	}
	return false, errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}

func debug(format string, v ...interface{}) {
	format = fmt.Sprintf("[debug] %s\n", format)
	log.Output(2, fmt.Sprintf(format, v...))
}
func (c *controller) CheckChartInstalled(settings *cli.EnvSettings) (int, error) {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), "", os.Getenv("HELM_DRIVER"), debug); err != nil {
		return -1, fmt.Errorf("Unable to init action config to list releases in shoot cluster %s: [%v]", c.namespace, err)
	}
	client := action.NewList(actionConfig)

	client.AllNamespaces = true
	client.Deployed = true
	release, err := client.Run()
	if err != nil {
		return -1, fmt.Errorf("Unable to list releases in shoot cluster %s: [%v]\n", c.namespace, err)
	}
	var CASE = GPUChartNotInstalled
	for _, r := range release {
		if strings.Contains(r.Name, "prometheus-adapter") {
			CASE = GPUChartAlreadyInstalled
			break
		}
		if strings.Contains(r.Name, "gpu-operator") {
			CASE = InstallPrometheusStack
		}
		if strings.Contains(r.Name, "kube-prometheus-stack") {
			CASE = InstallPrometheusAdapter
		}
	}
	return CASE, nil
}
func (c *controller) CleanReleaseFail(kubeconfigFile string) error {
	settings := CreateSetting("", kubeconfigFile)
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), "", os.Getenv("HELM_DRIVER"), debug); err != nil {
		return fmt.Errorf("Unable to init action config to list releases in shoot cluster %s: [%v]", c.namespace, err)
	}
	client := action.NewList(actionConfig)

	client.AllNamespaces = true
	// client.Failed = true
	// client.Pending = true
	// client.Deployed = false
	release, err := client.Run()
	if err != nil {
		return fmt.Errorf("Unable to list releases in shoot cluster %s: [%v]", c.namespace, err)
	}
	for _, r := range release {
		if r.Info.Status != "deployed" {
			settings = CreateSetting(r.Namespace, kubeconfigFile)
			err = UnInstallChart(r.Name, settings)
			if err != nil {
				klog.Infof("Uninstall release [%s] in namespace [%s] fail", r.Name, r.Namespace)
				return err
			}
		}
	}
	return nil
}

func DecodeProviderSpecFromMachineClass(machineClass *v1alpha1.MachineClass) (*ProviderSpec, error) {
	// Extract providerSpec
	var providerSpec *ProviderSpec

	if machineClass == nil {
		return nil, errors.New("MachineClass provided is nil")
	}

	jsonErr := json.Unmarshal(machineClass.ProviderSpec.Raw, &providerSpec)
	if jsonErr != nil {
		return nil, fmt.Errorf("Failed to parse JSON data provided as ProviderSpec: %v", jsonErr)
	}

	return providerSpec, nil
}
func (c *controller) getLabelWorkerGroup(ctx context.Context, machineDeployment *v1alpha1.MachineDeployment) map[string]string {
	// machineDeployment := c.getMachineDeploymentForMachine(ctx, machine)
	label := make(map[string]string)
	label = machineDeployment.Spec.Template.Spec.NodeTemplateSpec.Labels
	return label
}

// getDeploymentForMachine returns the deployment managing the given Machine.
// func (c *controller) getMachineDeploymentForMachine(ctx context.Context, machine *v1alpha1.Machine) *v1alpha1.MachineDeployment {
// 	// Find the owning machine set
// 	var is *v1alpha1.MachineSet
// 	var err error
// 	controllerRef := metav1.GetControllerOf(machine)
// 	if controllerRef == nil {
// 		// No controller owns this Machine.
// 		return nil
// 	}
// 	if controllerRef.Kind != "MachineSet" { //TODO: Remove hardcoded string
// 		// Not a Machine owned by a machine set.
// 		return nil
// 	}
// 	is, err = c.controlMachineClient.MachineSets(machine.Namespace).Get(ctx, controllerRef.Name, metav1.GetOptions{})
// 	if err != nil || is.UID != controllerRef.UID {
// 		klog.V(4).Infof("Cannot get machineset %q for machine %q: %v", controllerRef.Name, machine.Name, err)
// 		return nil
// 	}

// 	// Now find the Deployment that owns that MachineSet.
// 	controllerRef = metav1.GetControllerOf(is)
// 	if controllerRef == nil {
// 		return nil
// 	}
// 	return c.resolveDeploymentControllerRef(is.Namespace, controllerRef)
// }

// // resolveControllerRef returns the controller referenced by a ControllerRef,
// // or nil if the ControllerRef could not be resolved to a matching controller
// // of the correct Kind.
// func (c *controller) resolveDeploymentControllerRef(namespace string, controllerRef *metav1.OwnerReference) *v1alpha1.MachineDeployment {
// 	// We can't look up by UID, so look up by Name and then verify UID.
// 	// Don't even try to look up by Name if it's the wrong Kind.
// 	// controllerKind contains the schema.GroupVersionKind for this controller type.
// 	var controllerKind = v1alpha1.SchemeGroupVersion.WithKind("MachineDeployment")
// 	if controllerRef.Kind != controllerKind.Kind {
// 		return nil
// 	}
// 	d, err := c.controlMachineClient.MachineDeployments(namespace).Get(context.TODO(), controllerRef.Name, metav1.GetOptions{})
// 	if err != nil {
// 		return nil
// 	}
// 	if d.UID != controllerRef.UID {
// 		// The controller we found with this Name is not the same one that the
// 		// ControllerRef points to.
// 		return nil
// 	}
// 	return d
// }
