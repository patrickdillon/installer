locals {
  prefix              = var.cluster_id
  port_kubernetes_api = 6443
  port_machine_config = 22623
  subnet_count        = length(var.subnet_id_list)
  zone_count          = length(var.zone_list)
}

############################################
# Master nodes
############################################

resource "ibm_is_instance" "master_node" {
  count = var.instance_count
  depends_on = [
    var.lb_kubernetes_api_private_id,
    var.lb_kubernetes_api_public_id
  ]

  name           = "${local.prefix}-master-${count.index}"
  image          = var.vsi_image_id
  profile        = var.vsi_profile
  resource_group = var.resource_group_id
  tags           = var.tags

  primary_network_interface {
    name            = "eth0"
    subnet          = var.subnet_id_list[count.index % local.subnet_count]
    security_groups = var.security_group_id_list
  }

  vpc  = var.vpc_id
  zone = var.zone_list[count.index % local.zone_count]
  keys = []

  user_data = var.ignition
}

############################################
# Load balancer backend pool members
############################################

resource "ibm_is_lb_pool_member" "kubernetes_api_public" {
  count = var.public_endpoints ? var.instance_count : 0

  lb             = var.lb_kubernetes_api_public_id
  pool           = var.lb_pool_kubernetes_api_public_id
  port           = local.port_kubernetes_api
  target_address = ibm_is_instance.master_node[count.index].primary_network_interface.0.primary_ipv4_address
}

resource "ibm_is_lb_pool_member" "kubernetes_api_private" {
  count = var.instance_count

  lb             = var.lb_kubernetes_api_private_id
  pool           = var.lb_pool_kubernetes_api_private_id
  port           = local.port_kubernetes_api
  target_address = ibm_is_instance.master_node[count.index].primary_network_interface.0.primary_ipv4_address
}

resource "ibm_is_lb_pool_member" "machine_config" {
  count = var.instance_count

  lb             = var.lb_kubernetes_api_private_id
  pool           = var.lb_pool_machine_config_id
  port           = local.port_machine_config
  target_address = ibm_is_instance.master_node[count.index].primary_network_interface.0.primary_ipv4_address
}