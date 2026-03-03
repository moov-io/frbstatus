package frbstatus

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"

	"golang.org/x/net/html"
)

const baseURL = "https://www.frbservices.org"

type Service struct {
	Name      string
	Status    string
	DetailURL string
	OutletURL string
}

type StatusClient struct {
	client *http.Client
}

func NewClient() *StatusClient {
	jar, _ := cookiejar.New(nil)
	return &StatusClient{
		client: &http.Client{
			Jar: jar,
		},
	}
}

func (c *StatusClient) FetchStatusPage() ([]byte, error) {
	resp, err := c.client.Get(baseURL + "/app/status/serviceStatus.do")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return readBody(resp.Body)
}

func (c *StatusClient) FetchOutageDetail(outageID string) ([]byte, error) {
	url := fmt.Sprintf("%s/app/status/outage.do?oId=%s", baseURL, outageID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return readBody(resp.Body)
}

func readBody(body io.Reader) ([]byte, error) {
	return io.ReadAll(body)
}

func ParseStatusPage(htmlContent []byte) []Service {
	doc, _ := html.Parse(strings.NewReader(string(htmlContent)))

	var services []Service

	var inServiceTable bool
	var currentService *Service

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "table" {
			inServiceTable = isServiceTable(n)
		}

		if inServiceTable {
			if n.Type == html.ElementNode && n.Data == "tr" {
				if currentService != nil && currentService.Name != "" && currentService.Status != "" {
					services = append(services, *currentService)
				}
				cs := parseServiceTr(n)
				currentService = &cs
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(doc)

	if currentService != nil && currentService.Name != "" && currentService.Status != "" {
		services = append(services, *currentService)
	}

	return services
}

func isServiceTable(table *html.Node) bool {
	for c := table.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "thead" {
			for th := c.FirstChild; th != nil; th = th.NextSibling {
				if th.Type == html.ElementNode && th.Data == "tr" {
					for cell := th.FirstChild; cell != nil; cell = cell.NextSibling {
						if cell.Type == html.ElementNode && cell.Data == "th" {
							text := extractText(cell)
							if strings.Contains(text, "Services") || strings.Contains(text, "Status") {
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
}

func parseServiceTr(trNode *html.Node) Service {
	service := Service{}

	tds := 0
	for c := trNode.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "td" {
			tds++
			if tds == 1 {
				service.Name = extractText(c)
			}
			if tds >= 2 {
				for img := c.FirstChild; img != nil; img = img.NextSibling {
					if img.Type == html.ElementNode && img.Data == "img" {
						for _, attr := range img.Attr {
							if attr.Key == "alt" {
								service.Status = attr.Val
							}
						}
					}
					if img.Type == html.ElementNode && img.Data == "a" {
						for _, attr := range img.Attr {
							if attr.Key == "onclick" {
								// Extract URL from onclick like location.href='app/status/message.do?serviceId=1' or window.open('...')
								var url string
								if strings.Contains(attr.Val, "location.href=") {
									start := strings.Index(attr.Val, "'") + 1
									end := strings.LastIndex(attr.Val, "'")
									if start > 0 && end > start {
										url = attr.Val[start:end]
									}
								} else if strings.Contains(attr.Val, "window.open(") {
									start := strings.Index(attr.Val, "'") + 1
									end := strings.LastIndex(attr.Val, "'")
									if start > 0 && end > start {
										url = attr.Val[start:end]
									}
								}
								if url != "" {
									service.DetailURL = baseURL + url
									service.OutletURL = service.DetailURL
								}
							} else if attr.Key == "href" {
								service.DetailURL = baseURL + attr.Val
								service.OutletURL = service.DetailURL
							}
						}
					}
				}
			}
		}
	}

	if service.DetailURL == "" {
		service.DetailURL = baseURL + "/app/status/serviceStatus.do"
		service.OutletURL = service.DetailURL
	}

	return service
}

func ParseOutageDetail(htmlContent []byte) string {
	doc, _ := html.Parse(strings.NewReader(string(htmlContent)))

	var details []string

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "td" {
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				if child.Type == html.ElementNode && child.Data == "strong" {
					timestamp := extractText(child)
					if timestamp != "" {
						details = append(details, timestamp)
					}
				} else if len(details) > 0 && child.Type == html.TextNode {
					text := strings.TrimSpace(child.Data)
					if text != "" {
						details = append(details, text)
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(doc)

	return strings.Join(details, "\n")
}

func extractText(n *html.Node) string {
	var buf strings.Builder
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			buf.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return strings.TrimSpace(buf.String())
}

func IsUnhealthy(status string) bool {
	return status == "Service Issue" || status == "Service Disruption"
}

func ExtractOutageID(url string) string {
	parts := strings.Split(url, "oId=")
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

func ParseAlerts(htmlContent []byte) []Alert {
	doc, _ := html.Parse(strings.NewReader(string(htmlContent)))

	var alerts []Alert

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "h2" {
			if strings.Contains(extractText(n), "Alert") {
				alert := parseAlertSection(n)
				if alert.ServiceName != "" && alert.OutageID != "" {
					alerts = append(alerts, alert)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(doc)

	return alerts
}

func parseAlertSection(h2 *html.Node) Alert {
	alert := Alert{}

	for c := h2.NextSibling; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "h2" {
			break
		}
		if c.Type == html.ElementNode && c.Data == "p" {
			alert.ServiceName = extractServiceFromAlert(c)
			for link := c.FirstChild; link != nil; link = link.NextSibling {
				if link.Type == html.ElementNode && link.Data == "a" {
					for _, attr := range link.Attr {
						if attr.Key == "onclick" {
							outageID := extractOutageIDFromOnClick(attr.Val)
							if outageID != "" {
								alert.OutageID = outageID
							}
						}
					}
				}
			}
		}
	}

	return alert
}

func extractServiceFromAlert(p *html.Node) string {
	text := extractText(p)
	if strings.Contains(text, "FedACH") {
		return "FedACH"
	}
	if strings.Contains(text, "Fedwire Funds") {
		return "Fedwire Funds"
	}
	if strings.Contains(text, "Fedwire Securities") {
		return "Fedwire Securities"
	}
	if strings.Contains(text, "FedNow") {
		return "FedNow"
	}
	if strings.Contains(text, "FedCash") {
		return "FedCash"
	}
	if strings.Contains(text, "Account Services") {
		return "Account Services"
	}
	if strings.Contains(text, "Check 21") {
		return "Check 21"
	}
	if strings.Contains(text, "Check Adjustments") {
		return "Check Adjustments"
	}
	if strings.Contains(text, "Central Bank") {
		return "Central Bank"
	}
	if strings.Contains(text, "National Settlement") {
		return "National Settlement"
	}
	return ""
}

func extractOutageIDFromOnClick(onclick string) string {
	parts := strings.Split(onclick, "oId=")
	if len(parts) == 2 {
		id := strings.Split(parts[1], "'")[0]
		return id
	}
	return ""
}

type Alert struct {
	ServiceName string
	OutageID    string
}
