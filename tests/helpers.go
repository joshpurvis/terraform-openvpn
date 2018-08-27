package test

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/shell"
	"github.com/gruntwork-io/terratest/modules/ssh"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/gruntwork-io/terratest/modules/test-structure"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ssh/agent"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type SSHAgent struct {
	stop       chan bool
	stopped    chan bool
	socketDir  string
	socketFile string
	agent      agent.Agent
	ln         net.Listener
}

func dockerCommand(t *testing.T, args []string) string {
	output, _ := shell.RunCommandAndGetOutputE(t, shell.Command{
		Command: "docker",
		Args:    args,
	})

	return output
}

func openvpnConnectionTest(t *testing.T, terraformDirectory string, publicIP string) {
	// connects to openvpn server using generated ovpn file, and checks its actual ip
	// using http://ifconfig.co, to confirm it matches publicIP from terraform outputs.

	// confirm that an ovpn file was SCP'd from vpn server
	absoluteTerraformDirectory, _ := filepath.Abs(terraformDirectory)
	assert.FileExists(t, filepath.Join(absoluteTerraformDirectory, "terratest-openvpn.ovpn"))

	uniqueName := test_structure.LoadString(t, terraformDirectory, "uniqueName")

	// create parent container, which maintains connection to the vpn
	daemonName := fmt.Sprintf("daemon-%s", uniqueName)
	dockerCommand(t, []string{
		"run",
		"--name", daemonName,
		"-d",
		"--cap-add=NET_ADMIN",
		"--dns=8.8.8.8",
		"--device=/dev/net/tun",
		fmt.Sprintf("-v=%s:/vpn", absoluteTerraformDirectory),
		"dperson/openvpn-client",
	})

	time.Sleep(10 * time.Second)

	// TODO: is this step actually needed? seems so according dperson/openvpn-client doc
	dockerCommand(t, []string{
		"restart",
		daemonName,
	})

	time.Sleep(10 * time.Second)

	// execute curl command, while routing traffic through the daemon container above
	outputIP := dockerCommand(t, []string{
		"run",
		"--rm",
		"--name", fmt.Sprintf("client-%s", uniqueName),
		"--net", fmt.Sprintf("container:%s", daemonName),
		"joshpurvis/alpine-curl",
		"--fail",
		"--silent",
		"--show-error",
		"ifconfig.co",
	})

	// cleanup container
	dockerCommand(t, []string{
		"rm",
		"-f",
		daemonName,
	})

	// ensure that the actual IP matches the public IP of the vpn instance.
	assert.Equal(t, publicIP, outputIP)

}

// Create SSH agent, start it in background and returns control back to the main thread
func NewSSHAgent(socketDir string, socketFile string) (*SSHAgent, error) {
	var err error
	s := &SSHAgent{make(chan bool), make(chan bool), socketDir, socketFile, agent.NewKeyring(), nil}
	s.ln, err = net.Listen("unix", s.socketFile)
	if err != nil {
		return nil, err
	}
	go s.run()
	return s, nil
}

// SSH Agent listener and handler
func (s *SSHAgent) run() {
	defer close(s.stopped)
	for {
		select {
		case <-s.stop:
			return
		default:
			c, err := s.ln.Accept()
			if err != nil {
				select {
				// When s.Stop() closes the listner, s.ln.Accept() returns an error that can be ignored
				// since the agent is in stopping process
				case <-s.stop:
					return
					// When s.ln.Accept() returns a legit error, we print it and continue accepting further requests
				default:
					fmt.Errorf("Could not accept connection to agent %v", err)
					continue
				}
			} else {
				defer c.Close()
				go func(c io.ReadWriter) {
					err := agent.ServeAgent(s.agent, c)
					if err != nil {
						fmt.Errorf("Could not serve ssh agent %v", err)
					}
				}(c)
			}
		}
	}
}

// Stop and clean up SSH agent
func (s *SSHAgent) Stop() {
	close(s.stop)
	s.ln.Close()
	<-s.stopped
	os.RemoveAll(s.socketDir)
}

func sshAgentWithKeyPair(t *testing.T, keyPair *ssh.KeyPair) *SSHAgent {

	// Instantiate a temporary SSH agent
	socketDir, err := ioutil.TempDir("", "ssh-agent-")
	if err != nil {
		t.Fatal(err)
	}
	socketFile := filepath.Join(socketDir, "ssh_auth.sock")
	os.Setenv("SSH_AUTH_SOCK", socketFile)
	sshAgent, err := NewSSHAgent(socketDir, socketFile)
	if err != nil {
		t.Fatal(err)
	}

	// Create SSH key for the agent using the given AWS SSH key pair
	block, _ := pem.Decode([]byte(keyPair.PrivateKey))
	pkey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	key := agent.AddedKey{PrivateKey: pkey}

	// Add SSH key to the agent
	// Retry until agent is ready or give up with a fatal error
	for i := 0; i < 15; i++ {
		var keys []*agent.Key
		keys, err = sshAgent.agent.List()
		if err != nil {
			logger.Logf(t, "Error listing SSH keys %v", err)
		}
		if len(keys) > 0 {
			logger.Logf(t, "Agent SSH keys: %v", keys)
			break
		} else {
			err = sshAgent.agent.Add(key)
			if err != nil {
				logger.Logf(t, "Error adding SSH key %v", err)
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	if err != nil {
		t.Fatal("Could not add any SSH key to the agent after several retries")
	}

	return sshAgent
}

func configureTerraformOptions(t *testing.T, terraformDirectory string) (*terraform.Options, *aws.Ec2Keypair) {

	// randomize instance name
	uniqueId := random.UniqueId()
	uniqueName := fmt.Sprintf("terratest-openvpn-%s", uniqueId)
	test_structure.SaveString(t, terraformDirectory, "uniqueName", uniqueName)

	// Pick a random AWS region to test in
	awsRegion := aws.GetRandomRegion(t, nil, nil)

	// Create an EC2 KeyPair that we can use for SSH access
	keyPair := aws.CreateAndImportEC2KeyPair(t, awsRegion, uniqueName)

	terraformOptions := &terraform.Options{
		// The path to where our Terraform code is located
		TerraformDir: terraformDirectory,

		// Variables to pass to our Terraform code using -var options
		Vars: map[string]interface{}{
			"client_name":       "terratest-openvpn",
			"aws_region":        awsRegion,
			"instance_name":     uniqueName,
			"aws_key_pair_name": keyPair.Name,
		},
	}

	return terraformOptions, keyPair
}
