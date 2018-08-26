output "public_dns" {
  value = "${aws_instance.openvpn.public_dns}"
}

output "public_ip" {
  value = "${aws_instance.openvpn.public_ip}"
}

output "openvpn_config" {
  value = "${var.client_name}"
}

output "instance_id" {
  value = "${aws_instance.openvpn.id}"
}
