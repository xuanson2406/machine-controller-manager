package controller

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/startstop"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/utils/client"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/xuanson2406/machine-controller-manager/pkg/apis/machine/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// novaV2 is a NovaV2 client implementing the Compute interface.
type novaV2 struct {
	serviceClient *gophercloud.ServiceClient
}

func (c *controller) RebootInstanceOPS(machine *v1alpha1.Machine) error {
	credential := make(map[string]string)
	machineClass, err := c.machineClassLister.MachineClasses(c.namespace).Get(machine.Spec.Class.Name)
	if err != nil {
		klog.Errorf("MachineClass %s/%s not found. Skipping. %v", c.namespace, machine.Spec.Class.Name, err)
		return err
	}
	secretData, err := c.getSecretData(machineClass.Name, machineClass.SecretRef, machineClass.CredentialsSecretRef)
	if err != nil {
		klog.Errorf("Could not compute secret data: %+v", err)
		return err
	}
	credential["authURL"] = string(secretData["authURL"])
	credential["username"] = string(secretData["username"])
	credential["password"] = string(secretData["password"])
	credential["domainName"] = string(secretData["domainName"])
	credential["tenantName"] = string(secretData["tenantName"])
	credential["region"] = string(secretData["region"])
	client, err := GetClientOPSForController(credential)
	if err != nil {
		return err
	}
	pages, err := servers.List(client.serviceClient, servers.ListOpts{Name: machine.Name}).AllPages()
	if err != nil {
		return err
	}
	server, err := servers.ExtractServers(pages)
	if server == nil {
		return nil
	}
	if err != nil {
		return fmt.Errorf("Unable to get server [%s] in tenant [%s] - region [%s]: [%v]", machine.Name, credential["tenantName"], credential["region"], err.Error())
	}
	if server[0].Status == "SHUTOFF" {
		klog.V(3).Infof("Server [%s] in state [%s] - Starting Server", machine.Name, server[0].Status)
		result := startstop.Start(client.serviceClient, server[0].ID)
		if result.Err != nil {
			return fmt.Errorf("Unable to start the server [%s]: [%v]", machine.Name, result.Err)
		} else {
			return nil
		}
	}
	RebootOpt := &servers.RebootOpts{Type: servers.OSReboot}
	timeToHard := 5 * time.Minute
	Reboot := metav1.Now().Add(-timeToHard).Sub(machine.Status.CurrentStatus.LastUpdateTime.Time)
	if Reboot > 0 {
		klog.V(3).Infof("Machine %s need to hard reboot after 6 minute UnKnown", machine.Name)
		RebootOpt = &servers.RebootOpts{Type: servers.PowerCycle}
		time.Sleep(1 * time.Minute)
	}
	r := servers.Reboot(client.serviceClient, server[0].ID, RebootOpt)
	if r.Err != nil {
		return fmt.Errorf("Unable to reboot the server [%s]: [%v]", machine.Name, r.Err)
	}
	return nil
}
func GetClientOPSForController(credential map[string]string) (*novaV2, error) {
	// if one accidentially copies a newline character into the token, remove it!
	if strings.Contains(credential["userName"], "\n") {
		klog.V(4).Infof("Your vCD username contains a newline character. I will remove it for you but you should consider to remove it.")
		credential["userName"] = strings.Replace(credential["userName"], "\n", "", -1)
	}
	if strings.Contains(credential["password"], "\n") {
		klog.V(4).Infof("Your vCD password contains a newline character. I will remove it for you but you should consider to remove it.")
		credential["password"] = strings.Replace(credential["password"], "\n", "", -1)
	}
	config := &tls.Config{}
	config.InsecureSkipVerify = true
	clientOpts := new(clientconfig.ClientOpts)
	authInfo := &clientconfig.AuthInfo{
		AuthURL:     credential["authURL"],
		Username:    credential["username"],
		Password:    credential["password"],
		DomainName:  credential["domainName"],
		ProjectName: credential["tenantName"],
	}
	clientOpts.AuthInfo = authInfo

	if clientOpts.AuthInfo.ApplicationCredentialSecret != "" {
		clientOpts.AuthType = clientconfig.AuthV3ApplicationCredential
	}

	ao, err := clientconfig.AuthOptions(clientOpts)
	if err != nil {
		fmt.Printf("failed to create client auth options: %v", err)
	}

	provider, err := openstack.NewClient(ao.IdentityEndpoint)
	if err != nil {
		fmt.Printf("failed to create authenticated client: %v", err)
	}

	// Set UserAgent
	provider.UserAgent.Prepend("Machine Controller Provider Openstack")

	transport := &http.Transport{Proxy: http.ProxyFromEnvironment, TLSClientConfig: config}
	provider.HTTPClient = http.Client{
		Transport: transport,
	}

	provider.HTTPClient.Transport = &client.RoundTripper{
		Rt: provider.HTTPClient.Transport,
	}
	err = openstack.Authenticate(provider, *ao)
	if err != nil {
		return nil, fmt.Errorf("Unable to authenticate with credential: [%v]", err)
	}
	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: credential["region"],
	})
	if err != nil {
		return nil, fmt.Errorf("Unable to initial compute client with credential: [%s]", err)
	}
	return &novaV2{
		serviceClient: client,
	}, nil

}
