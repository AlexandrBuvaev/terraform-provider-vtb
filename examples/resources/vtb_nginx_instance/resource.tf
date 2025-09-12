data "vtb_core_data" "dev" {
  platform    = "OpenStack"       
  domain      = "corp.dev.vtb"   
  net_segment = "dev-srv-app"    
  zone        = "msk-north"      
}


data "vtb_flavor_data" "c2m4" {
  cores  = 2  
  memory = 4 
}

data "vtb_nginx_image_data" "name" {
  distribution = "astra"  
  os_version   = "1.7"    
}

resource "vtb_nginx_instance" "name" {
  label             = "TerraformNginx Astra"
  core              = data.vtb_core_data.dev
  flavor            = data.vtb_flavor_data.c2m4
  image             = data.vtb_nginx_image_data.name
  nginx_version     = "1.22.0"
  financial_project = "VTB.Cloud"
  extra_mounts = {
    "/app" = {
      size = 30
    },
  }
  access = {
    "user" = [
      "cloud-soub-developers1",
      "cloud-soub-developers2",
    ]
  }
}


