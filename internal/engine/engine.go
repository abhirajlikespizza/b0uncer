package engine

import (
	"encoding/json"
	"os"
	"strings"
)

type Decision struct {
	Action    string
	Reason    string
	RiskScore int
}

type Policies struct {
	CustomBlockPatterns []string `json:"custom_block_patterns"`
	CustomWarnPatterns  []string `json:"custom_warn_patterns"`
	LogAllowed          bool     `json:"log_allowed_commands"`
}

func Evaluate(command string, p *Policies) Decision {
	lower := strings.ToLower(command)

	// Special case: base64 decode piped to shell (check both space and no-space variants)
	if strings.Contains(lower, "base64 -d") {
		if strings.Contains(lower, "|bash") || strings.Contains(lower, "| bash") ||
			strings.Contains(lower, "|sh") || strings.Contains(lower, "| sh") {
			return Decision{"block", "Attempted to execute encoded payload", 95}
		}
	}

	// Built-in block rules
	type blockRule struct {
		pattern string
		reason  string
		score   int
	}
	blockRules := []blockRule{
		{"rm -rf /", "Attempted to delete root filesystem", 100},
		{"rm -rf ~", "Attempted to delete home directory", 95},
		{"rm -rf $home", "Attempted to delete home directory", 95},
		{":(){:|:&};", "Fork bomb detected", 100},
		{"dd if=/dev/zero", "Attempted to wipe a disk", 95},
		{"mkfs.", "Attempted to format a filesystem", 95},
		{"| bash", "Attempted to pipe remote content to bash", 90},
		{"| sh", "Attempted to pipe remote content to sh", 90},
		{"> /dev/sd", "Attempted to write directly to disk", 95},
		{"shred /dev/", "Attempted to shred disk device", 95},
		{"/etc/shadow", "Attempted to read password hashes", 90},
		{"/.aws/credentials", "Attempted to access AWS credentials", 90},
		{"/.ssh/id_rsa", "Attempted to access SSH private key", 90},
	}
	for _, rule := range blockRules {
		if strings.Contains(lower, rule.pattern) {
			return Decision{"block", rule.reason, rule.score}
		}
	}

	// Built-in warn rules
	type warnCheck struct {
		match  func(string) bool
		reason string
		score  int
	}
	warnChecks := []warnCheck{
		{func(s string) bool { return strings.HasPrefix(s, "sudo ") }, "Sudo usage detected", 60},
		{func(s string) bool { return strings.Contains(s, "chmod 777") }, "World-writable permissions", 65},
		{func(s string) bool { return strings.Contains(s, "chmod -r") }, "Recursive permission change", 55},
		{func(s string) bool { return strings.Contains(s, "kill -9") }, "Force kill", 50},
		{func(s string) bool { return strings.Contains(s, "npm install -g") }, "Global package install", 50},
		{func(s string) bool { return strings.Contains(s, "pip install") }, "Python package install", 50},
		{func(s string) bool { return strings.Contains(s, "apt-get install") }, "System package install", 55},
		{func(s string) bool { return strings.Contains(s, "wget ") }, "File download", 50},
		{func(s string) bool {
			return strings.Contains(s, "curl ") && strings.Contains(s, "-o ")
		}, "Downloading file to disk", 55},
	}

	var warnResult *Decision
	for _, wc := range warnChecks {
		if wc.match(lower) {
			d := Decision{"warn", wc.reason, wc.score}
			warnResult = &d
			break
		}
	}

	// Custom patterns (can escalate severity)
	if p != nil {
		for _, pat := range p.CustomBlockPatterns {
			if pat != "" && strings.Contains(lower, strings.ToLower(pat)) {
				return Decision{"block", "Blocked by custom policy", 90}
			}
		}
		if warnResult == nil {
			for _, pat := range p.CustomWarnPatterns {
				if pat != "" && strings.Contains(lower, strings.ToLower(pat)) {
					d := Decision{"warn", "Flagged by custom policy", 60}
					warnResult = &d
					break
				}
			}
		}
	}

	if warnResult != nil {
		return *warnResult
	}

	return Decision{"allow", "", 0}
}

func LoadPolicies(path string) (*Policies, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return &Policies{LogAllowed: true}, nil
	}
	var p Policies
	if err := json.Unmarshal(data, &p); err != nil {
		return &Policies{LogAllowed: true}, err
	}
	return &p, nil
}
