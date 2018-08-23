# Terraform OpenVPN Module

This terraform module deploys a very minimal OpenVPN to AWS. Intended to be cheap, disposable, and short lived 
-- so there's no niceties such as route53 records or http interface.

# Example terraform file:

```
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
  client_name       = "terraform-openvpn-eu-west-1"  # useful when importing resulting ovpn file into NetworkManager
}
```

Which can be deployed with:

```
terraform init
terraform apply
```


# License

MIT License. Please see [LICENSE](/LICENSE) file for further details.


