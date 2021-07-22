locals {
  prefix              = var.cluster_id
  port_kubernetes_api = 6443
  port_machine_config = 22623
}

############################################
# Bootstrap node
############################################

resource "ibm_is_instance" "bootstrap_node" {
  name           = "${local.prefix}-bootstrap"
  image          = var.vsi_image_id
  profile        = var.vsi_profile
  resource_group = var.resource_group_id
  tags           = var.tags

  primary_network_interface {
    name            = "eth0"
    subnet          = var.subnet_id
    security_groups = concat(var.security_group_id_list, [ibm_is_security_group.bootstrap.id])
  }

  vpc  = var.vpc_id
  zone = var.zone
  keys = []

  # Use custom ignition config that pulls content from COS bucket
  # TODO: Once support for the httpHeaders field is added to
  # terraform-provider-ignition, we should use it instead of this template.
  # https://github.com/community-terraform-providers/terraform-provider-ignition/issues/16
  user_data = templatefile("${path.module}/templates/bootstrap.ign", {
    HOSTNAME    = ibm_cos_bucket.bootstrap_ignition.s3_endpoint_public
    BUCKET_NAME = ibm_cos_bucket.bootstrap_ignition.bucket_name
    OBJECT_NAME = ibm_cos_bucket_object.bootstrap_ignition.key
    IAM_TOKEN   = data.ibm_iam_auth_token.iam_token.iam_access_token
  })
}

############################################
# Floating IP
############################################

resource "ibm_is_floating_ip" "bootstrap_floatingip" {
  count = var.public_endpoints ? 1 : 0

  name           = "${local.prefix}-bootstrap-node-ip"
  resource_group = var.resource_group_id
  target         = ibm_is_instance.bootstrap_node.primary_network_interface.0.id
  tags           = var.tags
}

############################################
# Security group
############################################

resource "ibm_is_security_group" "bootstrap" {
  name           = "${local.prefix}-security-group-bootstrap"
  resource_group = var.resource_group_id
  tags           = var.tags
  vpc            = var.vpc_id
}

# SSH
resource "ibm_is_security_group_rule" "bootstrap_ssh_inbound" {
  group     = ibm_is_security_group.bootstrap.id
  direction = "inbound"
  remote    = var.public_endpoints ? "0.0.0.0/0" : var.security_group_id_list.0.id
  tcp {
    port_min = 22
    port_max = 22
  }
}

############################################
# Load balancer backend pool members
############################################

resource "ibm_is_lb_pool_member" "kubernetes_api_public" {
  count = var.public_endpoints ? 1 : 0

  lb             = var.lb_kubernetes_api_public_id
  pool           = var.lb_pool_kubernetes_api_public_id
  port           = local.port_kubernetes_api
  target_address = ibm_is_instance.bootstrap_node.primary_network_interface.0.primary_ipv4_address
}

resource "ibm_is_lb_pool_member" "kubernetes_api_private" {
  lb             = var.lb_kubernetes_api_private_id
  pool           = var.lb_pool_kubernetes_api_private_id
  port           = local.port_kubernetes_api
  target_address = ibm_is_instance.bootstrap_node.primary_network_interface.0.primary_ipv4_address
}

resource "ibm_is_lb_pool_member" "machine_config" {
  lb             = var.lb_kubernetes_api_private_id
  pool           = var.lb_pool_machine_config_id
  port           = local.port_machine_config
  target_address = ibm_is_instance.bootstrap_node.primary_network_interface.0.primary_ipv4_address
}
