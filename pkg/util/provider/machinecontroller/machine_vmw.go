package controller

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/xuanson2406/go-vcloud-director-fptcloud/v2/govcd"
	"github.com/xuanson2406/machine-controller-manager/pkg/apis/machine/v1alpha1"

	"k8s.io/klog/v2"
)

func (c *controller) RebootVM(machine *v1alpha1.Machine) error {
	credential := make(map[string]string)
	machineClass, err := c.machineClassLister.MachineClasses(c.namespace).Get(machine.Spec.Class.Name)
	if err != nil {
		klog.Errorf("MachineClass %s/%s not found. Skipping. %v", c.namespace, machine.Spec.Class.Name, err)
		return err
	}
	secrets, err := c.getSecret(machineClass.SecretRef, machineClass.Name)
	if err != nil {
		return err
	}
	credential["userName"] = string(secrets.Data["fptcloudUser"])
	credential["password"] = string(secrets.Data["fptcloudPassword"])
	credential["orgName"] = string(secrets.Data["fptcloudOrg"])
	credential["vcdHref"] = string(secrets.Data["fptcloudHref"])
	credential["vdcName"] = string(secrets.Data["fptcloudVDC"])
	client, err := GetClientForController(credential)
	if err != nil {
		return err
	}
	vdc, _ := GetVdcByName(client, credential["orgName"], credential["vdcName"])
	vapp := vdc.GetVappList()
	for _, i := range vapp {
		if strings.Contains(i.Name, machine.Name) {
			VAPP, _ := vdc.GetVAppByName(i.Name, false)
			vm, _ := VAPP.GetVMById(VAPP.VApp.Children.VM[0].ID, false)
			if vm.VM.Status != 8 {
				for {
					unDeployTask, err := vm.Undeploy()
					if err != nil {
						if strings.Contains(err.Error(), "API Error") {
							time.Sleep(3 * time.Second)
							continue
						} else {
							return fmt.Errorf("unable to power of vm %s: [%v]", machine.Name, err)
						}
					}
					err = unDeployTask.WaitTaskCompletion()
					if err != nil {
						return fmt.Errorf("unable to wait for power of vm %s completion: [%v]", machine.Name, err)
					}
					time.Sleep(2 * time.Second)
					break
				}
			}
			klog.V(4).Infof("VM %s is power off - try to restart VM", machine.Name)
			for {
				err = vm.PowerOnAndForceCustomization()
				if err != nil {
					if strings.Contains(err.Error(), "API Error") {
						time.Sleep(3 * time.Second)
						continue
					} else {
						return fmt.Errorf("unable to power on vm %s with recustomizing option: [%v]", machine.Name, err)
					}
				}
				klog.V(4).Infof("VM %s is power on - wait to Machine ready", machine.Name)
				break
			}
			break
			// klog.V(4).Infof("VM %s is power on - try to reboot VM - wait to Machine ready", machine.Name)
			// task, err := VAPP.Reboot()
			// if err != nil {
			// 	if strings.Contains(err.Error(), "API Error") {
			// 		time.Sleep(3 * time.Second)
			// 	} else {
			// 		return fmt.Errorf("unable to reboot vm [%v]", err)
			// 	}
			// }
			// err = task.WaitTaskCompletion()
			// if err != nil {
			// 	return fmt.Errorf("unable to wait for reboot vm completion: [%v]", err)
			// }
			// break
		}
	}
	return nil
}
func GetClientForController(credential map[string]string) (*govcd.VCDClient, error) {
	// if one accidentially copies a newline character into the token, remove it!
	if strings.Contains(credential["userName"], "\n") {
		klog.V(4).Infof("Your vCD username contains a newline character. I will remove it for you but you should consider to remove it.")
		credential["userName"] = strings.Replace(credential["userName"], "\n", "", -1)
	}
	if strings.Contains(credential["password"], "\n") {
		klog.V(4).Infof("Your vCD password contains a newline character. I will remove it for you but you should consider to remove it.")
		credential["password"] = strings.Replace(credential["password"], "\n", "", -1)
	}
	client, err := Login(credential["userName"], credential["password"], credential["orgName"], credential["vcdHref"])

	if err != nil {
		klog.V(4).Infof(err.Error())
	}
	return client, err
}
func Login(User string, Password string, Org string, HREF string) (*govcd.VCDClient, error) {
	u, err := url.ParseRequestURI(HREF)
	if err != nil {
		return nil, fmt.Errorf("unable to pass url: %s", err)
	}
	vcdclient := govcd.NewVCDClient(*u, true)
	err = vcdclient.Authenticate(User, Password, Org)
	if err != nil {
		return nil, fmt.Errorf("unable to authenticate: %s", err)
	}
	return vcdclient, nil
}
func GetVdcByName(client *govcd.VCDClient, Org string, VDC string) (*govcd.Vdc, error) {
	org, err := client.GetOrgByName(Org)
	if err != nil {
		return nil, fmt.Errorf("unable to get org %s by name: [%v]", Org, err)
	}
	vdc, err := org.GetVDCByName(VDC, false)
	if err != nil {
		return nil, fmt.Errorf("unable to get vdc %s by name: [%v]", VDC, err)
	}
	return vdc, err
}
