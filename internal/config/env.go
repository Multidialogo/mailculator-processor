package config

import (
	"sync"
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"path/filepath"
)

// Registry to store environment variables
type EnvRegistry struct {
	vars map[string]string
	mu   sync.RWMutex
}

// Global registry instance
var registry *EnvRegistry
var once sync.Once

// GetRegistry ensures that the registry is initialized only once
func GetRegistry() *EnvRegistry {
	once.Do(func() {
		registry = &EnvRegistry{
			vars: make(map[string]string),
		}
		registry.loadEnvVars()
	})

	// Return the registry
	return registry
}

// loadEnvVars loads environment variables using godotenv and populates the registry
func (e *EnvRegistry) loadEnvVars() {
	// Load from .env file if it exists
	if err := godotenv.Load(); err != nil {
		envDir, _ := os.Getwd()

		panic(fmt.Errorf("failed to load .env file from %s: %w", filepath.Join(envDir, ".env"), err))
	}

	// Add environment variables to the registry
	// Iterate over all environment variables loaded by godotenv
	for _, key := range os.Environ() {
		// Split the environment variable into key-value pairs
		kv := splitKeyValue(key)
		if kv != nil {
			e.Set(kv[0], kv[1])
		}
	}
}

// Set adds a key-value pair to the registry
func (e *EnvRegistry) Set(key, value string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.vars[key] = value
}

// Get retrieves a value from the registry by key
func (e *EnvRegistry) Get(key string) string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.vars[key]
}

// GetOrDefault retrieves a value from the registry or returns the default if the key is not set
func (e *EnvRegistry) GetOrDefault(key, defaultValue string) string {
	value := e.Get(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// PrintAll prints all environment variables stored in the registry
func (e *EnvRegistry) PrintAll() {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.vars) == 0 {
		fmt.Println("No environment variables set.")
		return
	}

	fmt.Println("Environment Variables:")
	for key, value := range e.vars {
		fmt.Printf("%s=%s\n", key, value)
	}
}

// Helper function to split key-value pairs in environment variables
func splitKeyValue(envVar string) []string {
	// Split the string into key and value using the first '=' as the delimiter
	kv := []string{}
	parts := splitOnce(envVar, '=')
	if len(parts) == 2 {
		kv = parts
	}
	return kv
}

// Helper function to split a string by a delimiter, but only at the first occurrence
func splitOnce(s string, delimiter rune) []string {
	parts := []string{}
	index := -1
	for i, r := range s {
		if r == delimiter && index == -1 {
			index = i
		}
	}
	if index != -1 {
		parts = append(parts, s[:index], s[index+1:])
	} else {
		parts = append(parts, s)
	}
	return parts
}
