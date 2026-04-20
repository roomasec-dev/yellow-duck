package edr

import (
	"fmt"
	"strings"
)

// Filter 单个过滤条件
type Filter struct {
	Field      string `json:"field"`
	Operator   string `json:"operator"` // is, gt, lt, contain
	Value      string `json:"value"`
	IsDisabled bool   `json:"is_disabled"`
}

// Search KQL 查询结构
type Search struct {
	SearchSentence string `json:"search_sentence"`
	SearchType     string `json:"search_type"` // KQL
}

// HuntingPreset 预设狩猎条件
type HuntingPreset struct {
	ID          int
	Name        string // 英文标识
	NameCn      string // 中文名称
	Description string // 描述
	Filters     []Filter
	Search      Search
}

var huntingPresets = []HuntingPreset{
	{
		ID:          1,
		Name:        "unknown_process_start",
		NameCn:      "检测未知进程启动",
		Description: "processlevel 在 30-70 之间的未知进程",
		Filters: []Filter{
			{Field: "processlevel", Operator: "gt", Value: "30", IsDisabled: false},
			{Field: "processlevel", Operator: "lt", Value: "70", IsDisabled: false},
		},
		Search: Search{
			SearchSentence: "fltrid:1",
			SearchType:     "KQL",
		},
	},
	{
		ID:          2,
		Name:        "suspicious_schtasks_create",
		NameCn:      "检测可疑进程创建计划任务",
		Description: "schtasks.exe create",
		Filters: []Filter{
			{Field: "pe.internalname", Operator: "is", Value: "schtasks.exe", IsDisabled: false},
			{Field: "new_command_line", Operator: "is", Value: "create", IsDisabled: false},
			{Field: "processtreelevel", Operator: "gt", Value: "30", IsDisabled: false},
		},
		Search: Search{
			SearchSentence: "fltrid:1",
			SearchType:     "KQL",
		},
	},
	{
		ID:          3,
		Name:        "wmic_create_process",
		NameCn:      "检索通过 wmic 创建进程",
		Description: "wmic.exe create process",
		Filters: []Filter{
			{Field: "pe.internalname", Operator: "is", Value: "wmic.exe", IsDisabled: false},
			{Field: "new_command_line", Operator: "is", Value: "create", IsDisabled: false},
			{Field: "new_command_line", Operator: "is", Value: "process", IsDisabled: false},
		},
		Search: Search{
			SearchSentence: "",
			SearchType:     "KQL",
		},
	},
	{
		ID:          4,
		Name:        "suspicious_powershell",
		NameCn:      "检索可疑 PowerShell 命令",
		Description: "包含 FromBase64String",
		Filters:     []Filter{},
		Search: Search{
			SearchSentence: "fltrid: 50300 AND scripttype: Powershell AND scriptcontent: *FromBase64String*",
			SearchType:     "KQL",
		},
	},
	{
		ID:          5,
		Name:        "cmd_command_input",
		NameCn:      "检索 cmd 命令输入",
		Description: "fltrid: 8000",
		Filters:     []Filter{},
		Search: Search{
			SearchSentence: "fltrid: 8000",
			SearchType:     "KQL",
		},
	},
	{
		ID:          6,
		Name:        "lsass_memory_access",
		NameCn:      "检索 lsass 内存访问",
		Description: "lsass.exe 访问",
		Filters: []Filter{
			{Field: "processtreelevel", Operator: "gt", Value: "20", IsDisabled: false},
		},
		Search: Search{
			SearchSentence: "fltrid: 6001 and object: \"lsass.exe\"",
			SearchType:     "KQL",
		},
	},
	{
		ID:          7,
		Name:        "unknown_process_query_computername",
		NameCn:      "检索未知进程查询计算机名称",
		Description: "QueryValueKey",
		Filters: []Filter{
			{Field: "processlevel", Operator: "gt", Value: "20", IsDisabled: false},
		},
		Search: Search{
			SearchSentence: "operation : \"QueryValueKey\" and keyname : \"\\\\Control\\\\ComputerName\\\\ActiveComputerName\"",
			SearchType:     "KQL",
		},
	},
	{
		ID:          8,
		Name:        "process_whitelist_bypass",
		NameCn:      "检索进程白利用",
		Description: "低等级进程加载高风险 DLL",
		Filters: []Filter{
			{Field: "operation", Operator: "is", Value: "LoadImage", IsDisabled: false},
			{Field: "processlevel", Operator: "lt", Value: "30", IsDisabled: false},
			{Field: "newimagelevel", Operator: "is", Value: "40", IsDisabled: false},
			{Field: "newimage", Operator: "is", Value: "dll", IsDisabled: false},
		},
		Search: Search{
			SearchSentence: "",
			SearchType:     "KQL",
		},
	},
	{
		ID:          9,
		Name:        "no_filter_query",
		NameCn:      "不使用条件查询",
		Description: "直接查询最近15分钟",
		Filters:     []Filter{},
		Search: Search{
			SearchSentence: "",
			SearchType:     "KQL",
		},
	},
	{
		ID:          10,
		Name:        "custom_kql",
		NameCn:      "自定义条件查询",
		Description: "用户输入 KQL 语句",
		Filters:     []Filter{},
		Search: Search{
			SearchSentence: "",
			SearchType:     "KQL",
		},
	},
}

// GetPresetByID 根据 ID 获取预设
func GetPresetByID(id int) *HuntingPreset {
	for i := range huntingPresets {
		if huntingPresets[i].ID == id {
			return &huntingPresets[i]
		}
	}
	return nil
}

// FormatPresetsForClarification 格式化为"请选择..."询问文本
func FormatPresetsForClarification() string {
	lines := []string{"请选择预设狩猎条件："}
	for _, preset := range huntingPresets {
		lines = append(lines, fmt.Sprintf("%d - %s", preset.ID, preset.NameCn))
	}
	return strings.Join(lines, "\n")
}
