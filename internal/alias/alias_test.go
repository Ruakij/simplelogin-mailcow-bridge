package alias

import (
	"fmt"
	"testing"
)

func TestGenerateNames(t *testing.T) {
	// Sample email for template testing
	email := "user@example.com"

	// Test simple name generation
	templates := []string{
		"{names}@%d",
		"{firstname}.{lastname}@%d",
		"{firstname}{lastname}@%d",
		"{firstname}-{lastname}@%d",
		"{firstname}_{middlename}_{lastname}@%d",
		"{nickname}@%d",
		"{firstname:4,7}.{lastname:5,9}@%d",
		"{names:3,8}@%d",
	}

	fmt.Println("Generated Name Examples:")
	fmt.Println("=======================")

	// Generate multiple examples for each template
	for _, template := range templates {
		fmt.Printf("\nTemplate: %s\n", template)
		fmt.Println("Results:")
		for i := 0; i < 5; i++ {
			alias, err := GenerateAlias(email, template)
			if err != nil {
				t.Errorf("Error generating alias with template %s: %v", template, err)
				continue
			}
			fmt.Printf("  %s\n", alias)
		}
	}

	// Test coordinated name variations
	fmt.Println("\nCoordinated Name Examples:")
	fmt.Println("=========================")
	coordTemplates := []string{
		"{firstname}.{lastname}@%d",
		"{firstname}.{lastname}.{middlename}@%d",
		"{nickname}-{firstname}@%d",
	}

	for _, template := range coordTemplates {
		fmt.Printf("\nTemplate: %s\n", template)
		fmt.Println("Results:")
		for i := 0; i < 5; i++ {
			alias, err := GenerateAlias(email, template)
			if err != nil {
				t.Errorf("Error generating alias with template %s: %v", template, err)
				continue
			}
			fmt.Printf("  %s\n", alias)
		}
	}

	// Test length constrained names
	fmt.Println("\nLength-Constrained Name Examples:")
	fmt.Println("================================")
	lengthTemplates := []string{
		"{firstname:3,5}@%d",
		"{lastname:6,8}@%d",
		"{nickname:4}@%d",
		"{names:7,10}@%d",
	}

	for _, template := range lengthTemplates {
		fmt.Printf("\nTemplate: %s\n", template)
		fmt.Println("Results:")
		for i := 0; i < 5; i++ {
			alias, err := GenerateAlias(email, template)
			if err != nil {
				t.Errorf("Error generating alias with template %s: %v", template, err)
				continue
			}
			fmt.Printf("  %s\n", alias)
		}
	}

	// Skip actual assertions since this is a demonstration test
	// The test will pass as long as no errors occur during generation
}

func TestRandomWordGeneration(t *testing.T) {
	fmt.Println("\nRandom Word Generation Examples:")
	fmt.Println("===============================")

	fmt.Println("\nSingle Words:")
	for i := 0; i < 10; i++ {
		fmt.Printf("  %s\n", generateWords(1))
	}

	fmt.Println("\nTwo Words:")
	for i := 0; i < 10; i++ {
		fmt.Printf("  %s\n", generateWords(2))
	}

	fmt.Println("\nThree Words:")
	for i := 0; i < 5; i++ {
		fmt.Printf("  %s\n", generateWords(3))
	}
}

// This is not actually a test, but a utility to demonstrate the name generation
// Run with: go test -v ./internal/alias -run TestNameGenerator
func TestNameGenerator(t *testing.T) {
	// Skip in normal test runs
	if testing.Short() {
		t.Skip("Skipping demonstration test in short mode")
	}

	fmt.Println("\nRandom Name Generation:")
	fmt.Println("======================")

	// Generate random names of different lengths
	fmt.Println("\nStandard Names (3-10 chars):")
	for i := 0; i < 10; i++ {
		fmt.Printf("  %s\n", generateName(3, 10))
	}

	fmt.Println("\nShort Names (3-5 chars):")
	for i := 0; i < 10; i++ {
		fmt.Printf("  %s\n", generateName(3, 5))
	}

	fmt.Println("\nLong Names (8-12 chars):")
	for i := 0; i < 10; i++ {
		fmt.Printf("  %s\n", generateName(8, 12))
	}

	// Demonstrate name variations
	baseName := generateName(6, 8)
	fmt.Printf("\nBase Name: %s\n", baseName)
	fmt.Println("Variations:")
	for i := 0; i < 10; i++ {
		// Different change ratios
		ratio := 0.2 + float64(i)*0.05
		fmt.Printf("  %.2f: %s\n", ratio, generateNameVariation(baseName, ratio))
	}
}
