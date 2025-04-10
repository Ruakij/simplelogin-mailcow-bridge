package alias

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Constants for name generation
const (
	defaultMinNameLength = 3
	defaultMaxNameLength = 10
	defaultWordLength    = 10
	defaultCharLength    = 10
	defaultWordCount     = 1
)

// Generator types
const (
	TypeWords     = "words"
	TypeWordChars = "word-chars"
	TypeChars     = "chars"
	TypeNames     = "names"
)

// Template placeholders
const (
	DomainPlaceholder = "%d"
)

// Generator patterns
const (
	WordsPattern     = "{words}"
	WordCharsPattern = "{word-chars}"
	CharsPattern     = "{chars}"
	NamesPattern     = "{names}"
	// Add sub-patterns for name types
	FirstNamePattern  = "{firstname}"
	LastNamePattern   = "{lastname}"
	MiddleNamePattern = "{middlename}"
	NicknamePattern   = "{nickname}"
)

// Character sets
const (
	letterChars      = "abcdefghijklmnopqrstuvwxyz"
	numberChars      = "0123456789"
	specialChars     = ".-_"
	wordCharsAllowed = letterChars + numberChars
	allCharsAllowed  = wordCharsAllowed + specialChars
)

// Word lists for random word generation
var (
	// Common nouns that make good usernames
	commonNouns = []string{
		"apple", "arrow", "autumn", "beach", "bird", "book", "cake",
		"cloud", "coffee", "diamond", "dream", "eagle", "earth", "fire",
		"forest", "garden", "honey", "island", "jungle", "lake", "leaf",
		"lemon", "light", "lotus", "marble", "meadow", "moon", "mountain",
		"ocean", "panda", "paper", "planet", "river", "rocket", "rose",
		"shadow", "silver", "sky", "snow", "star", "storm", "summer",
		"sunset", "thunder", "tiger", "tree", "valley", "wave", "wind",
		"winter", "wolf", "zebra",
	}

	// Common adjectives that make good usernames
	adjectives = []string{
		"amber", "ancient", "azure", "bold", "brave", "bright", "calm",
		"clever", "cosmic", "crystal", "curious", "daring", "deep", "eager",
		"elegant", "emerald", "enchanted", "energetic", "gentle", "golden",
		"happy", "hidden", "humble", "infinite", "jade", "joyful", "kind",
		"loyal", "lucky", "magical", "mighty", "mystic", "noble", "peaceful",
		"proud", "purple", "quick", "quiet", "radiant", "royal", "ruby",
		"rustic", "serene", "silent", "silver", "smooth", "solar", "swift",
		"tranquil", "valiant", "vibrant", "wild", "wise", "zealous",
	}

	// Vowels and consonants for name generation
	vowels     = []rune{'a', 'e', 'i', 'o', 'u'}
	consonants = []rune{'b', 'c', 'd', 'f', 'g', 'h', 'j', 'k', 'l', 'm', 'n', 'p', 'q', 'r', 's', 't', 'v', 'w', 'x', 'y', 'z'}

	// Variables for more natural name generation

	// Common syllables for more English-sounding names
	commonSyllables = []string{
		"al", "an", "ar", "as", "ash", "ba", "be", "ben", "ber", "beth", "bi", "ble", "bri",
		"ca", "car", "ce", "cha", "che", "chi", "chris", "co", "con", "cy",
		"da", "dan", "de", "di", "do", "don", "dy", "ed", "el", "en", "er", "eth", "ey",
		"fa", "fe", "fi", "fo", "ford", "fred", "fy", "ga", "ge", "geor", "go", "gor",
		"ha", "han", "he", "hi", "ho", "hy", "in", "ing", "is", "ja", "jack", "jam", "je", "jen",
		"ji", "jo", "john", "jon", "ju", "ka", "ke", "ken", "ki", "kin", "la", "le", "len",
		"li", "lin", "lo", "ly", "ma", "mar", "matt", "me", "mel", "mi", "mich", "mo",
		"na", "ne", "ni", "nick", "no", "ny", "pa", "pe", "per", "phi", "pi", "po",
		"ra", "re", "ri", "rich", "rick", "ro", "rob", "ron", "ry",
		"sa", "sam", "se", "sha", "she", "si", "so", "son", "ste", "ster", "ston",
		"ta", "te", "ter", "tho", "thom", "ti", "to", "ton", "ty",
		"va", "ve", "vi", "vic", "vo", "wa", "we", "wil", "win", "wi",
		"ya", "ye", "yo", "za",
	}

	// Common English name endings
	nameEndings = []string{
		"a", "ah", "an", "ane", "ar", "ard", "as", "ay", "ce", "ch", "ck", "cy", "d", "dan",
		"don", "dy", "e", "ed", "el", "en", "er", "ers", "es", "ett", "ey", "feld", "ford",
		"fy", "h", "ia", "ian", "ie", "in", "ing", "ins", "io", "is", "ith", "le", "ley",
		"lyn", "man", "mer", "n", "na", "ne", "ner", "ney", "nie", "ny", "on", "or", "ry",
		"s", "son", "ston", "sy", "t", "th", "ton", "ty", "us", "y", "yn",
	}

	// Common English initial consonant clusters
	initialConsonantClusters = []string{
		"bl", "br", "ch", "cl", "cr", "dr", "fl", "fr", "gl", "gr",
		"pl", "pr", "sc", "sh", "sl", "sm", "sn", "sp", "st", "sw",
		"th", "tr", "tw", "wh", "wr",
	}
)

// RegEx patterns for parsing template options
var (
	// Match {type:length} or {type:min,max}
	lengthRegex = regexp.MustCompile(`\{([a-zA-Z-]+):(\d+)(?:,(\d+))?\}`)
)

// randSource is the source of randomness
var randSource = rand.New(rand.NewSource(time.Now().UnixNano()))

// GenerateAlias generates a new email alias based on the user's email domain and a pattern.
func GenerateAlias(email, pattern string) (string, error) {
	if email == "" || pattern == "" {
		return "", fmt.Errorf("email and pattern must be set")
	}

	// Extract domain from email
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid email format")
	}
	domain := parts[1]

	// Process template
	processed := pattern

	// First identify if there are matching name types to coordinate
	nameTypes := identifyNameTypes(pattern)
	nameValues := generateCoordinatedNames(nameTypes)

	// Replace template variables
	processed = replaceTemplateVariables(processed, nameValues)

	// Replace domain placeholder
	if strings.Contains(processed, DomainPlaceholder) {
		processed = strings.Replace(processed, DomainPlaceholder, domain, -1)
	}

	return processed, nil
}

// identifyNameTypes identifies all name type patterns in a template
func identifyNameTypes(pattern string) map[string]bool {
	nameTypes := make(map[string]bool)

	// Check for standard name patterns
	if strings.Contains(pattern, FirstNamePattern) {
		nameTypes[FirstNamePattern] = true
	}
	if strings.Contains(pattern, LastNamePattern) {
		nameTypes[LastNamePattern] = true
	}
	if strings.Contains(pattern, MiddleNamePattern) {
		nameTypes[MiddleNamePattern] = true
	}
	if strings.Contains(pattern, NicknamePattern) {
		nameTypes[NicknamePattern] = true
	}
	if strings.Contains(pattern, NamesPattern) {
		nameTypes[NamesPattern] = true
	}

	// Check for name patterns with length options
	matches := lengthRegex.FindAllStringSubmatch(pattern, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			typeStr := match[1]
			if typeStr == "firstname" || typeStr == "lastname" ||
				typeStr == "middlename" || typeStr == "nickname" ||
				typeStr == "names" {
				nameTypes["{"+typeStr+"}"] = true
			}
		}
	}

	return nameTypes
}

// generateCoordinatedNames generates name values that are coordinated if multiple name types exist
func generateCoordinatedNames(nameTypes map[string]bool) map[string]string {
	nameValues := make(map[string]string)

	// If we have multiple name types, try to coordinate them
	if len(nameTypes) > 1 {
		// Base name to derive others from
		baseName := generateName(defaultMinNameLength, defaultMaxNameLength)

		for pattern := range nameTypes {
			// Extract min/max length for custom length name patterns
			if matches := lengthRegex.FindStringSubmatch(pattern); len(matches) >= 3 {
				typeStr := matches[1]
				minLength, _ := strconv.Atoi(matches[2])
				maxLength := minLength
				if len(matches) >= 4 && matches[3] != "" {
					maxLength, _ = strconv.Atoi(matches[3])
				}

				// Ensure min <= max
				if minLength > maxLength {
					minLength, maxLength = maxLength, minLength
				}

				// Generate name with specific length range
				switch typeStr {
				case "firstname":
					nameValues[pattern] = generateNameWithLength(baseName, 0.3, minLength, maxLength)
				case "lastname":
					nameValues[pattern] = generateNameWithLength(baseName, 0.5, minLength, maxLength)
				case "middlename":
					nameValues[pattern] = generateNameWithLength(baseName, 0.7, minLength, maxLength)
				case "nickname":
					nameValues[pattern] = generateNameWithLength(baseName, 0.4, minLength, maxLength)
				case "names":
					nameValues[pattern] = generateNameWithLength(baseName, 0.2, minLength, maxLength)
				}
				continue
			}

			// Handle standard patterns without explicit length
			switch pattern {
			case FirstNamePattern:
				nameValues[FirstNamePattern] = generateNameVariation(baseName, 0.3)
			case LastNamePattern:
				nameValues[LastNamePattern] = generateNameVariation(baseName, 0.5)
			case MiddleNamePattern:
				nameValues[MiddleNamePattern] = generateNameVariation(baseName, 0.7)
			case NicknamePattern:
				// Nickname is often shorter
				if len(baseName) > 4 {
					nameValues[NicknamePattern] = baseName[:rand.Intn(len(baseName)-3)+3]
				} else {
					nameValues[NicknamePattern] = baseName
				}
			case NamesPattern:
				nameValues[NamesPattern] = baseName
			}
		}
	} else {
		// Just generate a single name for each type
		for pattern := range nameTypes {
			// Extract min/max length for custom length name patterns
			if matches := lengthRegex.FindStringSubmatch(pattern); len(matches) >= 3 {
				// typeStr := matches[1]
				minLength, _ := strconv.Atoi(matches[2])
				maxLength := minLength
				if len(matches) >= 4 && matches[3] != "" {
					maxLength, _ = strconv.Atoi(matches[3])
				}

				// Ensure min <= max
				if minLength > maxLength {
					minLength, maxLength = maxLength, minLength
				}

				nameValues[pattern] = generateName(minLength, maxLength)
				continue
			}

			// Handle standard patterns
			nameValues[pattern] = generateName(defaultMinNameLength, defaultMaxNameLength)
		}
	}

	return nameValues
}

// replaceTemplateVariables replaces all template variables in the pattern
func replaceTemplateVariables(pattern string, nameValues map[string]string) string {
	result := pattern

	// Replace name patterns first if they exist
	for namePattern, value := range nameValues {
		result = strings.Replace(result, namePattern, value, -1)
	}

	// Process length specifications using regex
	for {
		// Look for patterns with length or length range: {type:length} or {type:min,max}
		matches := lengthRegex.FindStringSubmatch(result)
		if len(matches) < 3 {
			break // No more matches
		}

		typeStr := matches[1]
		minLength, _ := strconv.Atoi(matches[2])
		maxLength := minLength
		if len(matches) >= 4 && matches[3] != "" {
			maxLength, _ = strconv.Atoi(matches[3])
		}

		// Ensure min <= max
		if minLength > maxLength {
			minLength, maxLength = maxLength, minLength
		}

		// Apply sensible defaults if values are unreasonable
		if minLength <= 0 {
			minLength = 1
		}
		if maxLength > 50 {
			maxLength = 50 // Reasonable upper limit
		}

		// Generate appropriate replacement based on type
		var replacement string
		switch typeStr {
		case "words":
			count := minLength
			replacement = generateWords(count)
		case "word-chars":
			length := randBetween(minLength, maxLength)
			replacement = generateWordChars(length)
		case "chars":
			length := randBetween(minLength, maxLength)
			replacement = generateChars(length)
		case "firstname", "lastname", "middlename", "nickname", "names":
			// These are handled in the nameValues map, but we need to handle any that weren't processed
			// Generate a new name with the specified length constraints
			replacement = generateName(minLength, maxLength)
		default:
			// Unknown type, leave it as is
			replacement = matches[0]
		}

		// Replace just the first occurrence
		result = strings.Replace(result, matches[0], replacement, 1)
	}

	// Handle simple patterns without length specifications
	result = processSimplePatterns(result)

	return result
}

// processSimplePatterns handles patterns without explicit length parameters
func processSimplePatterns(pattern string) string {
	result := pattern

	// Simple matches without count
	for {
		if strings.Contains(result, WordsPattern) {
			result = strings.Replace(result, WordsPattern, generateWords(defaultWordCount), 1)
			continue
		}

		if strings.Contains(result, WordCharsPattern) {
			result = strings.Replace(result, WordCharsPattern, generateWordChars(defaultWordLength), 1)
			continue
		}

		if strings.Contains(result, CharsPattern) {
			result = strings.Replace(result, CharsPattern, generateChars(defaultCharLength), 1)
			continue
		}

		// No more matches found
		break
	}

	return result
}

// generateWords generates a specified number of random words separated by a separator
func generateWords(count int) string {
	if count <= 0 {
		count = defaultWordCount
	}

	words := make([]string, count)
	for i := 0; i < count; i++ {
		// 70% chance of adjective-noun pair, 30% chance of just a noun
		if randSource.Float64() < 0.7 && count < 3 {
			adj := adjectives[randSource.Intn(len(adjectives))]
			noun := commonNouns[randSource.Intn(len(commonNouns))]
			words[i] = adj + noun
		} else {
			// Choose from adjectives or nouns
			if randSource.Float64() < 0.5 {
				words[i] = adjectives[randSource.Intn(len(adjectives))]
			} else {
				words[i] = commonNouns[randSource.Intn(len(commonNouns))]
			}
		}
	}

	separator := ""
	// Choose a separator if more than one word
	if count > 1 {
		separators := []string{".", "_", "-", ""}
		separator = separators[randSource.Intn(len(separators))]
	}

	return strings.Join(words, separator)
}

// generateWordChars generates a random string with word-chars (letters and numbers, starting with letter)
func generateWordChars(length int) string {
	if length <= 0 {
		length = defaultWordLength
	}

	// First character must be a letter
	result := string(letterChars[randSource.Intn(len(letterChars))])

	// Rest can be letters or numbers
	if length > 1 {
		result += generateRandomChars(wordCharsAllowed, length-1)
	}

	return result
}

// generateChars generates a random string with all allowed chars (starting with letter)
func generateChars(length int) string {
	if length <= 0 {
		length = defaultCharLength
	}

	// First character must be a letter
	result := string(letterChars[randSource.Intn(len(letterChars))])

	// Rest can be any allowed character
	if length > 1 {
		result += generateRandomChars(allCharsAllowed, length-1)
	}

	return result
}

// generateName generates a random name-like string with a length between min and max
func generateName(minLength, maxLength int) string {
	// Apply sensible defaults
	if minLength <= 0 {
		minLength = defaultMinNameLength
	}
	if maxLength <= 0 {
		maxLength = defaultMaxNameLength
	}
	if minLength > maxLength {
		minLength, maxLength = maxLength, minLength
	}

	// Determine a random length between min and max
	length := randBetween(minLength, maxLength)

	// For more English-sounding names, use a combination of common syllables
	// and typical English phonetic patterns
	var result string

	// Start with either a common syllable, an initial consonant cluster, or a single consonant
	startChoice := randSource.Float64()
	if startChoice < 0.5 {
		// Use a common English syllable to start (50% chance)
		result = commonSyllables[randSource.Intn(len(commonSyllables))]
	} else if startChoice < 0.8 {
		// Use a common English initial consonant cluster (30% chance)
		result = initialConsonantClusters[randSource.Intn(len(initialConsonantClusters))]
		// Add a vowel after the cluster
		result += string(vowels[randSource.Intn(len(vowels))])
	} else {
		// Start with a single consonant followed by a vowel (20% chance)
		result = string(consonants[randSource.Intn(len(consonants))]) +
			string(vowels[randSource.Intn(len(vowels))])
	}

	// Build the name with English patterns until we're close to the target length
	for len(result) < length-2 {
		// 60% chance to add a common English syllable if there's room
		if randSource.Float64() < 0.6 && len(result)+2 <= length {
			syllable := commonSyllables[randSource.Intn(len(commonSyllables))]
			// Make sure we don't exceed the target length
			if len(result)+len(syllable) <= length {
				result += syllable
				continue
			}
		}

		// Otherwise add alternating vowels and consonants following English patterns
		lastChar := []rune(result)[len([]rune(result))-1]
		if isVowel(lastChar) {
			// After a vowel, typically a consonant follows
			result += string(consonants[randSource.Intn(len(consonants))])
		} else {
			// After a consonant, typically a vowel follows
			result += string(vowels[randSource.Intn(len(vowels))])
		}
	}

	// Add a typical English ending if we need more characters
	if len(result) <= length-2 && len(nameEndings) > 0 {
		// Find a suitable ending that fits
		for i := 0; i < 5; i++ { // Try up to 5 times to find a fitting ending
			ending := nameEndings[randSource.Intn(len(nameEndings))]
			if len(result)+len(ending) <= length {
				result += ending
				break
			}
		}
	}

	// If we need just one more character
	if len(result) == length-1 {
		lastChar := []rune(result)[len([]rune(result))-1]
		if isVowel(lastChar) {
			// After a vowel, add a consonant that works well at the end of English names
			endConsonants := []string{"n", "l", "r", "s", "t", "m", "th", "y"}
			resultString := endConsonants[randSource.Intn(len(endConsonants))]
			// Handle special case for 'th'
			if resultString == "th" {
				if len(result)+2 <= length {
					result += "th"
				} else {
					result += "t"
				}
			} else {
				result += resultString
			}
		} else {
			// After a consonant, add a vowel that works well at the end of English names
			endVowels := []rune{'a', 'e', 'i', 'o', 'y'}
			result += string(endVowels[randSource.Intn(len(endVowels))])
		}
	}

	// Trim if too long
	if len(result) > length {
		result = result[:length]
	}

	// Capitalize first letter for English names (more common than not)
	if randSource.Float64() < 0.7 {
		runes := []rune(result)
		runes[0] = unicode.ToUpper(runes[0])
		result = string(runes)
	}

	return result
}

// generateNameVariation creates a variation of a name while keeping some similarity
func generateNameVariation(baseName string, changeRatio float64) string {
	runes := []rune(baseName)

	// Determine how many characters to change
	changesToMake := int(float64(len(runes)) * changeRatio)

	for i := 0; i < changesToMake; i++ {
		pos := randSource.Intn(len(runes))

		if isVowel(runes[pos]) {
			// Replace vowel with another vowel
			runes[pos] = vowels[randSource.Intn(len(vowels))]
		} else {
			// Replace consonant with another consonant
			runes[pos] = consonants[randSource.Intn(len(consonants))]
		}
	}

	// Sometimes add or remove a character
	if randSource.Float64() < 0.3 && len(runes) > 3 {
		// Remove a random character
		pos := randSource.Intn(len(runes))
		runes = append(runes[:pos], runes[pos+1:]...)
	} else if randSource.Float64() < 0.3 {
		// Add a random character
		pos := randSource.Intn(len(runes))
		var newChar rune
		if isVowel(runes[pos]) {
			newChar = consonants[randSource.Intn(len(consonants))]
		} else {
			newChar = vowels[randSource.Intn(len(vowels))]
		}
		runes = append(runes[:pos], append([]rune{newChar}, runes[pos:]...)...)
	}

	// Keep consistent capitalization with base name
	if unicode.IsUpper([]rune(baseName)[0]) {
		runes[0] = unicode.ToUpper(runes[0])
	} else {
		runes[0] = unicode.ToLower(runes[0])
	}

	return string(runes)
}

// generateNameWithLength creates a variation of a name with a specific length
func generateNameWithLength(baseName string, changeRatio float64, minLength, maxLength int) string {
	// Generate variation first
	variation := generateNameVariation(baseName, changeRatio)
	runes := []rune(variation)

	// If already within range, return as-is
	if len(runes) >= minLength && len(runes) <= maxLength {
		return variation
	}

	// If too short, add characters
	if len(runes) < minLength {
		for len(runes) < minLength {
			// Add characters alternating between vowels and consonants
			pos := len(runes) - 1 // Add to end
			lastIsVowel := isVowel(runes[pos])

			var newChar rune
			if lastIsVowel {
				newChar = consonants[randSource.Intn(len(consonants))]
			} else {
				newChar = vowels[randSource.Intn(len(vowels))]
			}

			runes = append(runes, newChar)
		}
	}

	// If too long, trim
	if len(runes) > maxLength {
		// Try to trim at syllable boundaries if possible
		trimPoint := maxLength
		for i := maxLength; i >= minLength; i-- {
			if i > 0 && i < len(runes) && isVowel(runes[i-1]) && !isVowel(runes[i]) {
				trimPoint = i
				break
			}
		}
		runes = runes[:trimPoint]
	}

	return string(runes)
}

// isVowel checks if a rune is a vowel
func isVowel(r rune) bool {
	r = unicode.ToLower(r)
	for _, v := range vowels {
		if r == v {
			return true
		}
	}
	return false
}

// generateRandomChars generates a random string from the given character set
func generateRandomChars(charset string, length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[randSource.Intn(len(charset))]
	}
	return string(b)
}

// randBetween returns a random integer between min and max (inclusive)
func randBetween(min, max int) int {
	if min == max {
		return min
	}
	return randSource.Intn(max-min+1) + min
}
