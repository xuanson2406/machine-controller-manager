// Handle request for worker group enable GPU
// Package controller is used to provide the core functionalities of machine-controller-manager
package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/flock"
	"github.com/xuanson2406/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/klog/v2"
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

func (c *controller) CheckGPUWorkerGroup(machineName string) (bool, error) {
	machineClass, err := c.machineClassLister.MachineClasses(c.namespace).Get(machineName)
	if err != nil {
		klog.Errorf("MachineClass %s/%s not found. Skipping. %v", c.namespace, machineClass.Name, err)
		return false, err
	}
	providerSpec, err := DecodeProviderSpecFromMachineClass(machineClass)
	if err != nil {
		return false, err
	}
	if providerSpec.VGPU == "gpu" {
		return true, nil
	}
	return false, nil
}

// RepoAdd adds repo with given name and url
func RepoAdd(name, url string) {
	settings := cli.New()
	repoFile := settings.RepositoryConfig

	//Ensure the file directory exists as it is required for file locking
	err := os.MkdirAll(filepath.Dir(repoFile), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		log.Fatal(err)
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
		log.Fatal(err)
	}

	b, err := ioutil.ReadFile(repoFile)
	if err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}

	var f repo.File
	if err := yaml.Unmarshal(b, &f); err != nil {
		log.Fatal(err)
	}

	if f.Has(name) {
		fmt.Printf("repository name (%s) already exists\n", name)
		return
	}

	c := repo.Entry{
		Name: name,
		URL:  url,
	}

	r, err := repo.NewChartRepository(&c, getter.All(settings))
	if err != nil {
		log.Fatal(err)
	}

	if _, err := r.DownloadIndexFile(); err != nil {
		err := errors.Wrapf(err, "looks like %q is not a valid chart repository or cannot be reached", url)
		log.Fatal(err)
	}

	f.Update(&c)

	if err := f.WriteFile(repoFile, 0644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%q has been added to your repositories\n", name)
}

// RepoUpdate updates charts for all helm repos
func RepoUpdate() {
	repoFile := settings.RepositoryConfig

	f, err := repo.LoadFile(repoFile)
	if os.IsNotExist(errors.Cause(err)) || len(f.Repositories) == 0 {
		log.Fatal(errors.New("no repositories found. You must add one before updating"))
	}
	var repos []*repo.ChartRepository
	for _, cfg := range f.Repositories {
		r, err := repo.NewChartRepository(cfg, getter.All(settings))
		if err != nil {
			log.Fatal(err)
		}
		repos = append(repos, r)
	}

	fmt.Printf("Hang tight while we grab the latest from your chart repositories...\n")
	var wg sync.WaitGroup
	for _, re := range repos {
		wg.Add(1)
		go func(re *repo.ChartRepository) {
			defer wg.Done()
			if _, err := re.DownloadIndexFile(); err != nil {
				fmt.Printf("...Unable to get an update from the %q chart repository (%s):\n\t%s\n", re.Config.Name, re.Config.URL, err)
			} else {
				fmt.Printf("...Successfully got an update from the %q chart repository\n", re.Config.Name)
			}
		}(re)
	}
	wg.Wait()
	fmt.Printf("Update Complete. ⎈ Happy Helming!⎈\n")
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
