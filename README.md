# Terraform OpenVPN Module

This terraform module deploys a very minimal OpenVPN to AWS. Intended to be cheap and disposable, so there are no niceties such as route53 records or http interface. It's currently based off of the [kylemanna/docker-openvpn](http://github.com/kylemanna/docker-openvpn) docker image, but this is configurable.
You can and you should rebuild this image from scratch, since you can never trust a public registry.

# Example 

Setup a basic terraform file:

```
# main.tf

provider "aws" {
  profile = "default"
  region  = "eu-west-1"
}

"aws_key_pair" "deployer" {
  key_name   = "deployer"
  public_key = "${file("/home/josh/.ssh/deployer.pub")}"
}

module "openvpn" {
  source            = "github.com/joshpurvis/terraform-openvpn?ref=0.0.1"
  aws_region        = "eu-west-1"
  aws_key_pair_name = "${aws_key_pair.deployer.key_name}"

  # optional
  #client_name      = "terraform-openvpn-client"   # useful with multiple VPNs in various regions
  #docker_image     = "kylemanna/openvpn"          # you should never trust public docker images, so rebuild this yourself and put your registry here
}
```

Which can be deployed and provisioned with:

```
terraform init
terraform apply
```

After which a file called `terraform-openvpn-client.ovpn` will be copied beside `main.tf`.

## Configure your local client using Network Manager

If you would like it to be available in your network settings, and you're using Network Manager, you can import it like this:

```
sudo nmcli connection import type openvpn file terraform-openvpn-client.ovpn
sudo nmcli connection up terraform-openvpn-client
```

## Configure your local client manually

```
openvpn terraform-openvpn-client.ovpn
```

# License

MIT License. Please see [LICENSE](/LICENSE) file for further details.


