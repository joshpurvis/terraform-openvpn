#!/bin/bash
trap "kill 0" SIGINT

REGION=eu-west-1
CLIENT_NAME=terraform-openvpn-${REGION}
DEPLOYER_KEY=deployer

# create a temporary ssh key which we'll destroy in a minute
ssh-keygen -t rsa -C "`hostname`" -f ${DEPLOYER_KEY} -P ""
ssh-add ${DEPLOYER_KEY}

terraform init
terraform apply -var "aws_region=${REGION}"

# get public IP of the instance
PUBLIC_IP=$(terraform output public_ip)
CLIENT_NAME=$(terraform output openvpn_config)

# add to known hosts
ssh-keygen -R ${PUBLIC_IP} && ssh-keyscan -H ${PUBLIC_IP} >> ~/.ssh/known_hosts

sudo openvpn ${CLIENT_NAME}

# remove deployer key
ssh-add -D ${DEPLOYER_KEY}
rm ${DEPLOYER_KEY} ${DEPLOYER_KEY}.pub

# teardown
terraform destroy