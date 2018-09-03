package test

import (
	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/gruntwork-io/terratest/modules/test-structure"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOpenVPNInstance(t *testing.T) {
	t.Parallel()

	terraformDirectory := "../"

	defer test_structure.RunTestStage(t, "teardown", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, terraformDirectory)
		ec2KeyPair := test_structure.LoadEc2KeyPair(t, terraformDirectory)
		terraform.Destroy(t, terraformOptions)
		aws.DeleteEC2KeyPair(t, ec2KeyPair)
	})

	test_structure.RunTestStage(t, "setup", func() {

		// setup basic terraform options, generate ec2 keypair, and sshAgent
		terraformOptions, ec2KeyPair, sshAgent := configureTerraformOptions(t, terraformDirectory)
		defer sshAgent.Stop()

		// save for later steps
		test_structure.SaveTerraformOptions(t, terraformDirectory, terraformOptions)
		test_structure.SaveEc2KeyPair(t, terraformDirectory, ec2KeyPair)

		terraform.InitAndApply(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "validate", func() {

		// load options
		terraformOptions := test_structure.LoadTerraformOptions(t, terraformDirectory)
		awsRegion := terraformOptions.Vars["aws_region"].(string)
		instanceId := terraform.Output(t, terraformOptions, "instance_id")
		publicIP := terraform.Output(t, terraformOptions, "public_ip")
		uniqueName := test_structure.LoadString(t, terraformDirectory, "uniqueName")

		// confirm that random name was properly applied as a tag
		instanceTags := aws.GetTagsForEc2Instance(t, awsRegion, instanceId)
		nameTag, containsNameTag := instanceTags["Name"]
		assert.True(t, containsNameTag)
		assert.Equal(t, uniqueName, nameTag)

		// test the ovpn file by actually running openvpn via docker container
		openvpnConnectionTest(t, terraformDirectory, publicIP)

	})



}
