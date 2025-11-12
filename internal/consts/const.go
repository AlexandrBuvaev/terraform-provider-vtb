package consts

const CLOUD_EXTRA_MOUNT_MAX_SIZE = 2048

const (
	CREATE_RES_FAIL  = "Resource creation failed:"
	DELETE_RES_FAIL  = "Resource destruction failed:"
	UPDATE_RES_FAIL  = "Resource update-in-place failed:"
	READ_RES_FAIL    = "Resource reading failed:"
	VALIDATION_FAIL  = "Config validation failed:"
	MODIFY_PLAN_FAIL = "Modify plan failed:"
)

var AVAILABILITY_ZONES = []string{
	"msk-north",
	"msk-east",
	"msk-t1",
}

var DOMAINS = []string{
	"corp.dev.vtb",
	"test.vtb.ru",
	"region.vtb.ru",
	"nova.nb",
}

var PLATFORMS = []string{
	"OpenStack",
	"VMware vSphere",
	"Nutanix",
	"ceph",
}
