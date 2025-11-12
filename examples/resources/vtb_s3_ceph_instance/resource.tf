data "vtb_s3_ceph_image_data" "nvme" {
  storage_type = "hdd"
}

resource "vtb_s3_ceph_instance" "tenant" {
  zone              = "msk-north"
  net_segment       = "dev-srv-app"
  financial_project = var.financial_project
  label             = var.label
  image             = data.vtb_s3_ceph_image_data.nvme
  users = {
    "user-test" = {
      access_key = "ZIO2SIG50S4M7VB3QY64WSZFP"
      secret_key = "9neKkHPuV3G8gN6TMWWS6sNTadqwBaf6Pbo2UQ7Q1qFaC"
    },
    "user-test2" = {
      access_key = "ZIO2SIG504MX7VB3QY64WSSZFP"
      secret_key = "9neKkHPuV3G82N6TMWW6NTssadqwBaf6Pbo2UQ7Q1qFaC"
    }
  }
  buckets = {
    "d5-soub-bucket-test" = {
      max_size_gb = 20
      versioning  = false
    }
    "d5-soub-bucket-test2" = {
      max_size_gb = 10
      versioning  = true
    }
  }
}


variable "label" {
  default = "S3 Ceph tenant tf"
}

variable "financial_project" {
  default = "VTB.Cloud"
}
