package main

import (
	"context"
	"fmt"
	"image/color"
	"net"
	"sort"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type DNSProvider struct {
	Name     string
	IP       string
	Selected widget.Bool
}

type TestResult struct {
	Provider    DNSProvider
	Latency    time.Duration
	Success    bool
	TestsDone  int
	TotalTests int
}

type UI struct {
	window        *app.Window
	theme         *material.Theme
	providers     []*DNSProvider
	startButton   widget.Clickable
	progress      float32
	results       string
	status        string
	testing       bool
	testsPerDomain int
	list          *widget.List
}

var (
	providers = []*DNSProvider{
		{Name: "Cloudflare", IP: "1.1.1.1"},
		{Name: "Cloudflare Secondary", IP: "1.0.0.1"},
		{Name: "Google", IP: "8.8.8.8"},
		{Name: "Google Secondary", IP: "8.8.4.4"},
		{Name: "Quad9", IP: "9.9.9.9"},
		{Name: "Quad9 Secondary", IP: "149.112.112.112"},
		{Name: "OpenDNS", IP: "208.67.222.222"},
		{Name: "OpenDNS Secondary", IP: "208.67.220.220"},
		{Name: "Comodo", IP: "8.26.56.26"},
		{Name: "Comodo Secondary", IP: "8.20.247.20"},
	}
	testDomains = []string{
		"www.google.com",
		"www.amazon.com",
		"www.netflix.com",
	}
	timeout = 3 * time.Second
)

func main() {
	go func() {
		w := app.NewWindow(app.Title("DNS Speed Test"))
		ui := &UI{
			window:        w,
			theme:        material.NewTheme(),
			providers:    providers,
			testsPerDomain: 3,
			list:         &widget.List{List: layout.List{Axis: layout.Vertical}},
		}
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
	if ui.startButton.Clicked() && !ui.testing {
		go ui.runTests()
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.H6(ui.theme, "DNS Providers").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			var children []layout.FlexChild
			for i := range ui.providers {
				i := i // capture loop variable
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					provider := ui.providers[i]
					return material.CheckBox(ui.theme, &provider.Selected, provider.Name+" ("+provider.IP+")").Layout(gtx)
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					btn := material.Button(ui.theme, &ui.startButton, "Start Test")
					if ui.testing {
						btn.Background = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
					}
					return btn.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Body1(ui.theme, ui.status).Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					progressBar := material.ProgressBar(ui.theme, ui.progress)
					return progressBar.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Body1(ui.theme, ui.results).Layout(gtx)
				}),
			)
		}),
	)
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

	totalTests := len(selectedProviders) * len(testDomains) * ui.testsPerDomain
	resultsChan := make(chan TestResult, len(selectedProviders))
	var wg sync.WaitGroup
	testsCompleted := 0

	for _, provider := range selectedProviders {
		wg.Add(1)
		go func(p *DNSProvider) {
			defer wg.Done()
			latency, success, testsDone := testProvider(DNSProvider{Name: p.Name, IP: p.IP}, func() {
				testsCompleted++
				ui.progress = float32(testsCompleted) / float32(totalTests)
				ui.window.Invalidate()
			}, ui.testsPerDomain)
			resultsChan <- TestResult{
				Provider:    DNSProvider{Name: p.Name, IP: p.IP},
				Latency:    latency,
				Success:    success,
				TestsDone:  testsDone,
				TotalTests: len(testDomains) * ui.testsPerDomain,
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

func testProvider(provider DNSProvider, onProgress func(), testsPerDomain int) (time.Duration, bool, int) {
	var totalLatency time.Duration
	var successfulTests int

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: timeout}
			return d.DialContext(ctx, "udp", provider.IP+":53")
		},
	}

	for _, domain := range testDomains {
		for i := 0; i < testsPerDomain; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			start := time.Now()
			_, err := resolver.LookupHost(ctx, domain)
			latency := time.Since(start)
			cancel()

			if err == nil && latency < timeout {
				totalLatency += latency
				successfulTests++
			}
			onProgress()
		}
	}

	if successfulTests == 0 {
		return timeout, false, len(testDomains) * testsPerDomain
	}
	return totalLatency / time.Duration(successfulTests), true, successfulTests
} 