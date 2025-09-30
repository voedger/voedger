variable "ssh_private_key" {}
variable "gh_token" {}
variable "git_commit_id" {}
variable "git_repo_url" {}
variable "ssh_port" {}
variable "run_id" {}
variable "included_nodes" {
   type = list(string)
}