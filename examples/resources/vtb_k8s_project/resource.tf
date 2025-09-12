resource "vtb_k8sproject_instance" "name" {
    label             = "Kubernetes project"
    net_segment       = "dev-srv-app"
    data_center       = "7"
    quota             = {
        cpu    = 1
        memory = 1
    }
    cluster_name      = "dk7-soub45"
    project_name      = "test-project"
    ingress           = "ingress-dk7-soub45-soub-001"
    region            = "region-dk7-soub45-soub-001"
    access            = {
        "edit" = [
            "cloud-soub-k8s-test"
        ]
    }
    lifetime          = 30
    financial_project = "VTB.Cloud"

    chaos_mesh        = {}
    istio = {
        control_plane = "dk7-soub45-istio-system-001"
        roles = [
            {
                role = "custom-istio-edit"
                groups = [
                    "cloud-soub-k8s-test"
                ]
            }
        ]
    }
    tyk = {
        roles = [
            {
              role = "tyk-gateway-viewer-role"
              groups = [
                "cloud-soub-k8s-test"
              ]
            }
          ]
    }
    tsam_operator = {
          roles = []
    }
    tslg_operator = {
        roles = []
    }
    omni_certificates = [
        {
            app_name = "kafka"
            client_name = "cert"
        }
    ]
} 