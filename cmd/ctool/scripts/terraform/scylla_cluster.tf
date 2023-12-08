provider "aws" {
  region = "us-east-2"
}

resource "aws_vpc" "scylla_cluster_vpc" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = "scylla-vpc"
  }
}

resource "aws_placement_group" "scylla_placement_group" {
  name = "Scylla cluster group"
  strategy = "cluster"
}

resource "aws_subnet" "scylla_subnet" {
  vpc_id            = aws_vpc.scylla_cluster_vpc.id
  cidr_block        = "10.0.0.0/24"
  availability_zone = "us-east-2a"
  map_public_ip_on_launch = true
  tags = {
    Name = "scylla"
  }
}

resource "aws_network_interface" "node_00" {
  subnet_id   = aws_subnet.scylla_subnet.id
  private_ips = ["10.0.0.11"]
  security_groups = [aws_security_group.scylla_hosts.id]
  tags = {
    Name = "node_00_interface"
  }
}

resource "aws_network_interface" "node_01" {
  subnet_id   = aws_subnet.scylla_subnet.id
  private_ips = ["10.0.0.12"]
  security_groups = [aws_security_group.scylla_hosts.id]
  tags = {
    Name = "node_01_interface"
  }
}

resource "aws_network_interface" "node_02" {
  subnet_id   = aws_subnet.scylla_subnet.id
  private_ips = ["10.0.0.13"]
  security_groups = [aws_security_group.scylla_hosts.id]
  tags = {
    Name = "node_02_interface"
  }
}

resource "aws_network_interface" "node_03" {
  subnet_id   = aws_subnet.scylla_subnet.id
  private_ips = ["10.0.0.14"]
  security_groups = [aws_security_group.scylla_hosts.id]
  tags = {
    Name = "node_03_interface"
  }
}

resource "aws_network_interface" "node_04" {
  subnet_id   = aws_subnet.scylla_subnet.id
  private_ips = ["10.0.0.15"]
  security_groups = [aws_security_group.scylla_hosts.id]
  tags = {
    Name = "node_04_interface"
  }
}

resource "aws_network_interface" "node_instead_00" {
  subnet_id   = aws_subnet.scylla_subnet.id
  private_ips = ["10.0.0.16"]
  security_groups = [aws_security_group.scylla_hosts.id]
  tags = {
    Name = "node_instead_00_interface"
  }
}

resource "aws_network_interface" "node_instead_01" {
  subnet_id   = aws_subnet.scylla_subnet.id
  private_ips = ["10.0.0.17"]
  security_groups = [aws_security_group.scylla_hosts.id]
  tags = {
    Name = "node_instead_01_interface"
  }
}


resource "aws_instance" "node_00" {
  ami = "ami-0568936c8d2b91c4e"
  instance_type = "i3.large"
  root_block_device {
    volume_size = "10"
    volume_type = "gp2"
  }
  placement_group = aws_placement_group.scylla_placement_group.id
  network_interface {
    network_interface_id = aws_network_interface.node_00.id
    device_index         = 0
  }

  key_name = "amazonKey"

  tags = {
    Name = "node_00"
  }
}

module "instance_sshd_provisioners" {
  source           = "./modules/provision-sshd"

  ssh_private_key  = var.ssh_private_key
  ssh_private_ips  = [
      aws_instance.node_00.public_ip,
      aws_instance.node_01.public_ip,
      aws_instance.node_02.public_ip,
      aws_instance.node_03.public_ip,
      aws_instance.node_04.public_ip,
      aws_instance.node_instead_00.public_ip,
      aws_instance.node_instead_01.public_ip,
    ]

  depends_on = [
      aws_instance.node_00,
      aws_instance.node_01,
      aws_instance.node_02,
      aws_instance.node_03,
      aws_instance.node_04,
      aws_instance.node_instead_00,
      aws_instance.node_instead_01,
    ]
}

module "instance_ctool_provision" {
  source           = "./modules/provision-ctool"

  ssh_private_key  = var.ssh_private_key
  ctool_node = aws_instance.node_00.public_ip

  depends_on = [ module.instance_sshd_provisioners ]
}


output "public_ip_node_00" {
  value = aws_instance.node_00.public_ip
}

resource "aws_instance" "node_01" {
  ami = "ami-0568936c8d2b91c4e"
  instance_type = "i3.large"
  root_block_device {
    volume_size = "30"
    volume_type = "gp2"
  }
  placement_group = aws_placement_group.scylla_placement_group.id
  network_interface {
    network_interface_id = aws_network_interface.node_01.id
    device_index         = 0
  }
  key_name = "amazonKey"
}

output "public_ip_node_01" {
  value = aws_instance.node_01.public_ip
}


resource "aws_instance" "node_02" {
  ami = "ami-0568936c8d2b91c4e"
  instance_type = "i3.large"
  root_block_device {
    volume_size = "30"
    volume_type = "gp2"
  }
  placement_group = aws_placement_group.scylla_placement_group.id
  network_interface {
    network_interface_id = aws_network_interface.node_02.id
    device_index         = 0
  }
  key_name = "amazonKey"
}

output "public_ip_node_02" {
  value = aws_instance.node_02.public_ip
}


resource "aws_instance" "node_03" {
  ami = "ami-0568936c8d2b91c4e"
  instance_type = "i3.large"
  root_block_device {
    volume_size = "30"
    volume_type = "gp2"
  }
  placement_group = aws_placement_group.scylla_placement_group.id
  network_interface {
    network_interface_id = aws_network_interface.node_03.id
    device_index         = 0
  }
  key_name = "amazonKey"
}

output "public_ip_node_03" {
  value = aws_instance.node_03.public_ip
}

resource "aws_instance" "node_04" {
  ami = "ami-0568936c8d2b91c4e"
  instance_type = "i3.large"
  root_block_device {
    volume_size = "15"
    volume_type = "gp2"
  }
  placement_group = aws_placement_group.scylla_placement_group.id
  network_interface {
    network_interface_id = aws_network_interface.node_04.id
    device_index         = 0
  }
  key_name = "amazonKey"
}

output "public_ip_node_04" {
  value = aws_instance.node_04.public_ip
}

resource "aws_instance" "node_instead_00" {
  ami = "ami-0568936c8d2b91c4e"
  instance_type = "i3.large"
  root_block_device {
    volume_size = "30"
    volume_type = "gp2"
  }
  placement_group = aws_placement_group.scylla_placement_group.id
  network_interface {
    network_interface_id = aws_network_interface.node_instead_00.id
    device_index         = 0
  }
  key_name = "amazonKey"
}

output "public_ip_node_instead_00" {
  value = aws_instance.node_instead_00.public_ip
}

resource "aws_instance" "node_instead_01" {
  ami = "ami-0568936c8d2b91c4e"
  instance_type = "i3.large"
  root_block_device {
    volume_size = "30"
    volume_type = "gp2"
  }
  placement_group = aws_placement_group.scylla_placement_group.id
  network_interface {
    network_interface_id = aws_network_interface.node_instead_01.id
    device_index         = 0
  }
  key_name = "amazonKey"
}

output "public_ip_node_instead_01" {
  value = aws_instance.node_instead_01.public_ip
}



resource "aws_internet_gateway" "gw" { vpc_id = aws_vpc.scylla_cluster_vpc.id }

resource "aws_route_table" "cluster_route" {
  vpc_id = aws_vpc.scylla_cluster_vpc.id
  tags = {
    Name = "Cluster route table"
  }
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.gw.id
  }
}

resource "aws_route_table_association" "private" {
  subnet_id      = aws_subnet.scylla_subnet.id
  route_table_id = aws_route_table.cluster_route.id
}
