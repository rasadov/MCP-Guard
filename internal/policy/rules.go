package policy

import (
	"encoding/json"
	"slices"
)

type Rules struct {
	DenyTools         []string            `json:"deny_tools"`
	DenyWriteForRoles []string            `json:"deny_write_for_roles"`
	WriteTools        []string            `json:"write_tools"`
	ChannelAllowlist  map[string][]string `json:"channel_allowlist"`
}

func ParseRules(data []byte) (Rules, error) {
	var rules Rules
	if len(data) == 0 {
		return rules, nil
	}
	err := json.Unmarshal(data, &rules)
	return rules, err
}

func ParseToolList(data []byte) ([]string, error) {
	var tools []string
	if len(data) == 0 {
		return tools, nil
	}
	err := json.Unmarshal(data, &tools)
	return tools, err
}

func ToolAllowedBySkill(skillTools []string, toolName string) bool {
	return slices.Contains(skillTools, toolName)
}

func IsWriteTool(rules Rules, toolName string) bool {
	return slices.Contains(rules.WriteTools, toolName)
}
