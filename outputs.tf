output "public_dns" {
  value = "${aws_instance.openvpn.public_dns}"
}

output "openvpn_config" {
  value = "${var.client_name}"
}
