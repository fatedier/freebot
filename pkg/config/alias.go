package config

type AliasOptions struct {
	Cmds   map[string]string `json:"cmds"`
	Labels map[string]string `json:"labels"`
	Users  map[string]string `json:"users"`
}
