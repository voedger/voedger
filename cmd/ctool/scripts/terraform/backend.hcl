terraform {
  backend "s3" {
    bucket = "ctool-terraform-state-bucket"
    key    = "terraform.tfstate"
    region = "us-east-2"
  }
}