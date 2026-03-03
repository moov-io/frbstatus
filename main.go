package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/moov-io/frbstatus/pkg/frbstatus"
)

func main() {
	unhealthyOnly := flag.Bool("unhealthy", false, "only report unhealthy services")
	format := flag.String("format", "table", "output format (table, json)")
	file := flag.String("file", "", "read HTML from file instead of fetching from API")
	flag.Parse()

	client := frbstatus.NewClient()

	var statusPage []byte
	var err error

	if *file != "" {
		statusPage, err = os.ReadFile(*file)
	} else {
		statusPage, err = client.FetchStatusPage()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading status page: %v\n", err)
		os.Exit(1)
	}

	services := frbstatus.ParseStatusPage(statusPage)
	alerts := frbstatus.ParseAlerts(statusPage)

	var outputServices []frbstatus.Service
	for _, svc := range services {
		if *unhealthyOnly && !frbstatus.IsUnhealthy(svc.Status) {
			continue
		}
		outputServices = append(outputServices, svc)
	}

	switch *format {
	case "json":
		outputJSON(outputServices)
	default:
		outputTable(outputServices, *unhealthyOnly)
	}

	if webhookURL := os.Getenv("SLACK_WEBHOOK_URL"); webhookURL != "" {
		alertMap := make(map[string]frbstatus.Alert)
		for _, alert := range alerts {
			alertMap[alert.ServiceName] = alert
		}
		for i := range services {
			if frbstatus.IsUnhealthy(services[i].Status) {
				if alert, ok := alertMap[services[i].Name]; ok && alert.OutageID != "" {
					services[i].OutletURL = alert.OutageID
				}
				sendToSlack(client, webhookURL, services[i])
			}
		}
	}
}

func outputTable(services []frbstatus.Service, unhealthyOnly bool) {
	fmt.Printf("\nFRB Service Status\n")
	fmt.Printf("==================\n\n")

	fmt.Printf("%-30s %s\n", "SERVICE", "STATUS")
	fmt.Printf("%-30s %s\n", strings.Repeat("-", 30), strings.Repeat("-", 20))

	for _, svc := range services {
		status := svc.Status
		if frbstatus.IsUnhealthy(svc.Status) {
			status = fmt.Sprintf("⚠️  %s", svc.Status)
		}
		fmt.Printf("%-30s %s\n", svc.Name, status)
	}
}

func outputJSON(services []frbstatus.Service) {
	data, _ := json.MarshalIndent(services, "", "  ")
	fmt.Println(string(data))
}

func sendToSlack(client *frbstatus.StatusClient, webhookURL string, svc frbstatus.Service) {
	if svc.OutletURL == "" {
		return
	}

	outageID := svc.OutletURL

	outageDetail, err := client.FetchOutageDetail(outageID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching outage detail: %v\n", err)
		return
	}

	details := frbstatus.ParseOutageDetail(outageDetail)

	message := fmt.Sprintf(
		"*⚠️ FRB Service Issue: %s*",
		svc.Name,
	)

	if details != "" {
		lines := strings.Split(details, "\n")
		for i, line := range lines {
			if i >= 5 {
				break
			}
			if strings.Contains(line, "Eastern Time") {
				message += fmt.Sprintf("\n\n%s", line)
			}
		}
	}

	message += fmt.Sprintf("\n\n<https://www.frbservices.org/app/status/outage.do?oId=%s|View Details>", outageID)

	webhook := map[string]interface{}{
		"text": message,
	}

	data, _ := json.Marshal(webhook)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "POST", webhookURL, strings.NewReader(string(data)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending to Slack: %v\n", err)
		return
	}
	defer resp.Body.Close()
}
