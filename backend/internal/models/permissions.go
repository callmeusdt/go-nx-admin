package models

import "strings"

type PermissionDef struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Group string `json:"group"`
}

var ExtraPermissionDefs []PermissionDef
var ExtraPermissionExpand func(string) [][]string

var PermissionDefs = []PermissionDef{
	{Key: "dashboard:view", Name: "查看仪表盘", Group: "仪表盘"},

	{Key: "profile:view", Name: "查看个人信息", Group: "个人中心"},
	{Key: "profile:edit", Name: "修改个人信息", Group: "个人中心"},
	{Key: "mfa:manage", Name: "管理MFA验证", Group: "个人中心"},

	{Key: "users:list", Name: "查看用户", Group: "用户管理"},
	{Key: "users:create", Name: "新增用户", Group: "用户管理"},
	{Key: "users:edit", Name: "编辑用户", Group: "用户管理"},
	{Key: "users:delete", Name: "删除用户", Group: "用户管理"},
	{Key: "users:mfa", Name: "清除用户MFA", Group: "用户管理"},
	{Key: "users:whitelist", Name: "管理IP白名单", Group: "用户管理"},

	{Key: "roles:list", Name: "查看角色", Group: "角色管理"},
	{Key: "roles:create", Name: "新增角色", Group: "角色管理"},
	{Key: "roles:edit", Name: "编辑角色", Group: "角色管理"},
	{Key: "roles:delete", Name: "删除角色", Group: "角色管理"},
	{Key: "roles:menus", Name: "菜单授权", Group: "角色管理"},
	{Key: "roles:permissions", Name: "权限分配", Group: "角色管理"},

	{Key: "menus:list", Name: "查看菜单", Group: "菜单管理"},
	{Key: "menus:create", Name: "新增菜单", Group: "菜单管理"},
	{Key: "menus:edit", Name: "编辑菜单", Group: "菜单管理"},
	{Key: "menus:delete", Name: "删除菜单", Group: "菜单管理"},

	{Key: "apis:list", Name: "查看接口", Group: "接口管理"},
	{Key: "apis:create", Name: "新增接口", Group: "接口管理"},
	{Key: "apis:edit", Name: "编辑接口", Group: "接口管理"},
	{Key: "apis:delete", Name: "删除接口", Group: "接口管理"},

	{Key: "audit:view", Name: "操作日志", Group: "日志审计"},
	{Key: "logins:view", Name: "登录日志", Group: "日志审计"},

	{Key: "online:list", Name: "查看在线用户", Group: "在线用户"},
	{Key: "online:kick", Name: "踢下线", Group: "在线用户"},

	{Key: "media:list", Name: "查看媒体", Group: "媒体管理"},
	{Key: "media:upload", Name: "上传媒体", Group: "媒体管理"},
	{Key: "media:delete", Name: "删除媒体", Group: "媒体管理"},

	{Key: "config:view", Name: "查看配置", Group: "系统配置"},
	{Key: "config:edit", Name: "修改配置", Group: "系统配置"},
}

func ExpandPermission(key string) [][]string {
	switch key {
	case "dashboard:view":
		return [][]string{{"GET", "/api/v1/dashboard/stats"}}

	// 个人中心
	case "profile:view":
		return [][]string{{"GET", "/api/v1/auth/me"}}
	case "profile:edit":
		return [][]string{{"PUT", "/api/v1/auth/profile"}}
	case "mfa:manage":
		return [][]string{
			{"GET", "/api/v1/auth/my-ip"},
			{"GET", "/api/v1/auth/menus"},
			{"POST", "/api/v1/auth/mfa/setup"},
			{"POST", "/api/v1/auth/mfa/enable"},
			{"POST", "/api/v1/auth/mfa/disable"},
			{"POST", "/api/v1/auth/mfa/recovery-codes"},
		}

	// 用户管理
	case "users:list":
		return [][]string{{"GET", "/api/v1/users"}}
	case "users:create":
		return [][]string{{"POST", "/api/v1/users"}}
	case "users:edit":
		return [][]string{{"PUT", "/api/v1/users/.*"}}
	case "users:delete":
		return [][]string{{"DELETE", "/api/v1/users/.*"}}
	case "users:mfa":
		return [][]string{{"DELETE", "/api/v1/users/.*/mfa"}}
	case "users:whitelist":
		return [][]string{
			{"GET", "/api/v1/users/.*/ip-whitelists"},
			{"POST", "/api/v1/users/.*/ip-whitelists"},
			{"PUT", "/api/v1/users/.*/ip-whitelists/.*"},
			{"DELETE", "/api/v1/users/.*/ip-whitelists/.*"},
		}

	// 角色管理
	case "roles:list":
		return [][]string{{"GET", "/api/v1/roles"}}
	case "roles:create":
		return [][]string{{"POST", "/api/v1/roles"}}
	case "roles:edit":
		return [][]string{{"PUT", "/api/v1/roles/.*"}}
	case "roles:delete":
		return [][]string{{"DELETE", "/api/v1/roles/.*"}}
	case "roles:menus":
		return [][]string{
			{"GET", "/api/v1/roles/.*/menus"},
			{"PUT", "/api/v1/roles/.*/menus"},
		}
	case "roles:permissions":
		return [][]string{
			{"GET", "/api/v1/roles/.*/policies"},
			{"PUT", "/api/v1/roles/.*/policies"},
			{"GET", "/api/v1/roles/.*/permissions"},
			{"PUT", "/api/v1/roles/.*/permissions"},
		}

	// 菜单管理
	case "menus:list":
		return [][]string{{"GET", "/api/v1/menus"}}
	case "menus:create":
		return [][]string{{"POST", "/api/v1/menus"}}
	case "menus:edit":
		return [][]string{{"PUT", "/api/v1/menus/.*"}}
	case "menus:delete":
		return [][]string{{"DELETE", "/api/v1/menus/.*"}}

	// 接口管理
	case "apis:list":
		return [][]string{{"GET", "/api/v1/apis"}}
	case "apis:create":
		return [][]string{{"POST", "/api/v1/apis"}}
	case "apis:edit":
		return [][]string{{"PUT", "/api/v1/apis/.*"}}
	case "apis:delete":
		return [][]string{{"DELETE", "/api/v1/apis/.*"}}

	// 日志审计
	case "audit:view":
		return [][]string{{"GET", "/api/v1/audit-logs"}}
	case "logins:view":
		return [][]string{{"GET", "/api/v1/login-logs"}}

	// 在线用户
	case "online:list":
		return [][]string{{"GET", "/api/v1/online-users"}}
	case "online:kick":
		return [][]string{{"DELETE", "/api/v1/online-users/.*"}}

	// 媒体管理
	case "media:list":
		return [][]string{{"GET", "/api/v1/media"}}
	case "media:upload":
		return [][]string{{"POST", "/api/v1/media/upload"}}
	case "media:delete":
		return [][]string{{"DELETE", "/api/v1/media/.*"}}

	// 系统配置
	case "config:view":
		return [][]string{{"GET", "/api/v1/system-configs"}}
	case "config:edit":
		return [][]string{{"PUT", "/api/v1/system-configs"}}

	default:
		if ExtraPermissionExpand != nil {
			return ExtraPermissionExpand(key)
		}
		return nil
	}
}

func AllPermissionDefs() []PermissionDef {
	return append(PermissionDefs, ExtraPermissionDefs...)
}

func ExpandPermissionKeys(keys []string) [][]string {
	var policies [][]string
	for _, key := range keys {
		policies = append(policies, ExpandPermission(key)...)
	}
	return policies
}

func GetGrantedPermissionKeys(rawPolicies [][]string) []string {
	granted := make(map[string]bool)
	for _, rd := range AllPermissionDefs() {
		expected := ExpandPermission(rd.Key)
		allGranted := true
		for _, ep := range expected {
			found := false
			for _, rp := range rawPolicies {
				if len(rp) >= 3 && policyMatch(rp[1], ep[0]) && policyMatch(rp[2], ep[1]) {
					found = true
					break
				}
			}
			if !found {
				allGranted = false
				break
			}
		}
		if allGranted && len(expected) > 0 {
			granted[rd.Key] = true
		}
	}
	var keys []string
	for _, rd := range AllPermissionDefs() {
		if granted[rd.Key] {
			keys = append(keys, rd.Key)
		}
	}
	return keys
}

func PermissionGroups() []string {
	seen := map[string]bool{}
	var groups []string
	for _, rd := range AllPermissionDefs() {
		if !seen[rd.Group] {
			seen[rd.Group] = true
			groups = append(groups, rd.Group)
		}
	}
	return groups
}

func PermissionsByGroup() map[string][]PermissionDef {
	m := map[string][]PermissionDef{}
	for _, rd := range AllPermissionDefs() {
		m[rd.Group] = append(m[rd.Group], rd)
	}
	return m
}

func policyMatch(pattern, target string) bool {
	if !strings.Contains(pattern, ".*") {
		return pattern == target
	}
	prefix := strings.TrimSuffix(pattern, ".*")
	return strings.HasPrefix(target, prefix)
}
