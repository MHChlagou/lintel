package checker

// Registry returns the built-in checkers in fixed order (mirrors §8).
func Registry() map[string]Checker {
	return map[string]Checker{
		"secrets":        Secrets{},
		"malicious_code": Malicious{},
		"dependencies":   Deps{},
		"lint":           Lint{},
		"format":         Format{},
	}
}
