package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/lexfrei/go-unifi/api/sitemanager"
)

var (
	apiKey  = flag.String("api-key", os.Getenv("UNIFI_API_KEY"), "UniFi API key (or use UNIFI_API_KEY env)")
	verbose = flag.Bool("verbose", false, "Verbose output with full JSON responses")
)

type TestResult struct {
	Endpoint    string
	Success     bool
	Error       string
	Issues      []string
	JSONSample  string
	Duration    time.Duration
	StatusCode  int
	AnyFields   []string // Fields typed as any/interface{}
	EmptyFields []string // Optional fields that were nil/empty
}

func main() {
	flag.Parse()

	if *apiKey == "" {
		log.Fatal("API key is required. Use -api-key flag or UNIFI_API_KEY environment variable")
	}

	fmt.Println("🧪 Testing go-unifi against reality...")
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println()

	client, err := sitemanager.New(*apiKey)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Get system information
	fmt.Println("📡 Connecting to UniFi API...")
	hostResp, err := client.ListHosts(ctx, nil)
	if err == nil && len(hostResp.Data) > 0 {
		host := hostResp.Data[0]
		if host.ReportedState != nil {
			if host.ReportedState.Hostname != nil {
				fmt.Printf("   Hostname: %s\n", *host.ReportedState.Hostname)
			}
			fmt.Printf("   Type: %s\n", host.Type)

			// Get UniFi OS version from Hardware.FirmwareVersion
			if host.ReportedState.Hardware != nil && host.ReportedState.Hardware.FirmwareVersion != nil {
				fmt.Printf("   UniFi OS: %s\n", *host.ReportedState.Hardware.FirmwareVersion)
			}

			// Find Network controller version
			if host.ReportedState.Controllers != nil {
				for _, controller := range *host.ReportedState.Controllers {
					if controller.Name != nil && *controller.Name == "network" {
						if controller.Version != nil {
							fmt.Printf("   Network: %s\n", *controller.Version)
						}
						break
					}
				}
			}
		}
		fmt.Println()
	}

	results := []TestResult{}

	// Test v1 endpoints
	results = append(results, testListHosts(ctx, client))
	results = append(results, testListSites(ctx, client))
	results = append(results, testListDevices(ctx, client))

	// Test EA endpoints
	results = append(results, testGetISPMetrics(ctx, client))
	results = append(results, testListSDWANConfigs(ctx, client))

	// Print summary
	fmt.Println()
	fmt.Println("📊 Test Summary")
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println()

	totalIssues := 0
	for _, result := range results {
		status := "✅"
		if !result.Success {
			status = "❌"
		} else if len(result.Issues) > 0 {
			status = "⚠️"
		}

		fmt.Printf("%s %s (HTTP %d, %v)\n", status, result.Endpoint, result.StatusCode, result.Duration)

		if result.Error != "" {
			fmt.Printf("   Error: %s\n", result.Error)
		}

		if len(result.AnyFields) > 0 {
			fmt.Printf("   ⚠️  Fields typed as 'any': %d\n", len(result.AnyFields))
			for _, field := range result.AnyFields {
				fmt.Printf("      - %s\n", field)
			}
			totalIssues += len(result.AnyFields)
		}

		if len(result.Issues) > 0 {
			fmt.Printf("   ⚠️  Type issues: %d\n", len(result.Issues))
			for _, issue := range result.Issues {
				fmt.Printf("      - %s\n", issue)
			}
			totalIssues += len(result.Issues)
		}

		if *verbose && result.JSONSample != "" {
			fmt.Printf("   JSON Sample:\n%s\n", indentJSON(result.JSONSample, "      "))
		}

		fmt.Println()
	}

	fmt.Println("=" + strings.Repeat("=", 60))
	if totalIssues == 0 {
		fmt.Println("✅ All tests passed! No type issues found.")
	} else {
		fmt.Printf("⚠️  Found %d potential type issues\n", totalIssues)
		fmt.Println()
		fmt.Println("Recommendations:")
		fmt.Println("  1. Replace 'any' with concrete types in OpenAPI spec")
		fmt.Println("  2. Add oneOf/anyOf schemas for polymorphic fields")
		fmt.Println("  3. Review optional fields - some might be required")
	}
}

func testListHosts(ctx context.Context, client *sitemanager.UnifiClient) TestResult {
	start := time.Now()
	result := TestResult{Endpoint: "ListHosts (v1)"}

	resp, err := client.ListHosts(ctx, nil)
	result.Duration = time.Since(start)

	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.StatusCode = resp.HttpStatusCode
	result.Success = true

	// Analyze response structure
	if len(resp.Data) > 0 {
		host := resp.Data[0]
		result.AnyFields = findAnyFields(host, "Host")
		result.Issues = analyzeStructFields(host, "Host")

		if *verbose {
			data, _ := json.MarshalIndent(host, "", "  ")
			result.JSONSample = string(data)
		}
	}

	return result
}

func testListSites(ctx context.Context, client *sitemanager.UnifiClient) TestResult {
	start := time.Now()
	result := TestResult{Endpoint: "ListSites (v1)"}

	resp, err := client.ListSites(ctx)
	result.Duration = time.Since(start)

	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.StatusCode = resp.HttpStatusCode
	result.Success = true

	if len(resp.Data) > 0 {
		site := resp.Data[0]
		result.AnyFields = findAnyFields(site, "Site")
		result.Issues = analyzeStructFields(site, "Site")

		if *verbose {
			data, _ := json.MarshalIndent(site, "", "  ")
			result.JSONSample = string(data)
		}
	}

	return result
}

func testListDevices(ctx context.Context, client *sitemanager.UnifiClient) TestResult {
	start := time.Now()
	result := TestResult{Endpoint: "ListDevices (v1)"}

	resp, err := client.ListDevices(ctx, nil)
	result.Duration = time.Since(start)

	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.StatusCode = resp.HttpStatusCode
	result.Success = true

	if len(resp.Data) > 0 {
		device := resp.Data[0]
		result.AnyFields = findAnyFields(device, "DeviceListItemsResponse")
		result.Issues = analyzeStructFields(device, "DeviceListItemsResponse")

		if *verbose {
			data, _ := json.MarshalIndent(device, "", "  ")
			result.JSONSample = string(data)
		}
	}

	return result
}

func testGetISPMetrics(ctx context.Context, client *sitemanager.UnifiClient) TestResult {
	start := time.Now()
	result := TestResult{Endpoint: "GetISPMetrics (EA)"}

	duration := sitemanager.GetISPMetricsParamsDuration("24h")
	resp, err := client.GetISPMetrics(ctx, "5m", &sitemanager.GetISPMetricsParams{
		Duration: &duration,
	})
	result.Duration = time.Since(start)

	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.StatusCode = resp.HttpStatusCode
	result.Success = true

	if len(resp.Data) > 0 {
		metric := resp.Data[0]
		result.AnyFields = findAnyFields(metric, "ISPMetric")
		result.Issues = analyzeStructFields(metric, "ISPMetric")

		if *verbose && metric.Periods != nil && len(*metric.Periods) > 0 {
			data, _ := json.MarshalIndent((*metric.Periods)[0], "", "  ")
			result.JSONSample = string(data)
		}
	}

	return result
}

func testListSDWANConfigs(ctx context.Context, client *sitemanager.UnifiClient) TestResult {
	start := time.Now()
	result := TestResult{Endpoint: "ListSDWANConfigs (EA)"}

	resp, err := client.ListSDWANConfigs(ctx)
	result.Duration = time.Since(start)

	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.StatusCode = resp.HttpStatusCode
	result.Success = true

	if len(resp.Data) > 0 {
		config := resp.Data[0]
		result.AnyFields = findAnyFields(config, "SDWANConfig")
		result.Issues = analyzeStructFields(config, "SDWANConfig")

		if *verbose {
			data, _ := json.MarshalIndent(config, "", "  ")
			result.JSONSample = string(data)
		}
	}

	return result
}

// findAnyFields recursively finds fields typed as interface{} or any
func findAnyFields(v interface{}, path string) []string {
	var fields []string

	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return fields
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fields
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if !field.CanInterface() {
			continue
		}

		fieldPath := path + "." + fieldType.Name

		// Check if field is interface{} or any
		if field.Kind() == reflect.Interface && field.Type().NumMethod() == 0 {
			fields = append(fields, fieldPath)
			continue
		}

		// Recursively check nested structs
		if field.Kind() == reflect.Struct {
			fields = append(fields, findAnyFields(field.Interface(), fieldPath)...)
		} else if field.Kind() == reflect.Ptr && !field.IsNil() {
			fields = append(fields, findAnyFields(field.Interface(), fieldPath)...)
		} else if field.Kind() == reflect.Slice && field.Len() > 0 {
			// Check first element of slice
			elem := field.Index(0)
			if elem.Kind() == reflect.Struct || (elem.Kind() == reflect.Ptr && elem.Elem().Kind() == reflect.Struct) {
				fields = append(fields, findAnyFields(elem.Interface(), fieldPath+"[]")...)
			}
		} else if field.Kind() == reflect.Map {
			// Check if map value is interface{}
			if field.Type().Elem().Kind() == reflect.Interface {
				fields = append(fields, fieldPath+" (map[*]interface{})")
			}
		}
	}

	return fields
}

// analyzeStructFields checks for common type issues
func analyzeStructFields(v interface{}, path string) []string {
	var issues []string

	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return issues
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return issues
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if !field.CanInterface() {
			continue
		}

		_ = path + "." + fieldType.Name // fieldPath for future analysis

		// Check for pointer fields that might not need to be pointers
		if field.Kind() == reflect.Ptr && !field.IsNil() {
			// If it's always populated, maybe it shouldn't be optional
			// This is just a heuristic - we'd need multiple samples to be sure
		}

		// Check for empty slices vs nil slices
		if field.Kind() == reflect.Slice && field.Len() == 0 && !field.IsNil() {
			// Empty slice might indicate the field is always present
		}
	}

	return issues
}

func indentJSON(jsonStr, indent string) string {
	lines := strings.Split(jsonStr, "\n")
	for i, line := range lines {
		lines[i] = indent + line
	}
	return strings.Join(lines, "\n")
}
