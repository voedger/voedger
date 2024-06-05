# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
# @date 2023-12-07

variable "ssh_private_ips" {
  description = "Private node IPs"
  type        = list(string)
  default     = []  # Make it an empty list by default
}

variable "ssh_private_key" {}


resource "null_resource" "configure_ssh" {
  count = length(var.ssh_private_ips)

  provisioner "remote-exec" {
    inline = [
      "echo '${var.ssh_private_key}' > /tmp/amazonKey.pem",
      "chmod 600 /tmp/amazonKey.pem",
      "if [ -f /etc/ssh/sshd_config ] && grep -q 'Port 22' /etc/ssh/sshd_config; then",
      "  sudo sed -i '/Port 22/ s/.*/Port 2214/' /etc/ssh/sshd_config",
      "  sudo systemctl restart ssh",
      "fi"
    ]

    connection {
      type        = "ssh"
      user        = "ubuntu"
      private_key = "${var.ssh_private_key}"
      host        = "${var.ssh_private_ips[count.index]}"
      agent       = false
    }
  }
}
