package main

import (
	"context"
	"fmt"
	"image/color"
	"net"
	"sort"
	"sync"
	"time"
	"strings"
	"encoding/csv"
	"os"
	"path/filepath"
	"os/exec"
	"runtime"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/unit"
)

type DNSProvider struct {
	Name     string
	IP       string
	Selected widget.Bool
	IPv6     string // Added IPv6 support
}

type TestResult struct {
	Provider    DNSProvider
	Latency    time.Duration
	Success    bool
	TestsDone  int
	TotalTests int
	Errors     []string
	TimeStamp  time.Time
}

type TestConfig struct {
	TestsPerDomain int
	Timeout        time.Duration
	UseTCP         bool
	UseIPv6        bool
	ParallelTests  bool
}

type UI struct {
	window         *app.Window
	theme          *material.Theme
	providers      []*DNSProvider
	startButton    widget.Clickable
	exportButton   widget.Clickable
	configButton   widget.Clickable
	progress       float32
	results        string
	status         string
	testing        bool
	config         TestConfig
	list           *widget.List
	tabs           *widget.Enum
	showConfig     bool
	testHistory    [][]TestResult
	errorLog       []string
	decreaseTests  widget.Clickable
	increaseTests  widget.Clickable
	decreaseTimeout widget.Clickable
	increaseTimeout widget.Clickable
	useTCPCheckbox widget.Bool
	useIPv6Checkbox widget.Bool
	parallelCheckbox widget.Bool
}

var (
	providers = []*DNSProvider{
		{Name: "Cloudflare", IP: "1.1.1.1", IPv6: "2606:4700:4700::1111"},
		{Name: "Cloudflare Secondary", IP: "1.0.0.1", IPv6: "2606:4700:4700::1001"},
		{Name: "Google", IP: "8.8.8.8", IPv6: "2001:4860:4860::8888"},
		{Name: "Google Secondary", IP: "8.8.4.4", IPv6: "2001:4860:4860::8844"},
		{Name: "Quad9", IP: "9.9.9.9", IPv6: "2620:fe::fe"},
		{Name: "Quad9 Secondary", IP: "149.112.112.112", IPv6: "2620:fe::9"},
		{Name: "OpenDNS", IP: "208.67.222.222", IPv6: "2620:119:35::35"},
		{Name: "OpenDNS Secondary", IP: "208.67.220.220", IPv6: "2620:119:53::53"},
		{Name: "Comodo", IP: "8.26.56.26"},
		{Name: "Comodo Secondary", IP: "8.20.247.20"},
	}
	testDomains = []string{
		"www.google.com",
		"www.amazon.com",
		"www.netflix.com",
		"www.facebook.com",
		"www.microsoft.com",
		"www.apple.com",
		"www.github.com",
	}
)

func main() {
	go func() {
		w := app.NewWindow(
			app.Title("DNS Speed Test"),
			app.Size(unit.Dp(800), unit.Dp(600)),
		)
		ui := &UI{
			window:  w,
			theme:   material.NewTheme(),
			providers: providers,
			list:    &widget.List{List: layout.List{Axis: layout.Vertical}},
			tabs:    &widget.Enum{},
			config: TestConfig{
				TestsPerDomain: 3,
				Timeout:        3 * time.Second,
				UseTCP:        false,
				UseIPv6:       false,
				ParallelTests: true,
			},
			// Initialize configuration controls
			useTCPCheckbox:    widget.Bool{Value: false},
			useIPv6Checkbox:   widget.Bool{Value: false},
			parallelCheckbox:  widget.Bool{Value: true},
		}
		ui.tabs.Value = "test"
		ui.status = "Ready to test DNS servers"
		if err := ui.loop(); err != nil {
			fmt.Printf("error: %v\n", err)
		}
	}()
	app.Main()
}

func (ui *UI) loop() error {
	var ops op.Ops
	for {
		e := <-ui.window.Events()
		switch e := e.(type) {
		case system.DestroyEvent:
			return e.Err
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)
			ui.layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

func (ui *UI) layout(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.layoutTabs(gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			switch ui.tabs.Value {
			case "test":
				return ui.layoutTest(gtx)
			case "history":
				return ui.layoutHistory(gtx)
			case "config":
				return ui.layoutConfig(gtx)
			default:
				return layout.Dimensions{}
			}
		}),
	)
}

func (ui *UI) layoutTabs(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.RadioButton(ui.theme, ui.tabs, "test", "Test").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.RadioButton(ui.theme, ui.tabs, "history", "History").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.RadioButton(ui.theme, ui.tabs, "config", "Config").Layout(gtx)
		}),
	)
}

func (ui *UI) layoutTest(gtx layout.Context) layout.Dimensions {
	if ui.startButton.Clicked() && !ui.testing {
		go ui.runTests()
	}
	if ui.exportButton.Clicked() && len(ui.results) > 0 {
		go ui.exportResults()
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.H6(ui.theme, "DNS Providers").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			var children []layout.FlexChild
			for i := range ui.providers {
				i := i
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					provider := ui.providers[i]
					return material.CheckBox(ui.theme, &provider.Selected, 
						fmt.Sprintf("%s (%s%s)", 
							provider.Name, 
							provider.IP,
							func() string {
								if provider.IPv6 != "" && ui.config.UseIPv6 {
									return ", " + provider.IPv6
								}
								return ""
							}(),
						)).Layout(gtx)
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					btn := material.Button(ui.theme, &ui.startButton, "Start Test")
					if ui.testing {
						btn.Background = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
					}
					return btn.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Button(ui.theme, &ui.exportButton, "Export Results").Layout(gtx)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Body1(ui.theme, ui.status).Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			progressBar := material.ProgressBar(ui.theme, ui.progress)
			return progressBar.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Body1(ui.theme, ui.results).Layout(gtx)
		}),
	)
}

func (ui *UI) layoutHistory(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.H6(ui.theme, "Test History").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			var children []layout.FlexChild
			for i := len(ui.testHistory) - 1; i >= 0; i-- {
				results := ui.testHistory[i]
				if len(results) == 0 {
					continue
				}
				timestamp := results[0].TimeStamp.Format("2006-01-02 15:04:05")
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Body1(ui.theme, fmt.Sprintf("\nTest Run: %s\n", timestamp)).Layout(gtx)
				}))
				for _, result := range results {
					result := result
					children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						text := fmt.Sprintf("%-20s: ", result.Provider.Name)
						if result.Success {
							text += fmt.Sprintf("%v", result.Latency)
						} else {
							text += "Failed"
						}
						return material.Body2(ui.theme, text).Layout(gtx)
					}))
				}
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
	)
}

func (ui *UI) layoutConfig(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.H6(ui.theme, "Configuration").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Body1(ui.theme, fmt.Sprintf("Tests per domain: %d", ui.config.TestsPerDomain)).Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							btn := material.Button(ui.theme, &ui.decreaseTests, "-")
							dims := btn.Layout(gtx)
							if ui.decreaseTests.Clicked() {
								if ui.config.TestsPerDomain > 1 {
									ui.config.TestsPerDomain--
								}
							}
							return dims
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							btn := material.Button(ui.theme, &ui.increaseTests, "+")
							dims := btn.Layout(gtx)
							if ui.increaseTests.Clicked() {
								if ui.config.TestsPerDomain < 10 {
									ui.config.TestsPerDomain++
								}
							}
							return dims
						}),
					)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Body1(ui.theme, fmt.Sprintf("Timeout: %v", ui.config.Timeout)).Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							btn := material.Button(ui.theme, &ui.decreaseTimeout, "-")
							dims := btn.Layout(gtx)
							if ui.decreaseTimeout.Clicked() {
								if ui.config.Timeout > time.Second {
									ui.config.Timeout -= time.Second
								}
							}
							return dims
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							btn := material.Button(ui.theme, &ui.increaseTimeout, "+")
							dims := btn.Layout(gtx)
							if ui.increaseTimeout.Clicked() {
								if ui.config.Timeout < 10*time.Second {
									ui.config.Timeout += time.Second
								}
							}
							return dims
						}),
					)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					dims := material.CheckBox(ui.theme, &ui.useTCPCheckbox, "Use TCP for DNS queries").Layout(gtx)
					ui.config.UseTCP = ui.useTCPCheckbox.Value
					return dims
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					dims := material.CheckBox(ui.theme, &ui.useIPv6Checkbox, "Use IPv6 when available").Layout(gtx)
					ui.config.UseIPv6 = ui.useIPv6Checkbox.Value
					return dims
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					dims := material.CheckBox(ui.theme, &ui.parallelCheckbox, "Run tests in parallel").Layout(gtx)
					ui.config.ParallelTests = ui.parallelCheckbox.Value
					return dims
				}),
			)
		}),
	)
}

func (ui *UI) exportResults() {
	if len(ui.results) == 0 {
		return
	}

	// Get user's Documents folder
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		ui.errorLog = append(ui.errorLog, fmt.Sprintf("Failed to get user home directory: %v", err))
		return
	}
	docsDir := filepath.Join(userHomeDir, "Documents")

	// Create filename with timestamp
	filename := fmt.Sprintf("dns_test_results_%s.csv", time.Now().Format("2006-01-02_15-04-05"))
	filepath := filepath.Join(docsDir, filename)

	// Create and write to file
	file, err := os.Create(filepath)
	if err != nil {
		ui.errorLog = append(ui.errorLog, fmt.Sprintf("Failed to create file: %v", err))
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"Provider", "IP", "Latency", "Success", "Tests Done", "Total Tests"})

	// Write data
	for _, line := range strings.Split(ui.results, "\n") {
		if strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				providerParts := strings.Split(strings.TrimSpace(parts[0]), " ")
				if len(providerParts) >= 2 {
					name := strings.Join(providerParts[:len(providerParts)-1], " ")
					ip := strings.Trim(providerParts[len(providerParts)-1], "()")
					latency := strings.TrimSpace(parts[1])
					writer.Write([]string{name, ip, latency})
				}
			}
		}
	}

	// Open the folder in explorer and highlight the file
	go func() {
		switch runtime.GOOS {
		case "windows":
			exec.Command("explorer", "/select,", filepath).Run()
		case "darwin": // macOS
			exec.Command("open", "-R", filepath).Run()
		default: // Linux and others
			exec.Command("xdg-open", docsDir).Run()
		}
	}()

	ui.status = fmt.Sprintf("Results exported to %s", filepath)
}

func testProvider(provider DNSProvider, onProgress func(), testsPerDomain int, config TestConfig) (time.Duration, bool, int) {
	var totalLatency time.Duration
	var successfulTests int

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: config.Timeout}
			protocol := "udp"
			if config.UseTCP {
				protocol = "tcp"
			}
			ip := provider.IP
			if config.UseIPv6 && provider.IPv6 != "" {
				ip = provider.IPv6
			}
			return d.DialContext(ctx, protocol, ip+":53")
		},
	}

	runTest := func(domain string) {
		ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
		defer cancel()
		
		start := time.Now()
		_, err := resolver.LookupHost(ctx, domain)
		latency := time.Since(start)

		if err == nil && latency < config.Timeout {
			totalLatency += latency
			successfulTests++
		}
		onProgress()
	}

	if config.ParallelTests {
		var wg sync.WaitGroup
		for _, domain := range testDomains {
			for i := 0; i < testsPerDomain; i++ {
				wg.Add(1)
				go func(d string) {
					defer wg.Done()
					runTest(d)
				}(domain)
			}
		}
		wg.Wait()
	} else {
		for _, domain := range testDomains {
			for i := 0; i < testsPerDomain; i++ {
				runTest(domain)
			}
		}
	}

	if successfulTests == 0 {
		return config.Timeout, false, len(testDomains) * testsPerDomain
	}
	return totalLatency / time.Duration(successfulTests), true, successfulTests
}

func (ui *UI) runTests() {
	ui.testing = true
	ui.results = ""
	ui.status = "Testing DNS servers..."
	ui.progress = 0

	var selectedProviders []*DNSProvider
	for _, p := range ui.providers {
		if p.Selected.Value {
			selectedProviders = append(selectedProviders, p)
		}
	}

	if len(selectedProviders) == 0 {
		ui.status = "Please select at least one DNS provider"
		ui.testing = false
		return
	}

	totalTests := len(selectedProviders) * len(testDomains) * ui.config.TestsPerDomain
	resultsChan := make(chan TestResult, len(selectedProviders))
	var wg sync.WaitGroup
	testsCompleted := 0

	testStartTime := time.Now()

	for _, provider := range selectedProviders {
		wg.Add(1)
		go func(p *DNSProvider) {
			defer wg.Done()
			latency, success, testsDone := testProvider(
				DNSProvider{Name: p.Name, IP: p.IP, IPv6: p.IPv6},
				func() {
					testsCompleted++
					ui.progress = float32(testsCompleted) / float32(totalTests)
					ui.window.Invalidate()
				},
				ui.config.TestsPerDomain,
				ui.config,
			)
			resultsChan <- TestResult{
				Provider:    DNSProvider{Name: p.Name, IP: p.IP, IPv6: p.IPv6},
				Latency:    latency,
				Success:    success,
				TestsDone:  testsDone,
				TotalTests: len(testDomains) * ui.config.TestsPerDomain,
				TimeStamp:  testStartTime,
			}
		}(provider)
	}

	go func() {
		wg.Wait()
		close(resultsChan)

		var testResults []TestResult
		for result := range resultsChan {
			testResults = append(testResults, result)
		}
		sort.Slice(testResults, func(i, j int) bool {
			return testResults[i].Latency < testResults[j].Latency
		})

		// Add to history
		ui.testHistory = append(ui.testHistory, testResults)
		if len(ui.testHistory) > 10 {
			ui.testHistory = ui.testHistory[1:]
		}

		var resultText string
		resultText = "DNS Provider Latency Results:\n"
		resultText += "----------------------------------------\n"
		for _, result := range testResults {
			if !result.Success {
				resultText += fmt.Sprintf("%-20s (%s): Timeout or Error\n",
					result.Provider.Name, result.Provider.IP)
			} else {
				resultText += fmt.Sprintf("%-20s (%s): %v\n",
					result.Provider.Name, result.Provider.IP, result.Latency)
			}
		}

		ui.results = resultText
		ui.status = "Testing completed"
		ui.testing = false
		ui.progress = 1.0
		ui.window.Invalidate()
	}()
} 