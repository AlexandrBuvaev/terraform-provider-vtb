data "vtb_s3_ceph_image_data" "nvme" {       // PROD only
    storage_type    = "nvme"
}

data "vtb_s3_ceph_image_data" "backup" {     // PROD only
    storage_type    = "backup"
}

data "vtb_s3_ceph_image_data" "hdd" {
    storage_type    = "hdd"
}