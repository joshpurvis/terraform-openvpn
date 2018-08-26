# Required Inputs
variable "aws_key_pair_name" {}

# Defaults
variable "client_name" {
  default = "terraform-openvpn-client"
}

variable "instance_name" {
  default = "openvpn"
}

variable "docker_image" {
  default = "kylemanna/openvpn"
}

variable "aws_region" {
  default = "eu-west-1"
}

variable "instance_type" {
  default = "t2.nano"
}

variable "ssh_user" {
  default = "ubuntu"
}

variable "ssh_port" {
  default = 22
}

variable "ssh_cidr" {
  default = "0.0.0.0/0"
}

variable "openvpn_cidr" {
  default = "0.0.0.0/0"
}

variable "vpc_cidr" {
  default = "10.1.0.0/16"
}

variable "subnet_cidr" {
  default = "10.1.1.0/24"
}

