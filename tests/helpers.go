package test

import (
	"fmt"
	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/shell"
	"github.com/gruntwork-io/terratest/modules/ssh"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/gruntwork-io/terratest/modules/test-structure"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ssh/agent"
	"net"
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

func configureTerraformOptions(t *testing.T, terraformDirectory string) (*terraform.Options, *aws.Ec2Keypair, *ssh.SshAgent) {

	// randomize instance name
	uniqueId := random.UniqueId()
	uniqueName := fmt.Sprintf("terratest-openvpn-%s", uniqueId)
	test_structure.SaveString(t, terraformDirectory, "uniqueName", uniqueName)

	// Pick a random AWS region to test in
	awsRegion := aws.GetRandomRegion(t, nil, nil)

	// Create an EC2 KeyPair that we can use for SSH access
	keyPair := aws.CreateAndImportEC2KeyPair(t, awsRegion, uniqueName)

	// start the agent
	sshAgent := ssh.SshAgentWithKeyPair(t, keyPair.KeyPair)

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

		SshAgent: sshAgent,
	}

	return terraformOptions, keyPair, sshAgent
}
