output "ip_addresses" {
  value = [for m in aws_network_interface.master : tolist(m.private_ips)[0]]
}

output "ids" {
  value = [for m in aws_instance.master : m.id]
}