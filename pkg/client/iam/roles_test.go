package iam

import (
	"fmt"
	"log"
	"terraform-provider-vtb/pkg/client/test"
	"testing"
)

func TestGetRoles(t *testing.T) {
	roles, err := GetRoles(test.SharedCreds)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("roles: %v+", roles)
}

func TestGetAvailableServiceRoles(t *testing.T) {
	avaliableRoles, err := GetAvailableServiceRoles(test.SharedCreds, "resource-manager/proj-e73127g7ry3p4t4")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("avaliable_roles: %v+", avaliableRoles)
}

func TestGetRoleByName(t *testing.T) {
	role, err := GetRoleByName(test.SharedCreds, "organizations/vtb/roles/terraform-test")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("role: %v+", role)
}

func TestGetOrganizationRoles(t *testing.T) {
	oranizationRoles, err := GetOrganizationRoles(test.SharedCreds, "vtb")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Roles: %v+", oranizationRoles)
}

func TestDeleteRole(t *testing.T) {
	err := DeleteRole(test.SharedCreds, "vtb", "organizations/vtb/roles/terraform-test")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Delete success!")
}

func TestCreateRole(t *testing.T) {
	roleAttrs := &CreateRoleAttrs{
		Name:        "terraform-test-2",
		Title:       "terraform-test-2",
		Description: "Тестовая роль",
		Permissions: []string{"accountmanager:accounts:get"},
	}
	err := CreateRole(test.SharedCreds, "vtb", *roleAttrs)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Role create success")
}

func TestUpdateRole(t *testing.T) {
	roleAttrs := &UpdateRoleAttrs{
		Title:       "terraform-test-2",
		Description: "Обновленная тестовая роль",
		Permissions: []string{"accountmanager:accounts:get"},
	}

	err := UpdateRolePermissions(test.SharedCreds, "vtb", "organizations/vtb/roles/terraform-test", *roleAttrs)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Role update success")
}
