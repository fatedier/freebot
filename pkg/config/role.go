package config

type RoleOptions map[string][]string // role -> []string{user1, user2}

type LabelRoles map[string]map[string][]string // label -> role -> users
