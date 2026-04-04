package values

import (
	"regexp"
	"strings"
)

// Disposable/temporary email domains (most common)
var disposableEmailDomains = map[string]bool{
	// Popular temp mail services
	"tempmail.com":          true,
	"temp-mail.org":         true,
	"guerrillamail.com":     true,
	"guerrillamail.org":     true,
	"guerrillamail.net":     true,
	"10minutemail.com":      true,
	"10minutemail.net":      true,
	"mailinator.com":        true,
	"maildrop.cc":           true,
	"throwaway.email":       true,
	"throwawaymail.com":     true,
	"fakeinbox.com":         true,
	"trashmail.com":         true,
	"trashmail.net":         true,
	"dispostable.com":       true,
	"mailnesia.com":         true,
	"yopmail.com":           true,
	"yopmail.fr":            true,
	"sharklasers.com":       true,
	"getairmail.com":        true,
	"getnada.com":           true,
	"tempail.com":           true,
	"mohmal.com":            true,
	"emailondeck.com":       true,
	"mintemail.com":         true,
	"tempinbox.com":         true,
	"spamgourmet.com":       true,
	"mailcatch.com":         true,
	"mytemp.email":          true,
	"mailsac.com":           true,
	"burnermail.io":         true,
	"inboxkitten.com":       true,
	"instantemail.net":      true,
	"tempmailo.com":         true,
	"emailfake.com":         true,
	"fakemailgenerator.com": true,
	"crazymailing.com":      true,
}

// Suspicious email patterns (regex)
var suspiciousEmailPatterns = []*regexp.Regexp{
	// Plus addressing with numbers: user+123@gmail.com
	regexp.MustCompile(`\+\d+@`),
	// Many dots before @: u.s.e.r@gmail.com
	regexp.MustCompile(`^[^@]*\.{2,}[^@]*@`),
	// Only numbers in local part: 12345@example.com
	regexp.MustCompile(`^[\d]+@`),
	// Suspicious random local part: 3f8a2b1c@example.com
	regexp.MustCompile(`^[0-9a-f]{8,}@`),
	// Very short random local part: ab@example.com
	regexp.MustCompile(`^[a-z]{1,2}@`),
}

// EmailValidationResult contains validation result
type EmailValidationResult struct {
	IsValid     bool
	IsSuspect   bool
	Reason      string
	SignalType  string
	BlockReason string
}

// ValidateEmail checks email for suspicious patterns and disposable domains
// Returns validation result with details
func ValidateEmail(email string) EmailValidationResult {
	email = strings.ToLower(strings.TrimSpace(email))

	// Check for empty email
	if email == "" {
		return EmailValidationResult{
			IsValid:     false,
			BlockReason: "email is required",
		}
	}

	// Extract domain
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return EmailValidationResult{
			IsValid:     false,
			BlockReason: "invalid email format",
		}
	}
	domain := parts[1]

	// Check disposable domains
	if disposableEmailDomains[domain] {
		return EmailValidationResult{
			IsValid:     true,
			IsSuspect:   true,
			Reason:      "disposable email domain: " + domain,
			SignalType:  "email_pattern",
			BlockReason: "disposable email addresses are not allowed",
		}
	}

	// Check subdomain variations of disposable domains
	for disposableDomain := range disposableEmailDomains {
		if strings.HasSuffix(domain, "."+disposableDomain) {
			return EmailValidationResult{
				IsValid:     true,
				IsSuspect:   true,
				Reason:      "subdomain of disposable domain: " + domain,
				SignalType:  "email_pattern",
				BlockReason: "disposable email addresses are not allowed",
			}
		}
	}

	// Check suspicious patterns
	for _, pattern := range suspiciousEmailPatterns {
		if pattern.MatchString(email) {
			return EmailValidationResult{
				IsValid:    true,
				IsSuspect:  true,
				Reason:     "suspicious email pattern detected",
				SignalType: "email_pattern",
				// Don't block, just flag as suspicious
			}
		}
	}

	return EmailValidationResult{
		IsValid:   true,
		IsSuspect: false,
	}
}

// IsDisposableEmailDomain checks if domain is in the disposable list
func IsDisposableEmailDomain(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	return disposableEmailDomains[parts[1]]
}
