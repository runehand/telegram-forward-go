package jobparse

import (
	"regexp"
	"slices"
	"strings"

	"zenfl-forwarder/apps/backend/internal/domain"
)

var hashtagRe = regexp.MustCompile(`(?i)#([a-z0-9_]+)`)

func FromTelegramText(text, jobLink string) domain.Job {
	lines := splitLines(text)
	title := ""
	if len(lines) > 0 {
		title = lines[0]
	}

	skills := splitSkills(sectionText(text, "Skills", "About Client"))
	description := sectionText(text, "Description", "Questions")
	questions := splitQuestions(sectionText(text, "Questions", "Reference Documentation"))
	clientLocation := parseClientLocation(text)
	budget, projectType, experienceLevel, category, subcategory := parseHeaderMeta(text)
	tags, specialTags, countries := parseTags(text)
	onlyUS := containsFold(specialTags, "onlyus") || containsCountry(countries, "united states")
	onlyMobile := containsFold(specialTags, "onlymobile")
	onlyCountry := containsFold(specialTags, "onlycountry") || strings.Contains(strings.ToLower(text), "only us") || strings.Contains(strings.ToLower(text), "only united states")

	return domain.Job{
		Title:           title,
		RawText:         text,
		JobLink:         jobLink,
		Skills:          skills,
		Tags:            tags,
		SpecialTags:     specialTags,
		Description:     strings.TrimSpace(description),
		Questions:       questions,
		ClientLocation:  clientLocation,
		Countries:       countries,
		Budget:          budget,
		ProjectType:     projectType,
		ExperienceLevel: experienceLevel,
		Category:        category,
		Subcategory:     subcategory,
		PaymentVerified: strings.Contains(strings.ToLower(text), "payment verified"),
		OnlyUS:          onlyUS,
		OnlyMobile:      onlyMobile,
		OnlyCountry:     onlyCountry,
		PostedAgoText:   parsePostedAgoText(text),
	}
}

func splitLines(text string) []string {
	raw := strings.Split(text, "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func sectionText(text, start, end string) string {
	startIdx := strings.Index(text, start+"\n")
	if startIdx == -1 {
		return ""
	}
	body := text[startIdx+len(start)+1:]
	if end != "" {
		if endIdx := strings.Index(body, "\n"+end); endIdx >= 0 {
			body = body[:endIdx]
		}
	}
	return strings.TrimSpace(body)
}

func splitSkills(text string) []string {
	if text == "" {
		return nil
	}
	chunks := strings.Split(strings.ReplaceAll(text, "\n", " "), "•")
	out := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		v := strings.TrimSpace(chunk)
		if v != "" {
			out = append(out, v)
		}
	}
	return unique(out)
}

func splitQuestions(text string) []string {
	lines := splitLines(text)
	if len(lines) == 0 {
		return nil
	}
	return unique(lines)
}

func parseClientLocation(text string) string {
	section := sectionText(text, "About Client", "Description")
	if section == "" {
		return ""
	}
	parts := strings.Split(section, "•")
	for _, part := range parts {
		val := strings.TrimSpace(part)
		if val == "" {
			continue
		}
		if looksLikeLocation(val) {
			return val
		}
	}
	return ""
}

func looksLikeLocation(v string) bool {
	lower := strings.ToLower(v)
	if strings.Contains(lower, "client since") || strings.Contains(lower, "payment verified") || strings.Contains(lower, "jobs") || strings.Contains(lower, "spent $") || strings.Contains(lower, "hire rate") || strings.Contains(lower, "reviews") {
		return false
	}
	return strings.Contains(v, "United") || strings.Contains(v, "Canada") || strings.Contains(v, "Australia") || strings.Contains(v, "India") || strings.Contains(v, "Germany") || strings.Contains(v, "UK") || strings.Contains(v, "France") || strings.Contains(v, "Europe")
}

func parseHeaderMeta(text string) (budget, projectType, experienceLevel, category, subcategory string) {
	lines := splitLines(text)
	if len(lines) < 2 {
		return "", "", "", "", ""
	}
	parts := strings.Split(lines[1], "•")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		v := strings.TrimSpace(part)
		if v != "" {
			clean = append(clean, v)
		}
	}
	if len(clean) > 0 {
		category = clean[0]
	}
	if len(clean) > 1 {
		subcategory = clean[1]
	}
	for _, part := range clean {
		switch {
		case strings.Contains(part, "$"):
			budget = part
		case strings.Contains(strings.ToLower(part), "level"):
			experienceLevel = part
		case strings.Contains(strings.ToLower(part), "project"):
			projectType = part
		}
	}
	return
}

func parseTags(text string) (tags, specialTags, countries []string) {
	matches := hashtagRe.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		tag := strings.ToLower(strings.TrimSpace(match[1]))
		if tag == "" {
			continue
		}
		tags = append(tags, tag)
		switch tag {
		case "onlyus":
			specialTags = append(specialTags, tag)
			countries = append(countries, "United States")
		case "onlymobile", "onlycountry", "onlytoday":
			specialTags = append(specialTags, tag)
		default:
			if strings.HasPrefix(tag, "only") && len(tag) > 4 {
				specialTags = append(specialTags, tag)
			}
		}
	}
	return unique(tags), unique(specialTags), unique(countries)
}

func parsePostedAgoText(text string) string {
	lines := splitLines(text)
	for _, line := range lines {
		if strings.Contains(line, "ago") {
			return line
		}
	}
	return ""
}

func unique(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, item := range in {
		if item == "" || slices.Contains(out, item) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func containsFold(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(item, target) {
			return true
		}
	}
	return false
}

func containsCountry(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(item, target) {
			return true
		}
	}
	return false
}
