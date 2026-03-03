package frbstatus

import (
	"os"
	"strings"
	"testing"
)

func TestExtractOutageID(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{
			url:      "/app/status/outage.do?oId=145787",
			expected: "145787",
		},
		{
			url:      "https://www.frbservices.org/app/status/outage.do?oId=99999",
			expected: "99999",
		},
		{
			url:      "/app/status/message.do?sId=4",
			expected: "",
		},
		{
			url:      "",
			expected: "",
		},
		{
			url:      "app/status/outage.do?other=123&oId=56789",
			expected: "56789",
		},
	}

	for _, tt := range tests {
		result := ExtractOutageID(tt.url)
		if result != tt.expected {
			t.Errorf("ExtractOutageID(%q) = %q, want %q", tt.url, result, tt.expected)
		}
	}
}

func TestParseOutageDetail(t *testing.T) {
	content, err := os.ReadFile("testdata/outage-145787.html")
	if err != nil {
		t.Fatal(err)
	}

	details := ParseOutageDetail(content)

	// Should contain multiple timestamps
	if !strings.Contains(details, "March 03, 2026 9:30 AM") {
		t.Errorf("Expected to find latest timestamp in details, got: %s", details)
	}

	// Should contain "FedACH"
	if !strings.Contains(details, "FedACH") {
		t.Errorf("Expected to find service name in details")
	}
}

func TestParseAlerts(t *testing.T) {
	content, err := os.ReadFile("testdata/fedach-down.html")
	if err != nil {
		t.Fatal(err)
	}

	alerts := ParseAlerts(content)

	if len(alerts) == 0 {
		t.Fatal("Expected alerts to be parsed")
	}

	found := false
	for _, alert := range alerts {
		if alert.ServiceName == "FedACH" && alert.OutageID == "145787" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected to find FedACH alert with outage ID 145787, got: %+v", alerts)
	}
}

func TestStatusPageParsing(t *testing.T) {
	content, err := os.ReadFile("testdata/fedach-down.html")
	if err != nil {
		t.Fatal(err)
	}

	services := ParseStatusPage(content)

	fedACHFound := false
	for _, svc := range services {
		if svc.Name == "FedACH" && IsUnhealthy(svc.Status) {
			fedACHFound = true
			break
		}
	}

	if !fedACHFound {
		t.Error("Expected to find FedACH as unhealthy service")
	}
}
