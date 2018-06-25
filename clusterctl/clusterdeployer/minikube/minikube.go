package minikube

import (
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

type Minikube struct {
	kubeconfigpath string
	vmDriver       string
	proxy          string
	// minikubeExec implemented as function variable for testing hooks
	minikubeExec func(env []string, args ...string) (string, error)
}

func New(vmDriver, proxy string) *Minikube {
	return &Minikube{
		minikubeExec: minikubeExec,
		vmDriver:     vmDriver,
		proxy:        proxy,
		// Arbitrary file name. Can potentially be randomly generated.
		kubeconfigpath: "minikube.kubeconfig",
	}
}

var minikubeExec = func(env []string, args ...string) (string, error) {
	const executable = "minikube"
	glog.V(3).Infof("Running: %v %v", executable, args)
	cmd := exec.Command(executable, args...)
	cmd.Env = env
	cmdOut, err := cmd.CombinedOutput()
	glog.V(2).Infof("Ran: %v %v Output: %v", executable, args, string(cmdOut))
	if err != nil {
		err = fmt.Errorf("error running command '%v %v': %v", executable, strings.Join(args, " "), err)
	}
	return string(cmdOut), err
}

func (m *Minikube) Create() error {
	args := []string{"start", "--bootstrapper=kubeadm"}
	if m.vmDriver != "" {
		args = append(args, fmt.Sprintf("--vm-driver=%v", m.vmDriver))
	}
	if m.proxy != "" {
		proxyUrl, err := url.Parse(m.proxy)
		if proxyUrl == nil {
			glog.Error("could not parse proxy. did you forget \"http://\"?")
		}
		if err != nil {
			err = fmt.Errorf("error parsing proxy '%v %v': %v", m.proxy, strings.Join(args, " "), err)
		}
		args = append(args, "--docker-env")
		args = append(args, fmt.Sprintf("http_proxy=%s", m.proxy))
		args = append(args, "--docker-env")
		args = append(args, fmt.Sprintf("https_proxy=%s", strings.Replace(m.proxy, "http://", "https://", 1)))
		args = append(args, "--docker-env")
		args = append(args, fmt.Sprintf("no_proxy=%s,192.168.0.0/16", proxyUrl.Hostname()))
	}
	_, err := m.exec(args...)
	if err != nil {
		return err
	}

	args = []string{"update-context"}
	_, err = m.exec(args...)

	return err
}

func (m *Minikube) Delete() error {
	_, err := m.exec("delete")
	os.Remove(m.kubeconfigpath)
	return err
}

func (m *Minikube) GetKubeconfig() (string, error) {
	b, err := ioutil.ReadFile(m.kubeconfigpath)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (m *Minikube) exec(args ...string) (string, error) {
	// Override kubeconfig environment variable in call
	// so that minikube will generate and reference
	// the kubeconfig in the desired location.
	// Note that the last value set for a key is the final value.
	const kubeconfigEnvVar = "KUBECONFIG"
	env := append(os.Environ(), fmt.Sprintf("%v=%v", kubeconfigEnvVar, m.kubeconfigpath))
	return m.minikubeExec(env, args...)
}
