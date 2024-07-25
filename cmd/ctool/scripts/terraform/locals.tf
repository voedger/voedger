locals {
  node_count = length(var.included_nodes)
  node_configurations = {
	    "node_00"         = { private_ip = "10.0.0.11" }
	    "node_01"         = { private_ip = "10.0.0.12" }
	    "node_02"         = { private_ip = "10.0.0.13" }
	    "node_03"         = { private_ip = "10.0.0.14" }
	    "node_04"         = { private_ip = "10.0.0.15" }
	    "node_instead_00" = { private_ip = "10.0.0.16" }
	    "node_instead_01" = { private_ip = "10.0.0.17" }
  }
}
