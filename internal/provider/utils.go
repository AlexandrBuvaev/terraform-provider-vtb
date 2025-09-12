package provider

import (
	"context"
	"crypto/rand"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"

	"github.com/Masterminds/semver"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	lowercaseLetters = "abcdefghijklmnopqrstuvwxyz"
	uppercaseLetters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits           = "0123456789"
	specialChars     = "@$!%*?&"
	allChars         = lowercaseLetters + uppercaseLetters + digits + specialChars
	minLength        = 16
)

func GetMinorVersion(version string) string {
	return strings.Split(version, ".")[1]
}

func GenerateSecurePassword(length int) string {
	if length <= 15 {
		length = minLength
	}
	// Создаем слайс для хранения символов пароля
	password := make([]byte, length)

	// Гарантированно добавляем по одному символу каждого типа
	password[0] = secureRandomChar(lowercaseLetters)
	password[1] = secureRandomChar(uppercaseLetters)
	password[2] = secureRandomChar(digits)
	password[3] = secureRandomChar(specialChars)

	// Заполняем оставшуюся часть пароля случайными символами
	for i := 4; i < length; i++ {
		password[i] = secureRandomChar(allChars)
	}

	// Перемешиваем символы в пароле
	// Используем crypto/rand для перемешивания
	for i := len(password) - 1; i > 0; i-- {
		j := secureRandomInt(i + 1)
		password[i], password[j] = password[j], password[i]
	}

	return string(password)
}

// Функция для генерации случайного целого числа в диапазоне [0, max)
func secureRandomInt(max int) int {
	b := make([]byte, 1)
	for {
		_, err := rand.Read(b)
		if err != nil {
			panic(err)
		}
		if int(b[0]) < max {
			return int(b[0])
		}
	}
}

// Функция для выбора случайного символа из строки с использованием crypto/rand
func secureRandomChar(chars string) byte {
	n := len(chars)
	b := make([]byte, 1)
	for {
		// Генерируем случайное число в диапазоне [0, n)
		_, err := rand.Read(b)
		if err != nil {
			panic(err) // Если произошла ошибка, завершаем программу
		}
		if int(b[0]) < n {
			return chars[int(b[0])]
		}
	}
}

func findIndexInSlice(target string, slice []string) int {
	index := -1
	for i, value := range slice {
		if value == target {
			index = i
			break
		}
	}
	return index
}

func selectPlatform(platform string) string {

	if platform == "vsphere" {
		return "VMware vSphere"
	} else {
		return "OpenStack"
	}
}

func readAccessMapV2(vmACLs []entities.AccessACL) map[string][]types.String {
	var accessMap map[string][]types.String = make(map[string][]types.String)
	for _, access := range vmACLs {
		_, ok := accessMap[access.Role]
		if !ok {
			accessMap[access.Role] = make([]types.String, 0)
		}
		for _, group := range access.Members {
			exists := false
			for _, existsGroup := range accessMap[access.Role] {
				if group == existsGroup.ValueString() {
					exists = true
					break
				}
			}
			if !exists {
				accessMap[access.Role] = append(accessMap[access.Role], types.StringValue(group))
			}
		}
	}
	return accessMap
}

func readAccessMapVV1(vmACLs []entities.AccessACL) map[string][]string {
	var accessMap map[string][]string = make(map[string][]string)
	for _, acl := range vmACLs {
		if _, ok := accessMap[acl.Role]; !ok {
			accessMap[acl.Role] = acl.Members
		} else {
			for _, member := range acl.Members {
				if !slices.Contains(accessMap[acl.Role], member) {
					accessMap[acl.Role] = append(accessMap[acl.Role], member)
				}
			}
		}
	}
	return accessMap
}

func prepareADLogonGrants(access map[string][]types.String) []entities.ADLogonGrants {
	ADLogonGrants := []entities.ADLogonGrants{}
	for role, groups := range access {
		var groupsNames []string
		for _, group := range groups {
			groupsNames = append(groupsNames, group.ValueString())
		}
		ADLogonGrants = append(ADLogonGrants, entities.ADLogonGrants{
			Role:   role,
			Groups: groupsNames,
		})
	}
	return ADLogonGrants
}

func prepareExtraMountsAttrs(planExtraMount map[string]ExtraMountModel) []entities.ExtraMount {
	var extraMounts []entities.ExtraMount
	for path, mount := range planExtraMount {
		extraMounts = append(extraMounts, entities.ExtraMount{
			Path:       path,
			Size:       mount.Size.ValueInt64(),
			FileSystem: mount.FileSystem.ValueString(),
		})
	}
	return extraMounts
}

func prepareBasicAttrs(
	flavor *FlavorModel,
	core *CoreModel,
	access map[string][]types.String,
	planExtraMounts map[string]ExtraMountModel,
	OsVersion string,
	ADIntegration bool,
	OnSupport bool,
) orders.BasicAttrs {
	ba := orders.BasicAttrs{
		ADIntegration:    ADIntegration,
		ADLogonGrants:    prepareADLogonGrants(access),
		ExtraMounts:      prepareExtraMountsAttrs(planExtraMounts),
		OnSupport:        OnSupport,
		OsVersion:        OsVersion,
		AvailabilityZone: core.Zone.ValueString(),
		Domain:           core.Domain.ValueString(),
		Platform:         core.Platform.ValueString(),
		DefaultNic: entities.DefaultNic{
			NetSegment: core.NetSegmentCode.ValueString(),
		},
		Flavor: entities.Flavor{
			Cores:  flavor.Cores.ValueInt64(),
			Memory: flavor.Memory.ValueInt64(),
			Name:   flavor.Name.ValueString(),
			UUID:   flavor.UUID.ValueString(),
		},
		CreatedWithOpenTofu: true,
	}
	// if finProjectID != "" {
	// 	ba.FinProjectID = finProjectID
	// }

	return ba
}

type OrderInterface interface {
	ChangeLabel(label string) error
}

func changeOrderLabel(order OrderInterface, label string, resp *resource.UpdateResponse) {
	err := order.ChangeLabel(label)
	if err != nil {
		resp.Diagnostics.AddError(
			"Change order label",
			fmt.Sprintf(
				"Changing order label ended with error.\nError message: %s",
				err.Error(),
			),
		)
		return
	}
}

func extractRabbitMQNumber(layoutName string) (int, error) {
	re := regexp.MustCompile(`rabbitmq-(\d+)`)
	matches := re.FindStringSubmatch(layoutName)
	if len(matches) < 2 {
		return 0, fmt.Errorf("the number was not found in the string")
	}
	return strconv.Atoi(matches[1])
}

func validateRabbitMQCount(currentLayout, newLayout string) error {
	currentCount, err := extractRabbitMQNumber(currentLayout)
	if err != nil {
		return fmt.Errorf("error when extracting the current quantity: %v", err)
	}
	newCount, err := extractRabbitMQNumber(newLayout)
	if err != nil {
		return fmt.Errorf("error when extracting a new quantity: %v", err)
	}
	if newCount < currentCount {
		return fmt.Errorf("scaling is not available: it is not possible to increase the number of rabbitmq in a smaller direction")
	}
	return nil
}

func isADLogonGrantsEqual(plan, state map[string][]types.String) bool {
	if len(plan) != len(state) {
		return false
	}
	for role, planGroups := range plan {
		if _, ok := state[role]; !ok {
			return false
		}
		if len(planGroups) != len(state[role]) {
			return false
		}
		for _, grp := range planGroups {
			if !slices.Contains(state[role], grp) {
				return false
			}
		}

	}
	return true
}

func isVersionOlder(current, latest string) (bool, error) {
	currentVersion, err := semver.NewVersion(current)
	if err != nil {
		return false, fmt.Errorf("error parsing current version (%s): %w", current, err)
	}

	latestVersion, err := semver.NewVersion(latest)
	if err != nil {
		return false, fmt.Errorf("error parsing latest version (%s): %w", latest, err)
	}

	return currentVersion.LessThan(latestVersion), nil
}

func containsDuplicate(names []string) bool {
	if len(names) <= 1 {
		return false
	}

	sort.Strings(names)

	for i := 0; i < len(names)-1; i++ {
		if names[i] == names[i+1] {
			return true
		}
	}
	return false
}

func convertSetToList(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if set.IsNull() || set.IsUnknown() {
		return nil, diags
	}

	elements := make([]string, 0, len(set.Elements()))
	diags.Append(set.ElementsAs(ctx, &elements, false)...)
	return elements, diags
}

func sliceDifference(new, old []string) []string {
	oldSet := make(map[string]struct{}, len(old))
	for _, v := range old {
		oldSet[v] = struct{}{}
	}

	var diff []string
	for _, v := range new {
		if _, found := oldSet[v]; !found {
			diff = append(diff, v)
		}
	}
	return diff
}

func extractZoneNumber(zoneName string) (int64, error) {
	parts := strings.Split(zoneName, "-")
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid zone name format, zoneName: %s", zoneName)
	}
	return strconv.ParseInt(parts[1], 10, 64)
}

func convertAccessMap(tfMap map[string][]types.String) map[string][]string {
	result := make(map[string][]string)

	for key, tfStrings := range tfMap {
		strSlice := make([]string, len(tfStrings))
		for i, tfStr := range tfStrings {
			strSlice[i] = tfStr.ValueString()
		}
		result[key] = strSlice
	}

	return result
}

func compareSlices(oldSlice, newSlice []string) (toAdd, toDelete []string) {
	oldSet := make(map[string]struct{})
	newSet := make(map[string]struct{})

	for _, item := range oldSlice {
		oldSet[item] = struct{}{}
	}
	for _, item := range newSlice {
		newSet[item] = struct{}{}
	}

	for item := range newSet {
		if _, exists := oldSet[item]; !exists {
			toAdd = append(toAdd, item)
		}
	}

	for item := range oldSet {
		if _, exists := newSet[item]; !exists {
			toDelete = append(toDelete, item)
		}
	}

	return toAdd, toDelete
}

func RetryWithExponentialBackoff(attempts int, initialDelay time.Duration, fn func() error) (int, error) {
	var err error

	for attempt := 0; attempt < attempts; attempt++ {
		err = fn()
		if err == nil {
			return attempt + 1, nil
		}

		if attempt < attempts-1 {
			currentDelay := time.Duration(attempt+1) * initialDelay
			time.Sleep(currentDelay)
		}
	}

	return attempts, err
}

// добавление элементов в types.Set
func appendToSet(ctx context.Context, set basetypes.SetValue, values ...string) (basetypes.SetValue, diag.Diagnostics) {
	var diags diag.Diagnostics
	var existingValues []string

	if !set.IsNull() && !set.IsUnknown() {
		diags = set.ElementsAs(ctx, &existingValues, false)
		if diags.HasError() {
			return set, diags
		}
	}

	existingValues = append(existingValues, values...)
	return types.SetValueFrom(ctx, types.StringType, existingValues)
}
